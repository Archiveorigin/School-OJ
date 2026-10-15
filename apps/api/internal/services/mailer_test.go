package services

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestDisabledMailerDoesNotLogSecretsOrPretendDelivery(t *testing.T) {
	var output bytes.Buffer
	original := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(original)
	if err := (Mailer{}).Send("private@example.test", "private subject", "verification code 739281"); err == nil {
		t.Fatal("disabled delivery reported success")
	}
	for _, secret := range []string{"private@example.test", "private subject", "739281"} {
		if strings.Contains(output.String(), secret) {
			t.Fatal("mail content leaked")
		}
	}
}
