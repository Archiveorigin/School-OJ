package middleware

import (
	"fmt"
	"io"
	"sync/atomic"
)

var authorizationDenied, authorizationSensitiveAllow, authorizationAuditFailures atomic.Uint64

func RecordAuthorization(denied bool) {
	if denied {
		authorizationDenied.Add(1)
	} else {
		authorizationSensitiveAllow.Add(1)
	}
}
func RecordAuthorizationAuditFailure() { authorizationAuditFailures.Add(1) }
func writeAuthorizationMetrics(w io.Writer) {
	fmt.Fprintf(w, "oj_authorization_decisions_total{decision=\"deny\"} %d\n", authorizationDenied.Load())
	fmt.Fprintf(w, "oj_authorization_decisions_total{decision=\"sensitive_allow\"} %d\n", authorizationSensitiveAllow.Load())
	fmt.Fprintf(w, "oj_authorization_audit_failures_total %d\n", authorizationAuditFailures.Load())
}
