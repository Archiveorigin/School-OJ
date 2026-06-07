package models

import "testing"

func TestFormatClassJoinCode(t *testing.T) {
	code := FormatClassJoinCode(1)
	if len(code) != 7 || code[0] != 'C' {
		t.Fatalf("unexpected join code: %q", code)
	}
	if code == FormatClassJoinCode(2) {
		t.Fatal("join codes should differ by class id")
	}
	if code == "C000001" {
		t.Fatal("join code should not expose the raw class id")
	}
}
