package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL    string
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool
	Stream         string
	Group          string
	Consumer       string
	SandboxSeccomp string
	SandboxWorkRoot string
	SandboxCPU     string
	SandboxMemory  int
	SandboxPids    int
}

func Load() Config {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "worker-1"
	}
	return Config{
		DatabaseURL:    env("DATABASE_URL", "host=localhost user=oj password=ojpass dbname=oj port=5432 sslmode=disable TimeZone=Asia/Shanghai"),
		RedisAddr:      env("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  env("REDIS_PASSWORD", ""),
		RedisDB:        envInt("REDIS_DB", 0),
		MinIOEndpoint:  env("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey: env("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey: env("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:    env("MINIO_BUCKET", "oj-artifacts"),
		MinIOUseSSL:    envBool("MINIO_USE_SSL", false),
		Stream:         env("SUBMISSION_STREAM", "oj.submissions"),
		Group:          env("SUBMISSION_GROUP", "judge-workers"),
		Consumer:       env("SUBMISSION_CONSUMER", hostname),
		SandboxSeccomp: env("DOCKER_SANDBOX_SECCOMP", "/tmp/school-oj-worker/seccomp/oj-seccomp.json"),
		SandboxWorkRoot: env("SANDBOX_WORK_ROOT", "/tmp/school-oj-worker"),
		SandboxCPU:     env("SANDBOX_CPU", "1.0"),
		SandboxMemory:  envInt("SANDBOX_MEMORY_MB", 256),
		SandboxPids:    envInt("SANDBOX_PIDS", 128),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}
