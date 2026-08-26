package services

import (
	"testing"
	"time"
)

func TestSubmissionOutboxDispatcherDefaults(t *testing.T) {
	dispatcher := SubmissionOutboxDispatcher{}
	if dispatcher.PollInterval != 0 || dispatcher.BatchSize != 0 || dispatcher.LockTTL != 0 {
		t.Fatal("zero-value dispatcher options must use runtime defaults")
	}
}

func TestSubmissionOutboxLockTTLCanCoverPublishRoundTrip(t *testing.T) {
	dispatcher := SubmissionOutboxDispatcher{LockTTL: 30 * time.Second}
	if dispatcher.LockTTL < 10*time.Second {
		t.Fatal("outbox lock TTL is too short for a Redis publish round trip")
	}
}
