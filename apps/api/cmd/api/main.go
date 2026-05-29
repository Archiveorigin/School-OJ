package main

import (
	"context"
	"log"

	"school-oj/apps/api/internal/config"
	"school-oj/apps/api/internal/db"
	"school-oj/apps/api/internal/handlers"
	"school-oj/apps/api/internal/services"
	"school-oj/apps/api/internal/streams"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	ctx := context.Background()
	cfg := config.Load()

	gdb, err := db.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	if cfg.AutoMigrate {
		if err := db.AutoMigrate(gdb); err != nil {
			log.Fatalf("migrate: %v", err)
		}
	}
	redisClient, err := streams.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	minioClient, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKey, cfg.MinIOSecretKey, ""),
		Secure: cfg.MinIOUseSSL,
	})
	if err != nil {
		log.Fatalf("minio: %v", err)
	}
	if err := services.EnsureBucket(ctx, minioClient, cfg); err != nil {
		log.Fatalf("minio bucket: %v", err)
	}
	if cfg.SeedData {
		if err := services.Seed(ctx, gdb, minioClient, cfg); err != nil {
			log.Fatalf("seed: %v", err)
		}
	}
	server := handlers.Server{DB: gdb, Redis: redisClient, MinIO: minioClient, Cfg: cfg}
	if err := server.Router().Run(cfg.Addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}
