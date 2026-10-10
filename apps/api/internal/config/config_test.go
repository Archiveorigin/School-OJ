package config

import (
	"reflect"
	"strings"
	"testing"
)

func TestSigningSecretValidation(t *testing.T) {
	for _, secret := range []string{"", "dev-secret", strings.Repeat("a", 31), "dev-secret" + strings.Repeat("a", 40)} {
		if (Config{JWTSecret: secret}).Validate() == nil {
			t.Fatal("accepted a missing, short or development signing secret")
		}
	}
	if err := (Config{JWTSecret: strings.Repeat("x", 64)}).Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestEnvListTrimsAndDeduplicates(t *testing.T) {
	t.Setenv("TEST_LIST", " https://oj.example.edu,https://admin.example.edu, https://oj.example.edu ,,")
	got := envList("TEST_LIST", "")
	want := []string{"https://oj.example.edu", "https://admin.example.edu"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("envList() = %#v, want %#v", got, want)
	}
}

func TestEnvListUsesFallback(t *testing.T) {
	t.Setenv("TEST_LIST", "")
	got := envList("TEST_LIST", "one, two")
	want := []string{"one", "two"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("envList() = %#v, want %#v", got, want)
	}
}
