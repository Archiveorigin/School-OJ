package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr           string
	DatabaseURL    string
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOBucket    string
	MinIOUseSSL    bool
	JWTSecret      string
	SMTPHost       string
	SMTPPort       int
	SMTPUsername   string
	SMTPPassword   string
	MailFrom       string
	MailFromName   string
	AutoMigrate    bool
	SeedData       bool
	JPlagJarPath   string
	JPlagWorkDir   string
	RequestTimeout time.Duration
}

func Load() Config {
	return Config{
		Addr:           env("APP_ADDR", ":8080"),
		DatabaseURL:    env("DATABASE_URL", "host=localhost user=oj password=ojpass dbname=oj port=5432 sslmode=disable TimeZone=Asia/Shanghai"),
		RedisAddr:      env("REDIS_ADDR", "localhost:6379"),
		RedisPassword:  env("REDIS_PASSWORD", ""),
		RedisDB:        envInt("REDIS_DB", 0),
		MinIOEndpoint:  env("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey: env("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey: env("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:    env("MINIO_BUCKET", "oj-artifacts"),
		MinIOUseSSL:    envBool("MINIO_USE_SSL", false),
		JWTSecret:      env("JWT_SECRET", "dev-secret"),
		SMTPHost:       env("SMTP_HOST", ""),
		SMTPPort:       envInt("SMTP_PORT", 1025),
		SMTPUsername:   env("SMTP_USERNAME", ""),
		SMTPPassword:   env("SMTP_PASSWORD", ""),
		MailFrom:       env("MAIL_FROM", "no-reply@huanghailuoj.local"),
		MailFromName:   env("MAIL_FROM_NAME", "黄海在线"),
		AutoMigrate:    envBool("AUTO_MIGRATE", true),
		SeedData:       envBool("SEED_DATA", true),
		JPlagJarPath:   env("JPLAG_JAR_PATH", ""),
		JPlagWorkDir:   env("JPLAG_WORK_DIR", os.TempDir()+"/jplag"),
		RequestTimeout: time.Duration(envInt("REQUEST_TIMEOUT_SECONDS", 30)) * time.Second,
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
