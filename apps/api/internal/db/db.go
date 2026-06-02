package db

import (
	"context"
	"fmt"
	"time"

	"school-oj/apps/api/internal/config"
	"school-oj/apps/api/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(ctx context.Context, cfg config.Config) (*gorm.DB, error) {
	var lastErr error
	for i := 0; i < 30; i++ {
		gdb, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
		if err == nil {
			sqlDB, err := gdb.DB()
			if err != nil {
				return nil, err
			}
			sqlDB.SetMaxOpenConns(30)
			sqlDB.SetMaxIdleConns(10)
			sqlDB.SetConnMaxLifetime(30 * time.Minute)
			if pingErr := sqlDB.PingContext(ctx); pingErr == nil {
				return gdb, nil
			} else {
				lastErr = pingErr
			}
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return nil, fmt.Errorf("connect database: %w", lastErr)
}

func AutoMigrate(gdb *gorm.DB) error {
	if err := cleanupOrphanProblemLinks(gdb); err != nil {
		return err
	}
	return gdb.AutoMigrate(models.AllModels()...)
}

func cleanupOrphanProblemLinks(gdb *gorm.DB) error {
	statements := map[string]string{
		"assignment_problems": `
DELETE FROM assignment_problems
WHERE NOT EXISTS (
  SELECT 1 FROM problems WHERE problems.id = assignment_problems.problem_id
)`,
		"exam_problems": `
DELETE FROM exam_problems
WHERE NOT EXISTS (
  SELECT 1 FROM problems WHERE problems.id = exam_problems.problem_id
)`,
		"prepared_problems": `
DELETE FROM prepared_problems
WHERE NOT EXISTS (
  SELECT 1 FROM problems WHERE problems.id = prepared_problems.problem_id
)`,
	}
	for table, sql := range statements {
		if !gdb.Migrator().HasTable(table) || !gdb.Migrator().HasTable("problems") {
			continue
		}
		if err := gdb.Exec(sql).Error; err != nil {
			return err
		}
	}
	return nil
}
