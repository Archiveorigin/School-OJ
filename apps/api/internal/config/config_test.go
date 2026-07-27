package config

import (
	"reflect"
	"testing"
)

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
