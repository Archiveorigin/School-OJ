package streams

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"school-oj/apps/worker/internal/config"
	"school-oj/apps/worker/internal/models"
	"school-oj/apps/worker/internal/runner"
)

func recoveryRedis(t *testing.T) *redis.Client {
	t.Helper()
	addr := os.Getenv("OJ_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("requires isolated Redis")
	}
	if addr != "127.0.0.1:56379" {
		t.Fatal("unsafe Redis address")
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr, DB: 14})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

func TestCrashedWorkerLeaseCanBeReclaimedRedis(t *testing.T) {
	rdb := recoveryRedis(t)
	ctx := context.Background()
	c := Consumer{Redis: rdb}
	id := uint(time.Now().UnixNano())
	old, ok, err := c.acquireJudgeLease(ctx, id, "crashed-worker")
	if err != nil || !ok {
		t.Fatalf("acquire: %v %v", ok, err)
	}
	t.Cleanup(old.Release)
	if _, acquired, err := c.acquireJudgeLease(ctx, id, "duplicate-worker"); err != nil || acquired {
		t.Fatalf("parallel judge acquired live lease: %v %v", acquired, err)
	}
	// Stop the heartbeat as process death would; advance the Redis lease deadline
	// without waiting the production two-minute safety interval.
	close(old.stop)
	<-old.done
	if err := rdb.PExpire(ctx, old.key, time.Millisecond).Err(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for rdb.Exists(ctx, old.key).Val() != 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	replacement, ok, err := c.acquireJudgeLease(ctx, id, "replacement-worker")
	if err != nil || !ok {
		t.Fatalf("recover expired lease: %v %v", ok, err)
	}
	t.Cleanup(replacement.Release)
	old.Release()
	if token := rdb.Get(ctx, old.key).Val(); token != replacement.token {
		t.Fatal("stale worker released replacement worker's lease")
	}
}

func recoveryDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("OJ_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires isolated PostgreSQL")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil || cfg.Host != "127.0.0.1" || cfg.Database != "oj_exam_security_test" {
		t.Fatal("unsafe database")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("worker_recovery_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	database, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool, _ := database.DB()
		_ = pool.Close()
		_ = admin.Exec("DROP SCHEMA " + schema + " CASCADE").Error
		pool, _ = admin.DB()
		_ = pool.Close()
	})
	if err := database.AutoMigrate(&models.Problem{}, &models.ProblemVersion{}, &models.Submission{}); err != nil {
		t.Fatal(err)
	}
	return database
}

