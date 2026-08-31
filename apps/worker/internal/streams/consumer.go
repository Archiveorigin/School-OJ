package streams

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"school-oj/apps/worker/internal/config"
	"school-oj/apps/worker/internal/models"
	"school-oj/apps/worker/internal/runner"

	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const submissionChannelPrefix = "oj.submission."

var (
	errJudgeLocked      = errors.New("submission is already being judged")
	errStaleJudgeResult = errors.New("submission no longer accepts this judge result")
	renewJudgeLock      = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`)
	releaseJudgeLock = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)
)

type Consumer struct {
	DB     *gorm.DB
	Redis  *redis.Client
	MinIO  *minio.Client
	Cfg    config.Config
	Runner runner.DockerRunner
}

type retryableHandleError struct {
	err error
}

func (e retryableHandleError) Error() string {
	return e.err.Error()
}

func (e retryableHandleError) Unwrap() error {
	return e.err
}

func retryable(err error) error {
	if err == nil {
		return nil
	}
	return retryableHandleError{err: err}
}

func isRetryable(err error) bool {
	var retryErr retryableHandleError
	return errors.As(err, &retryErr)
}

func Redis(ctx context.Context, cfg config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	var lastErr error
	for i := 0; i < 30; i++ {
		if err := client.Ping(ctx).Err(); err == nil {
			return client, nil
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return nil, fmt.Errorf("connect redis: %w", lastErr)
}

func (c Consumer) EnsureGroup(ctx context.Context) error {
	err := c.Redis.XGroupCreateMkStream(ctx, c.Cfg.Stream, c.Cfg.Group, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func (c Consumer) Run(ctx context.Context) error {
	if err := c.EnsureGroup(ctx); err != nil {
		return err
	}
	for {
		if err := c.reclaimPending(ctx); err != nil && !errors.Is(err, redis.Nil) {
			log.Printf("reclaim pending submissions failed: %v", err)
		}
		streams, err := c.Redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.Cfg.Group,
			Consumer: c.Cfg.Consumer,
			Streams:  []string{c.Cfg.Stream, ">"},
			Count:    1,
			Block:    5 * time.Second,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			return err
		}
		for _, stream := range streams {
			for _, msg := range stream.Messages {
				c.processMessage(ctx, msg)
			}
		}
	}
}

func (c Consumer) reclaimPending(ctx context.Context) error {
	idle := c.retryIdle()
	pending, err := c.Redis.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: c.Cfg.Stream,
		Group:  c.Cfg.Group,
		Start:  "-",
		End:    "+",
		Count:  10,
	}).Result()
	if err != nil {
		return err
	}
	for _, item := range pending {
		if item.Idle < idle {
			continue
		}
		messages, err := c.Redis.XClaim(ctx, &redis.XClaimArgs{
			Stream:   c.Cfg.Stream,
			Group:    c.Cfg.Group,
			Consumer: c.Cfg.Consumer,
			MinIdle:  idle,
			Messages: []string{item.ID},
		}).Result()
		if err != nil {
			log.Printf("xclaim %s failed: %v", item.ID, err)
			continue
		}
		for _, msg := range messages {
			c.processMessage(ctx, msg)
		}
	}
	return nil
}

func (c Consumer) processMessage(ctx context.Context, msg redis.XMessage) {
	err := c.handle(ctx, msg)
	if err != nil {
		log.Printf("submission message %s failed: %v", msg.ID, err)
	}
	if errors.Is(err, errJudgeLocked) {
		// Leave the message pending. The active judge owns the lease and will ACK
		// when it finishes; if it crashed, this message can be reclaimed later.
		return
	}
	ack := true
	if err != nil && isRetryable(err) && c.messageRetryCount(msg) < c.maxRetries() {
		retryID, retryErr := c.requeue(ctx, msg)
		if retryErr != nil {
			log.Printf("requeue submission message %s failed: %v", msg.ID, retryErr)
			ack = false
		} else {
			log.Printf("requeued submission message %s as %s", msg.ID, retryID)
		}
	} else if err != nil && isRetryable(err) {
		c.markMessageSystemError(msg, err)
	}
	if !ack {
		return
	}
	if err := c.Redis.XAck(ctx, c.Cfg.Stream, c.Cfg.Group, msg.ID).Err(); err != nil {
		log.Printf("xack %s failed: %v", msg.ID, err)
	}
}

func (c Consumer) requeue(ctx context.Context, msg redis.XMessage) (string, error) {
	subID, err := submissionIDFromMessage(msg)
	if err != nil {
		return "", err
	}
	nextRetry := c.messageRetryCount(msg) + 1
	if err := c.DB.Model(&models.Submission{}).Where("id = ? AND status IN ?", subID, []models.SubmissionStatus{
		models.StatusQueued,
		models.StatusRunning,
	}).Updates(map[string]any{
		"status":  models.StatusQueued,
		"message": fmt.Sprintf("retrying judge job (%d/%d)", nextRetry, c.maxRetries()),
	}).Error; err != nil {
		return "", err
	}
	retryID, err := c.Redis.XAdd(ctx, &redis.XAddArgs{
		Stream: c.Cfg.Stream,
		Values: map[string]any{
			"submission_id": subID,
			"retry_count":   nextRetry,
		},
	}).Result()
	if err != nil {
		return "", err
	}
	c.publishSubmission(subID)
	return retryID, nil
}

func (c Consumer) markMessageSystemError(msg redis.XMessage, err error) {
	subID, parseErr := submissionIDFromMessage(msg)
	if parseErr != nil {
		return
	}
	var sub models.Submission
	if loadErr := c.DB.First(&sub, subID).Error; loadErr != nil {
		return
	}
	c.markSystemError(&sub, err)
}

func (c Consumer) handle(ctx context.Context, msg redis.XMessage) error {
	subID, err := submissionIDFromMessage(msg)
	if err != nil {
		return err
	}
	var sub models.Submission
	if err := c.DB.Preload("Problem").Preload("ProblemVersion").First(&sub, subID).Error; err != nil {
		return retryable(err)
	}
	if sub.Status != models.StatusQueued && sub.Status != models.StatusRunning {
		return nil
	}

	lease, acquired, err := c.acquireJudgeLease(ctx, sub.ID, msg.ID)
	if err != nil {
		return retryable(err)
	}
	if !acquired {
		return errJudgeLocked
	}
	defer lease.Release()

	claim := c.DB.Model(&models.Submission{}).
		Where("id = ? AND status IN ?", sub.ID, []models.SubmissionStatus{models.StatusQueued, models.StatusRunning}).
		Updates(map[string]any{"status": models.StatusRunning, "message": "judging"})
	if claim.Error != nil {
		return retryable(claim.Error)
	}
	if claim.RowsAffected == 0 {
		return nil
	}
	c.publishSubmission(sub.ID)

	judgeProblem := problemSnapshotForSubmission(sub)
	obj, err := c.MinIO.GetObject(ctx, c.Cfg.MinIOBucket, judgeProblem.PackageObject, minio.GetObjectOptions{})
	if err != nil {
		_ = c.DB.Model(&sub).Where("status = ?", models.StatusRunning).Updates(map[string]any{"status": models.StatusQueued, "message": "waiting for problem package"}).Error
		c.publishSubmission(sub.ID)
		return retryable(err)
	}
	body, err := readLimited(obj, 128<<20, "problem package")
	_ = obj.Close()
	if err != nil {
		_ = c.DB.Model(&sub).Where("status = ?", models.StatusRunning).Updates(map[string]any{"status": models.StatusQueued, "message": "waiting for problem package"}).Error
		c.publishSubmission(sub.ID)
		return retryable(err)
	}
	pkg, err := runner.ParsePackage(body)
	if err != nil {
		c.markSystemError(&sub, err)
		return err
	}

	judgeCtx, cancelJudge := context.WithCancel(ctx)
	defer cancelJudge()
	go func() {
		select {
		case <-lease.Lost():
			cancelJudge()
		case <-judgeCtx.Done():
		}
	}()
	result := c.Runner.Judge(judgeCtx, runner.JudgeRequest{
		SubmissionID: sub.ID,
		Language:     sub.Language,
		SourceCode:   sub.SourceCode,
		Problem:      judgeProblem,
		Package:      pkg,
	})
	if lease.IsLost() {
		return retryable(errors.New("judge lease was lost while the sandbox was running"))
	}
	if result.Status == models.StatusSystemError && isRetryableSystemMessage(result.Message) {
		_ = c.DB.Model(&sub).Where("status = ?", models.StatusRunning).Updates(map[string]any{"status": models.StatusQueued, "message": result.Message}).Error
		c.publishSubmission(sub.ID)
		return retryable(errors.New(result.Message))
	}
	if err := c.persistJudgeResult(ctx, &sub, result); err != nil {
		if errors.Is(err, errStaleJudgeResult) {
			return nil
		}
		return retryable(err)
	}
	c.publishSubmission(sub.ID)
	return nil
}

func problemSnapshotForSubmission(sub models.Submission) models.Problem {
	problem := sub.Problem
	if sub.ProblemVersion.ID == 0 {
		return problem
	}
	problem.Title = sub.ProblemVersion.Title
	problem.Statement = sub.ProblemVersion.Statement
	problem.Tags = sub.ProblemVersion.Tags
	problem.TimeLimitMS = sub.ProblemVersion.TimeLimitMS
	problem.MemoryLimitMB = sub.ProblemVersion.MemoryLimitMB
	problem.OutputLimitKB = sub.ProblemVersion.OutputLimitKB
	problem.PackageObject = sub.ProblemVersion.PackageObject
	problem.PackageChecksum = sub.ProblemVersion.PackageChecksum
	problem.Manifest = sub.ProblemVersion.Manifest
	return problem
}

func (c Consumer) markSystemError(sub *models.Submission, err error) {
	if updateErr := c.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Submission{}).
			Where("id = ? AND status IN ?", sub.ID, []models.SubmissionStatus{models.StatusQueued, models.StatusRunning}).
			Updates(map[string]any{"status": models.StatusSystemError, "message": err.Error()})
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}
		return updateProgress(tx, sub, models.StatusSystemError)
	}); updateErr != nil {
		log.Printf("mark submission %d as system error failed: %v", sub.ID, updateErr)
		return
	}
	c.publishSubmission(sub.ID)
}

func (c Consumer) persistJudgeResult(ctx context.Context, sub *models.Submission, result runner.JudgeResult) error {
	return c.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		update := tx.Model(&models.Submission{}).
			Where("id = ? AND status = ?", sub.ID, models.StatusRunning).
			Updates(map[string]any{
				"status":    result.Status,
				"score":     result.Score,
				"time_ms":   result.TimeMS,
				"memory_kb": result.MemoryKB,
				"message":   result.Message,
				"trace":     result.Trace,
			})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected == 0 {
			return errStaleJudgeResult
		}
		if err := tx.Where("submission_id = ?", sub.ID).Delete(&models.SubmissionResult{}).Error; err != nil {
			return err
		}
		if len(result.Cases) > 0 {
			items := make([]models.SubmissionResult, 0, len(result.Cases))
			for _, item := range result.Cases {
				items = append(items, models.SubmissionResult{
					SubmissionID: sub.ID,
					CaseName:     item.Name,
					Status:       item.Status,
					TimeMS:       item.TimeMS,
					MemoryKB:     item.MemoryKB,
					Message:      item.Message,
				})
			}
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		return updateProgress(tx, sub, result.Status)
	})
}

func (c Consumer) acquireJudgeLease(ctx context.Context, submissionID uint, messageID string) (*judgeLease, bool, error) {
	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, false, err
	}
	ttl := c.judgeLeaseTTL()
	lease := &judgeLease{
		client: c.Redis,
		key:    fmt.Sprintf("oj:submission:lease:%d", submissionID),
		token:  c.Cfg.Consumer + ":" + messageID + ":" + hex.EncodeToString(tokenBytes),
		ttl:    ttl,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
		lost:   make(chan struct{}),
	}
	acquired, err := c.Redis.SetNX(ctx, lease.key, lease.token, ttl).Result()
	if err != nil || !acquired {
		return nil, acquired, err
	}
	go lease.renew()
	return lease, true, nil
}

func (c Consumer) judgeLeaseTTL() time.Duration {
	ttl := c.retryIdle() * 2
	if ttl < 2*time.Minute {
		return 2 * time.Minute
	}
	return ttl
}

func (l *judgeLease) renew() {
	defer close(l.done)
	interval := l.ttl / 3
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-l.stop:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			renewed, err := renewJudgeLock.Run(ctx, l.client, []string{l.key}, l.token, l.ttl.Milliseconds()).Int64()
			cancel()
			if err != nil {
				log.Printf("renew judge lease %s failed: %v", l.key, err)
				continue
			}
			if renewed == 0 {
				if l.ended.CompareAndSwap(false, true) {
					close(l.lost)
				}
				return
			}
		}
	}
}

func (l *judgeLease) Lost() <-chan struct{} {
	return l.lost
}

func (l *judgeLease) IsLost() bool {
	return l.ended.Load()
}

func (l *judgeLease) Release() {
	select {
	case <-l.done:
	default:
		close(l.stop)
		<-l.done
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := releaseJudgeLock.Run(ctx, l.client, []string{l.key}, l.token).Result(); err != nil {
		log.Printf("release judge lease %s failed: %v", l.key, err)
	}
}

func (c Consumer) publishSubmission(submissionID uint) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	channel := fmt.Sprintf("%s%d", submissionChannelPrefix, submissionID)
	if err := c.Redis.Publish(ctx, channel, submissionID).Err(); err != nil {
		log.Printf("publish submission %d status failed: %v", submissionID, err)
	}
}

func submissionIDFromMessage(msg redis.XMessage) (uint, error) {
	raw, ok := msg.Values["submission_id"]
	if !ok {
		return 0, fmt.Errorf("submission_id missing")
	}
	id64, err := strconv.ParseUint(fmt.Sprint(raw), 10, 64)
	if err != nil || id64 == 0 {
		return 0, fmt.Errorf("invalid submission_id: %v", raw)
	}
	return uint(id64), nil
}

func (c Consumer) messageRetryCount(msg redis.XMessage) int {
	raw, ok := msg.Values["retry_count"]
	if !ok {
		return 0
	}
	retries, err := strconv.Atoi(fmt.Sprint(raw))
	if err != nil || retries < 0 {
		return 0
	}
	return retries
}

func (c Consumer) maxRetries() int {
	if c.Cfg.MaxRetries <= 0 {
		return 0
	}
	return c.Cfg.MaxRetries
}

func (c Consumer) retryIdle() time.Duration {
	if c.Cfg.RetryIdleSeconds <= 0 {
		return 60 * time.Second
	}
	return time.Duration(c.Cfg.RetryIdleSeconds) * time.Second
}

func readLimited(r io.Reader, maxBytes int64, label string) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > maxBytes {
		return nil, fmt.Errorf("%s is too large", label)
	}
	return body, nil
}

func isRetryableSystemMessage(message string) bool {
	return runner.IsInfrastructureError(message)
}

type judgeLease struct {
	client *redis.Client
	key    string
	token  string
	ttl    time.Duration
	stop   chan struct{}
	done   chan struct{}
	lost   chan struct{}
	ended  atomic.Bool
}

func updateProgress(database *gorm.DB, sub *models.Submission, status models.SubmissionStatus) error {
	now := time.Now()
	progress := models.ProblemProgress{
		UserID:        sub.UserID,
		ProblemID:     sub.ProblemID,
		Status:        models.ProgressUnattempted,
		LastSubmitted: &now,
	}
	if err := database.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "problem_id"}},
		DoNothing: true,
	}).Create(&progress).Error; err != nil {
		return fmt.Errorf("initialize progress for submission %d: %w", sub.ID, err)
	}
	if err := database.Where("user_id = ? AND problem_id = ?", sub.UserID, sub.ProblemID).
		First(&progress).Error; err != nil {
		return fmt.Errorf("load progress for submission %d: %w", sub.ID, err)
	}

	updates := map[string]any{"last_submitted": now}
	if status == models.StatusAccepted {
		if progress.Status != models.ProgressAccepted {
			updates["status"] = models.ProgressAccepted
			updates["points"] = 1
			updates["points_awarded"] = true
			updates["first_accepted"] = now
		}
	} else if progress.Status == models.ProgressUnattempted {
		updates["status"] = models.ProgressAttempted
	}
	if err := database.Model(&progress).Updates(updates).Error; err != nil {
		return fmt.Errorf("update progress for submission %d: %w", sub.ID, err)
	}
	return nil
}
