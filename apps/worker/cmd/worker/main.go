package main

import (
	"context"
	"log"

	"school-oj/apps/worker/internal/config"
	"school-oj/apps/worker/internal/db"
	"school-oj/apps/worker/internal/runner"
	"school-oj/apps/worker/internal/streams"

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
	redisClient, err := streams.Redis(ctx, cfg)
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
	consumer := streams.Consumer{
		DB:     gdb,
		Redis:  redisClient,
		MinIO:  minioClient,
		Cfg:    cfg,
		Runner: runner.DockerRunner{Cfg: cfg},
	}
	log.Printf("judge-worker consumer=%s stream=%s group=%s", cfg.Consumer, cfg.Stream, cfg.Group)
	if err := consumer.Run(ctx); err != nil {
		log.Fatalf("consumer: %v", err)
	}
}
