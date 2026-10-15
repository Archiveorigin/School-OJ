package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
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
	CORSOrigins    []string
	TrustedProxies []string
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
		JWTSecret:      env("JWT_SECRET", ""),
		SMTPHost:       env("SMTP_HOST", ""),
		SMTPPort:       envInt("SMTP_PORT", 1025),
		SMTPUsername:   env("SMTP_USERNAME", ""),
		SMTPPassword:   env("SMTP_PASSWORD", ""),
		MailFrom:       env("MAIL_FROM", "no-reply@huanghailuoj.local"),
		MailFromName:   env("MAIL_FROM_NAME", "黄海在线"),
		AutoMigrate:    envBool("AUTO_MIGRATE", false),
		SeedData:       envBool("SEED_DATA", false),
		JPlagJarPath:   env("JPLAG_JAR_PATH", ""),
		JPlagWorkDir:   env("JPLAG_WORK_DIR", os.TempDir()+"/jplag"),
		RequestTimeout: time.Duration(envInt("REQUEST_TIMEOUT_SECONDS", 30)) * time.Second,
		CORSOrigins:    envList("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173"),
		TrustedProxies: envList("TRUSTED_PROXIES", "127.0.0.1/32,::1/128,172.16.0.0/12"),
	}
}

// Validate prevents a missing/default signing key from reaching the HTTP server.
func (c Config) Validate() error {
	if len(c.JWTSecret) < 32 || strings.HasPrefix(strings.ToLower(c.JWTSecret), "dev-secret") {
		return errors.New("JWT_SECRET must contain at least 32 bytes of a randomly generated secret")
	}
	database, err := pgx.ParseConfig(c.DatabaseURL)
	if err != nil || strings.TrimSpace(c.DatabaseURL) == "" {
		return errors.New("DATABASE_URL must be a valid PostgreSQL connection string")
	}
	if database.Password == "" || database.Password == "ojpass" {
		return errors.New("DATABASE_URL must use a non-default database password")
	}
	if c.MinIOAccessKey == "" || c.MinIOAccessKey == "minioadmin" || c.MinIOSecretKey == "" || c.MinIOSecretKey == "minioadmin" {
		return errors.New("MinIO credentials must be set and must not use minioadmin")
	}
	if strings.TrimSpace(c.RedisPassword) == "" {
		return errors.New("REDIS_PASSWORD must be set")
	}
	return nil
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

func envList(key, fallback string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		value = fallback
	}
	if value == "" {
		return nil
	}
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}
