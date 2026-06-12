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
	if err != nil && isRetryable(err) && c.messageRetryCount(msg) < c.maxRetries() {
		retryID, retryErr := c.requeue(ctx, msg)
		if retryErr != nil {
			log.Printf("requeue submission message %s failed: %v", msg.ID, retryErr)
		} else {
			log.Printf("requeued submission message %s as %s", msg.ID, retryID)
		}
	} else if err != nil && isRetryable(err) {
		c.markMessageSystemError(msg, err)
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
	_ = c.DB.Model(&models.Submission{}).Where("id = ?", subID).Updates(map[string]any{
		"status":  models.StatusQueued,
		"message": fmt.Sprintf("retrying judge job (%d/%d)", nextRetry, c.maxRetries()),
	}).Error
	return c.Redis.XAdd(ctx, &redis.XAddArgs{
		Stream: c.Cfg.Stream,
		Values: map[string]any{
			"submission_id": subID,
			"retry_count":   nextRetry,
		},
	}).Result()
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
	if err := c.DB.Preload("Problem").First(&sub, subID).Error; err != nil {
		return err
	}
	if sub.Status != models.StatusQueued {
		return nil
	}
	c.DB.Model(&sub).Updates(map[string]any{"status": models.StatusRunning, "message": "judging"})
	obj, err := c.MinIO.GetObject(ctx, c.Cfg.MinIOBucket, sub.Problem.PackageObject, minio.GetObjectOptions{})
	if err != nil {
		_ = c.DB.Model(&sub).Updates(map[string]any{"status": models.StatusQueued, "message": "waiting for problem package"})
		return retryable(err)
	}
	body, err := readLimited(obj, 128<<20, "problem package")
	_ = obj.Close()
	if err != nil {
		_ = c.DB.Model(&sub).Updates(map[string]any{"status": models.StatusQueued, "message": "waiting for problem package"})
		return retryable(err)
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
	if result.Status == models.StatusSystemError && isRetryableSystemMessage(result.Message) {
		_ = c.DB.Model(&sub).Updates(map[string]any{"status": models.StatusQueued, "message": result.Message})
		return retryable(errors.New(result.Message))
	}
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
	text := strings.ToLower(message)
	markers := []string{
		"docker sandbox image or daemon is not ready",
		"cannot connect to the docker daemon",
		"unable to find image",
		"no such image",
		"failed to resolve",
		"toomanyrequests",
	}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func (c Consumer) updateProgress(sub *models.Submission, status models.SubmissionStatus) {
	now := time.Now()
	var progress models.ProblemProgress
	err := c.DB.Where("user_id = ? AND problem_id = ?", sub.UserID, sub.ProblemID).
		Attrs(models.ProblemProgress{
			UserID:        sub.UserID,
			ProblemID:     sub.ProblemID,
			Status:        models.ProgressUnattempted,
			LastSubmitted: &now,
		}).
		FirstOrCreate(&progress).Error
	if err != nil {
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
