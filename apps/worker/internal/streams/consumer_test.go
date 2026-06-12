package streams

import (
	"errors"
	"testing"

	"school-oj/apps/worker/internal/config"

	"github.com/redis/go-redis/v9"
)

func TestMessageRetryCount(t *testing.T) {
	consumer := Consumer{Cfg: config.Config{MaxRetries: 3}}
	msg := redis.XMessage{Values: map[string]any{"submission_id": "42", "retry_count": "2"}}
	if got := consumer.messageRetryCount(msg); got != 2 {
		t.Fatalf("retry count = %d, want 2", got)
	}
	if got := consumer.messageRetryCount(redis.XMessage{Values: map[string]any{"submission_id": "42"}}); got != 0 {
		t.Fatalf("empty retry count = %d, want 0", got)
	}
}

func TestRetryableErrorClassification(t *testing.T) {
	err := retryable(errors.New("temporary"))
	if !isRetryable(err) {
		t.Fatal("expected retryable error")
	}
	if isRetryable(errors.New("permanent")) {
		t.Fatal("did not expect permanent error to be retryable")
	}
}

func TestRetryableSystemMessage(t *testing.T) {
	if !isRetryableSystemMessage("docker sandbox image or daemon is not ready") {
		t.Fatal("expected docker daemon message to be retryable")
	}
	if isRetryableSystemMessage("compile failed") {
		t.Fatal("compile failure must not be retryable")
	}
}

func TestSubmissionIDFromMessage(t *testing.T) {
	id, err := submissionIDFromMessage(redis.XMessage{Values: map[string]any{"submission_id": "42"}})
	if err != nil {
		t.Fatal(err)
	}
	if id != 42 {
		t.Fatalf("submission id = %d, want 42", id)
	}
	if _, err := submissionIDFromMessage(redis.XMessage{Values: map[string]any{"submission_id": "bad"}}); err == nil {
		t.Fatal("expected invalid submission id error")
	}
}
