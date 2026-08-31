package streams

import (
	"errors"
	"testing"
	"time"

	"school-oj/apps/worker/internal/config"
	"school-oj/apps/worker/internal/models"

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
	if !isRetryableSystemMessage("docker sandbox infrastructure is not ready") {
		t.Fatal("expected docker daemon message to be retryable")
	}
	if !isRetryableSystemMessage("mkdir /tmp/school-oj-worker/seccomp: no such file or directory") {
		t.Fatal("expected missing seccomp directory to be retryable")
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

func TestJudgeLeaseTTLExceedsPendingReclaimWindow(t *testing.T) {
	consumer := Consumer{Cfg: config.Config{RetryIdleSeconds: 90}}
	if got, want := consumer.judgeLeaseTTL(), 3*time.Minute; got != want {
		t.Fatalf("judge lease ttl = %s, want %s", got, want)
	}
	consumer.Cfg.RetryIdleSeconds = 10
	if got := consumer.judgeLeaseTTL(); got != 2*time.Minute {
		t.Fatalf("short reclaim window must still use safe minimum lease, got %s", got)
	}
}

func TestProblemSnapshotForSubmissionUsesPinnedVersion(t *testing.T) {
	submission := models.Submission{
		Problem:        models.Problem{Title: "current", PackageObject: "problems/current.zip", TimeLimitMS: 1000},
		ProblemVersion: models.ProblemVersion{ID: 7, Title: "pinned", PackageObject: "problems/v1.zip", TimeLimitMS: 2500},
	}
	problem := problemSnapshotForSubmission(submission)
	if problem.Title != "pinned" || problem.PackageObject != "problems/v1.zip" || problem.TimeLimitMS != 2500 {
		t.Fatalf("judge problem = %#v; want pinned immutable version", problem)
	}
	if submission.Problem.Title != "current" {
		t.Fatal("snapshot selection must not mutate the canonical problem")
	}
}
