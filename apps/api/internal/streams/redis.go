package streams

import (
	"context"
	"fmt"
	"time"

	"school-oj/apps/api/internal/config"

	"github.com/redis/go-redis/v9"
)

const SubmissionStream = "oj.submissions"

func Connect(ctx context.Context, cfg config.Config) (*redis.Client, error) {
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

func EnqueueSubmission(ctx context.Context, client *redis.Client, submissionID uint) (string, error) {
	return client.XAdd(ctx, &redis.XAddArgs{
		Stream: SubmissionStream,
		Values: map[string]any{
			"submission_id": submissionID,
		},
	}).Result()
}