func TestCrashedWorkerPendingMessageRecoveryPostgresRedis(t *testing.T) {
	database, rdb := recoveryDB(t), recoveryRedis(t)
	ctx := context.Background()
	c := Consumer{DB: database, Redis: rdb, Cfg: config.Config{Stream: fmt.Sprintf("qa.worker.%d", time.Now().UnixNano()), Group: "judge", Consumer: "replacement", RetryIdleSeconds: 1, MaxRetries: 3}}
	t.Cleanup(func() { rdb.Del(ctx, c.Cfg.Stream) })
	if err := c.EnsureGroup(ctx); err != nil {
		t.Fatal(err)
	}
	problem := models.Problem{Title: "current"}
	if err := database.Create(&problem).Error; err != nil {
		t.Fatal(err)
	}
	version := models.ProblemVersion{ProblemID: problem.ID, Version: 1, Title: "pinned"}
	if err := database.Create(&version).Error; err != nil {
		t.Fatal(err)
	}
	sub := models.Submission{ProblemID: problem.ID, ProblemVersionID: version.ID, Status: models.StatusAccepted, Score: 100, SourceCode: "print(3)"}
	if err := database.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	id, err := rdb.XAdd(ctx, &redis.XAddArgs{Stream: c.Cfg.Stream, Values: map[string]any{"submission_id": sub.ID}}).Result()
	if err != nil {
		t.Fatal(err)
	}
	items, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{Group: c.Cfg.Group, Consumer: "crashed", Streams: []string{c.Cfg.Stream, ">"}, Count: 1, Block: -1}).Result()
	if err != nil {
		t.Fatal(err)
	}
	msg := items[0].Messages[0]
	// Simulate a crash after the result committed but before ACK.
	if err := rdb.Do(ctx, "XCLAIM", c.Cfg.Stream, c.Cfg.Group, "crashed", "0", id, "IDLE", "2000").Err(); err != nil {
		t.Fatal(err)
	}
	if err := c.reclaimPending(ctx); err != nil {
		t.Fatal(err)
	}
	if pending := rdb.XPending(ctx, c.Cfg.Stream, c.Cfg.Group).Val(); pending.Count != 0 {
		t.Fatal("completed message was not acknowledged on restart")
	}
	database.First(&sub, sub.ID)
	if sub.Status != models.StatusAccepted || sub.Score != 100 || sub.ProblemVersionID != version.ID {
		t.Fatal("recovery changed committed result or version")
	}
	// Requeue cannot ACK the original message when Redis publication is down.
	database.Model(&sub).Update("status", models.StatusQueued)
	id, err = rdb.XAdd(ctx, &redis.XAddArgs{Stream: c.Cfg.Stream, Values: msg.Values}).Result()
	if err != nil {
		t.Fatal(err)
	}
	items, err = rdb.XReadGroup(ctx, &redis.XReadGroupArgs{Group: c.Cfg.Group, Consumer: "interrupted", Streams: []string{c.Cfg.Stream, ">"}, Count: 1, Block: -1}).Result()
	if err != nil {
		t.Fatal(err)
	}
	offline := redis.NewClient(&redis.Options{Addr: "127.0.0.1:56379", DB: 14})
	_ = offline.Close()
	broken := c
	broken.Redis = offline
	broken.processMessage(ctx, items[0].Messages[0])
	if pending := rdb.XPending(ctx, c.Cfg.Stream, c.Cfg.Group).Val(); pending.Count != 1 {
		t.Fatal("Redis failure lost unacknowledged work")
	}
	// Model a recovered judge committing its result, then acknowledge normally.
	database.Model(&sub).Update("status", models.StatusAccepted)
	c.processMessage(ctx, items[0].Messages[0])
	if pending := rdb.XPending(ctx, c.Cfg.Stream, c.Cfg.Group).Val(); pending.Count != 0 {
		t.Fatal("recovered result remains pending")
	}
	// Exhausted retry budget is not permission to ACK while PostgreSQL is down.
	// Cancel the SQL transport context while keeping the real Redis connection up.
	if err := database.Model(&sub).Update("status", models.StatusQueued).Error; err != nil {
		t.Fatal(err)
	}
	if err := rdb.XAdd(ctx, &redis.XAddArgs{Stream: c.Cfg.Stream, Values: msg.Values}).Err(); err != nil {
		t.Fatal(err)
	}
	items, err = rdb.XReadGroup(ctx, &redis.XReadGroupArgs{Group: c.Cfg.Group, Consumer: "db-interrupted", Streams: []string{c.Cfg.Stream, ">"}, Count: 1, Block: -1}).Result()
	if err != nil {
		t.Fatal(err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	broken = c
	broken.DB = database.WithContext(cancelled)
	broken.Cfg.MaxRetries = 0
	broken.processMessage(ctx, items[0].Messages[0])
	if pending := rdb.XPending(ctx, c.Cfg.Stream, c.Cfg.Group).Val(); pending.Count != 1 {
		t.Fatal("terminal persistence failure acknowledged unfinished work")
	}
	if err := database.Model(&sub).Update("status", models.StatusAccepted).Error; err != nil {
		t.Fatal(err)
	}
	c.processMessage(ctx, items[0].Messages[0])
	if pending := rdb.XPending(ctx, c.Cfg.Stream, c.Cfg.Group).Val(); pending.Count != 0 {
		t.Fatal("database recovery did not acknowledge the committed result")
	}
}

func TestJudgePhaseMetricsExportedRedis(t *testing.T) {
	rdb := recoveryRedis(t)
	ctx := context.Background()
	if err := rdb.Del(ctx, "oj:metrics:worker").Err(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rdb.Del(context.Background(), "oj:metrics:worker") })
	stream := fmt.Sprintf("oj.test.phase.%d", time.Now().UnixNano())
	c := Consumer{Redis: rdb, Cfg: config.Config{Stream: stream, Group: "phase-test"}}
	if err := c.EnsureGroup(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rdb.Del(context.Background(), stream) })
	timing := runner.JudgeTiming{Compile: 2 * time.Second, CaseWall: 3 * time.Second, Program: time.Second, Cases: 4}
	c.observePhases(100*time.Millisecond, 200*time.Millisecond, timing)
	c.observePhases(100*time.Millisecond, 200*time.Millisecond, timing)

	response := httptest.NewRecorder()
	c.Monitor().ServeHTTP(response, httptest.NewRequest("GET", "/metrics", nil))
	body := response.Body.String()
	for _, want := range []string{
		`oj_judge_phase_seconds_sum{phase="download"} 0.200`,
		`oj_judge_phase_seconds_sum{phase="parse"} 0.400`,
		`oj_judge_phase_seconds_sum{phase="compile"} 4.000`,
		`oj_judge_phase_seconds_sum{phase="case_wall"} 6.000`,
		`oj_judge_phase_seconds_sum{phase="program"} 2.000`,
		"oj_judge_phase_judges_total 2",
		"oj_judge_phase_cases_total 8",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in:\n%s", want, body)
		}
	}
}
