package streams

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"school-oj/apps/worker/internal/config"
	"school-oj/apps/worker/internal/models"
	"school-oj/apps/worker/internal/runner"

	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Consumer struct {
	DB     *gorm.DB
	Redis  *redis.Client
	MinIO  *minio.Client
	Cfg    config.Config
	Runner runner.DockerRunner
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
				if err := c.handle(ctx, msg); err != nil {
					log.Printf("submission message %s failed: %v", msg.ID, err)
				}
				if err := c.Redis.XAck(ctx, c.Cfg.Stream, c.Cfg.Group, msg.ID).Err(); err != nil {
					log.Printf("xack %s failed: %v", msg.ID, err)
				}
			}
		}
	}
}

func (c Consumer) handle(ctx context.Context, msg redis.XMessage) error {
	raw, ok := msg.Values["submission_id"]
	if !ok {
		return fmt.Errorf("submission_id missing")
	}
	id64, err := strconv.ParseUint(fmt.Sprint(raw), 10, 64)
	if err != nil {
		return err
	}
	subID := uint(id64)
	var sub models.Submission
	if err := c.DB.Preload("Problem").First(&sub, subID).Error; err != nil {
		return err
	}
	if sub.Status != models.StatusQueued {
		return nil
	}
	c.DB.Model(&sub).Updates(map[string]any{"status": models.StatusRunning, "message": "judging"})
	obj, err := c.MinIO.GetObject(ctx, c.Cfg.MinIOBucket, sub.Problem.PackageObject, minio.GetObjectOptions{})
	if err != nil {
		c.markSystemError(&sub, err)
		return err
	}
	body, err := io.ReadAll(io.LimitReader(obj, 128<<20))
	_ = obj.Close()
	if err != nil {
		c.markSystemError(&sub, err)
		return err
	}
	pkg, err := runner.ParsePackage(body)
	if err != nil {
		c.markSystemError(&sub, err)
		return err
	}
	result := c.Runner.Judge(ctx, runner.JudgeRequest{
		SubmissionID: sub.ID,
		Language:     sub.Language,
		SourceCode:   sub.SourceCode,
		Problem:      sub.Problem,
		Package:      pkg,
	})
	c.DB.Where("submission_id = ?", sub.ID).Delete(&models.SubmissionResult{})
	for _, item := range result.Cases {
		c.DB.Create(&models.SubmissionResult{
			SubmissionID: sub.ID,
			CaseName:     item.Name,
			Status:       item.Status,
			TimeMS:       item.TimeMS,
			MemoryKB:     item.MemoryKB,
			Message:      item.Message,
		})
	}
	c.DB.Model(&sub).Updates(map[string]any{
		"status":    result.Status,
		"score":     result.Score,
		"time_ms":   result.TimeMS,
		"memory_kb": result.MemoryKB,
		"message":   result.Message,
		"trace":     result.Trace,
	})
	c.updateProgress(&sub, result.Status)
	return nil
}

func (c Consumer) markSystemError(sub *models.Submission, err error) {
	c.DB.Model(sub).Updates(map[string]any{"status": models.StatusSystemError, "message": err.Error()})
	c.updateProgress(sub, models.StatusSystemError)
}

func (c Consumer) updateProgress(sub *models.Submission, status models.SubmissionStatus) {
	now := time.Now()
	var progress models.ProblemProgress
	err := c.DB.Where("user_id = ? AND problem_id = ?", sub.UserID, sub.ProblemID).First(&progress).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		progress = models.ProblemProgress{
			UserID:        sub.UserID,
			ProblemID:     sub.ProblemID,
			Status:        models.ProgressUnattempted,
			LastSubmitted: &now,
		}
		if createErr := c.DB.Create(&progress).Error; createErr != nil {
			log.Printf("create progress for submission %d failed: %v", sub.ID, createErr)
			return
		}
	} else if err != nil {
		log.Printf("load progress for submission %d failed: %v", sub.ID, err)
		return
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
	if err := c.DB.Model(&progress).Updates(updates).Error; err != nil {
		log.Printf("update progress for submission %d failed: %v", sub.ID, err)
	}
}
