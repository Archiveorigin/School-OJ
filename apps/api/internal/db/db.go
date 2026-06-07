package db

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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
	if err := gdb.AutoMigrate(models.AllModels()...); err != nil {
		return err
	}
	if err := migrateProblemSlugUniqueness(gdb); err != nil {
		return err
	}
	if err := backfillProblemDisplayCodes(gdb); err != nil {
		return err
	}
	return backfillClassJoinCodes(gdb)
}

func migrateProblemSlugUniqueness(gdb *gorm.DB) error {
	if !gdb.Migrator().HasTable("problems") {
		return nil
	}
	statements := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_problems_slug_active ON problems(slug) WHERE deleted_at IS NULL`,
		`ALTER TABLE problems DROP CONSTRAINT IF EXISTS problems_slug_key`,
		`ALTER TABLE problems DROP CONSTRAINT IF EXISTS idx_problems_slug`,
		`DROP INDEX IF EXISTS idx_problems_slug`,
	}
	for _, statement := range statements {
		if err := gdb.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func backfillProblemDisplayCodes(gdb *gorm.DB) error {
	var problems []models.Problem
	if err := gdb.Order("id asc").Find(&problems).Error; err != nil {
		return err
	}
	maxIndex := 0
	for _, problem := range problems {
		if index := parseProblemDisplayCode(problem.DisplayCode); index > maxIndex {
			maxIndex = index
		}
	}
	for _, problem := range problems {
		if strings.TrimSpace(problem.DisplayCode) != "" {
			continue
		}
		maxIndex += 1
		if err := gdb.Model(&models.Problem{}).Where("id = ?", problem.ID).Update("display_code", models.FormatProblemDisplayCode(maxIndex)).Error; err != nil {
			return err
		}
	}
	return nil
}

func backfillClassJoinCodes(gdb *gorm.DB) error {
	if !gdb.Migrator().HasTable("classes") {
		return nil
	}
	var classes []models.Class
	if err := gdb.Where("join_code IS NULL OR join_code = ''").Order("id asc").Find(&classes).Error; err != nil {
		return err
	}
	for _, class := range classes {
		if err := gdb.Model(&models.Class{}).Where("id = ?", class.ID).Update("join_code", models.FormatClassJoinCode(class.ID)).Error; err != nil {
			return err
		}
	}
	return nil
}

func parseProblemDisplayCode(value string) int {
	value = strings.TrimSpace(strings.ToUpper(value))
	if !strings.HasPrefix(value, "T") {
		return 0
	}
	index, err := strconv.Atoi(strings.TrimPrefix(value, "T"))
	if err != nil || index < 0 {
		return 0
	}
	return index
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
