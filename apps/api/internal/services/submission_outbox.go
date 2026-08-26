package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"school-oj/apps/api/internal/streams"

	"github.com/redis/go-redis/v9"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const submissionOutboxVersion = 1

type submissionOutboxEvent struct {
	ID           uint64            `gorm:"column:id"`
	SubmissionID uint              `gorm:"column:submission_id"`
	EventVersion int               `gorm:"column:event_version"`
	Payload      datatypes.JSONMap `gorm:"column:payload"`
}

type SubmissionOutboxDispatcher struct {
	DB           *gorm.DB
	Redis        *redis.Client
	PollInterval time.Duration
	BatchSize    int
	LockTTL      time.Duration
}

func AddSubmissionOutbox(tx *gorm.DB, submissionID uint) error {
	payload, err := json.Marshal(map[string]any{
		"submission_id": submissionID,
		"event_version": submissionOutboxVersion,
	})
	if err != nil {
		return fmt.Errorf("marshal submission outbox payload: %w", err)
	}
	if err := tx.Exec(`
INSERT INTO submission_outbox(submission_id, event_version, payload)
VALUES (?, ?, ?::jsonb)`, submissionID, submissionOutboxVersion, string(payload)).Error; err != nil {
		return fmt.Errorf("create submission outbox event: %w", err)
	}
	return nil
}

func (d SubmissionOutboxDispatcher) Run(ctx context.Context) {
	interval := d.PollInterval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if _, err := d.DispatchOnce(ctx); err != nil && ctx.Err() == nil {
			log.Printf("submission outbox dispatch failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (d SubmissionOutboxDispatcher) DispatchOnce(ctx context.Context) (int, error) {
	if d.DB == nil || d.Redis == nil {
		return 0, fmt.Errorf("submission outbox dispatcher is not configured")
	}
	batchSize := d.BatchSize
	if batchSize <= 0 {
		batchSize = 20
	}
	lockTTL := d.LockTTL
	if lockTTL <= 0 {
		lockTTL = 30 * time.Second
	}

	events, err := d.claim(ctx, batchSize, lockTTL)
	if err != nil {
		return 0, err
	}
	published := 0
	for _, event := range events {
		_, publishErr := d.Redis.XAdd(ctx, &redis.XAddArgs{
			Stream: streams.SubmissionStream,
			Values: map[string]any{
				"submission_id": event.SubmissionID,
				"outbox_id":     event.ID,
				"event_version": event.EventVersion,
			},
		}).Result()
		if publishErr != nil {
			if err := d.releaseWithError(ctx, event.ID, publishErr); err != nil {
				return published, err
			}
			continue
		}
		if err := d.markPublished(ctx, event.ID); err != nil {
			return published, err
		}
		published++
	}
	return published, nil
}

func (d SubmissionOutboxDispatcher) claim(ctx context.Context, batchSize int, lockTTL time.Duration) ([]submissionOutboxEvent, error) {
	var events []submissionOutboxEvent
	lockUntil := time.Now().Add(lockTTL)
	err := d.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return tx.Raw(`
WITH pending AS (
  SELECT id
  FROM submission_outbox
  WHERE published_at IS NULL
    AND available_at <= CURRENT_TIMESTAMP
    AND (locked_until IS NULL OR locked_until < CURRENT_TIMESTAMP)
  ORDER BY id
  FOR UPDATE SKIP LOCKED
  LIMIT ?
)
UPDATE submission_outbox AS event
SET locked_until = ?, attempts = attempts + 1
FROM pending
WHERE event.id = pending.id
RETURNING event.id, event.submission_id, event.event_version, event.payload`, batchSize, lockUntil).Scan(&events).Error
	})
	if err != nil {
		return nil, fmt.Errorf("claim submission outbox events: %w", err)
	}
	return events, nil
}

func (d SubmissionOutboxDispatcher) markPublished(ctx context.Context, eventID uint64) error {
	if err := d.DB.WithContext(ctx).Exec(`
UPDATE submission_outbox
SET published_at = CURRENT_TIMESTAMP, locked_until = NULL, last_error = ''
WHERE id = ? AND published_at IS NULL`, eventID).Error; err != nil {
		return fmt.Errorf("mark submission outbox event %d published: %w", eventID, err)
	}
	return nil
}

func (d SubmissionOutboxDispatcher) releaseWithError(ctx context.Context, eventID uint64, publishErr error) error {
	nextAttempt := time.Now().Add(5 * time.Second)
	if err := d.DB.WithContext(ctx).Exec(`
UPDATE submission_outbox
SET locked_until = NULL, available_at = ?, last_error = ?
WHERE id = ? AND published_at IS NULL`, nextAttempt, publishErr.Error(), eventID).Error; err != nil {
		return fmt.Errorf("release submission outbox event %d: %w", eventID, err)
	}
	return nil
}
