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
	if err := validConfig().Validate(); err != nil {
		t.Fatal(err)
	}
}

func validConfig() Config {
	return Config{JWTSecret: strings.Repeat("x", 64), DatabaseURL: "postgres://oj:random-database-secret@localhost/oj", RedisPassword: "redis-secret", MinIOAccessKey: "oj-storage", MinIOSecretKey: "storage-secret"}
}

func TestUnsafeDefaultsRejected(t *testing.T) {
	t.Setenv("PGPASSWORD", "")
	t.Setenv("PGPASSFILE", t.TempDir()+"/absent")
	for _, mutate := range []func(*Config){
		func(c *Config) { c.DatabaseURL = "host=localhost user=oj password=ojpass dbname=oj" },
		func(c *Config) { c.DatabaseURL = "postgres://oj:ojpass@localhost/oj" },
		func(c *Config) { c.DatabaseURL = "postgres://oj@localhost/oj" },
		func(c *Config) { c.MinIOAccessKey = "minioadmin" },
		func(c *Config) { c.MinIOSecretKey = "minioadmin" },
		func(c *Config) { c.RedisPassword = "" },
	} {
		cfg := validConfig()
		mutate(&cfg)
		if cfg.Validate() == nil {
			t.Fatal("unsafe credentials accepted")
		}
	}
	t.Setenv("AUTO_MIGRATE", "")
	t.Setenv("SEED_DATA", "")
	if cfg := Load(); cfg.AutoMigrate || cfg.SeedData {
		t.Fatal("unsafe startup defaults")
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
