package db

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"school-oj/apps/api/internal/models"
)

func TestFreshSQLInstallationAndRestartPostgres(t *testing.T) {
	database := migrationTestDB(t)
	for i := 0; i < 2; i++ {
		if err := Migrate(database); err != nil {
			t.Fatal(err)
		}
	}
	user := models.User{Email: "fresh@example.test", Name: "Fresh", Role: models.RoleStudent, PasswordHash: "test"}
	if err := database.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	user.ID = 0
	if err := database.Create(&user).Error; err == nil {
		t.Fatal("email uniqueness lost")
	}
}

func TestMigrationLockReleasedAfterCancellationPostgres(t *testing.T) {
	database := migrationTestDB(t)
	pool, _ := database.DB()
	pool.SetMaxOpenConns(1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := withMigrationLock(database.WithContext(ctx), func(conn *gorm.DB) error { cancel(); return ctx.Err() })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled migration error = %v", err)
	}
	var held bool
	if err := database.Raw("SELECT EXISTS(SELECT 1 FROM pg_locks WHERE pid = pg_backend_pid() AND locktype = 'advisory')").Scan(&held).Error; err != nil {
		t.Fatal(err)
	}
	if held {
		t.Fatal("cancelled request left an advisory lock in the pool")
	}
}

func migrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("OJ_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires isolated PostgreSQL")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil || cfg.Host != "127.0.0.1" || cfg.Database != "oj_exam_security_test" {
		t.Fatal("unsafe test database")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("migration_qa_%d", time.Now().UnixNano())
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
	return database
}

func TestMigrationFailureRollbackAndRetryPostgres(t *testing.T) {
	database := migrationTestDB(t)
	if err := ensureMigrationLedger(database); err != nil {
		t.Fatal(err)
	}
	file := func(v int, sql string) migrationFile {
		return migrationFile{Version: v, Name: fmt.Sprintf("%03d_test.sql", v), SQL: sql, Checksum: fmt.Sprintf("%x", sha256.Sum256([]byte(sql)))}
	}
	first := file(1, "CREATE TABLE durable_data(id INTEGER PRIMARY KEY); INSERT INTO durable_data VALUES(7)")
	failing := file(2, "CREATE TABLE transient_data(id INTEGER); INSERT INTO missing_table VALUES(1)")
	if err := applyVersionedMigrations(database, []migrationFile{first, failing}); err == nil {
		t.Fatal("expected injected SQL failure")
	}
	if database.Migrator().HasTable("transient_data") {
		t.Fatal("DDL did not roll back")
	}
	var count int64
	database.Table("schema_migrations").Count(&count)
	if count != 1 {
		t.Fatalf("failed migration entered ledger: %d", count)
	}
	database.Table("durable_data").Where("id = 7").Count(&count)
	if count != 1 {
		t.Fatal("previous committed data was lost")
	}
	corrected := file(2, "CREATE TABLE transient_data(id INTEGER)")
	if err := applyVersionedMigrations(database, []migrationFile{first, corrected}); err != nil {
		t.Fatal(err)
	}
	first.SQL += "; SELECT 1"
	first.Checksum = "tampered"
	if err := applyVersionedMigrations(database, []migrationFile{first}); err == nil || !strings.Contains(err.Error(), "changed after") {
		t.Fatalf("checksum guard: %v", err)
	}
}

func TestMigrationLockUsesOneSessionAndSerializesPostgres(t *testing.T) {
	database := migrationTestDB(t)
	var wg sync.WaitGroup
	errors := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errors <- withMigrationLock(database, func(conn *gorm.DB) error {
				if err := ensureMigrationLedger(conn); err != nil {
					return err
				}
				var held bool
				if err := conn.Raw("SELECT EXISTS(SELECT 1 FROM pg_locks WHERE pid = pg_backend_pid() AND locktype = 'advisory' AND granted)").Scan(&held).Error; err != nil {
					return err
				}
				if !held {
					return fmt.Errorf("migration used a different PostgreSQL session")
				}
				return applyVersionedMigrations(conn, []migrationFile{{Version: 1, Name: "001_serial.sql", Checksum: "serial", SQL: "CREATE TABLE only_once(id INTEGER)"}})
			})
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	database.Table("schema_migrations").Count(&count)
	if count != 1 {
		t.Fatalf("ledger count = %d", count)
	}
	// Other packages may concurrently migrate their own isolated schemas, so
	// verify release directly on each connection instead of counting their locks.
	if err := database.Connection(func(conn *gorm.DB) error {
		var heldHere bool
		conn.Raw("SELECT EXISTS(SELECT 1 FROM pg_locks WHERE pid = pg_backend_pid() AND locktype = 'advisory')").Scan(&heldHere)
		if heldHere {
			return fmt.Errorf("advisory lock leaked into connection pool")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
