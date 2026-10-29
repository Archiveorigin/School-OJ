package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"time"

	"school-oj/apps/api/internal/config"
	"school-oj/apps/api/internal/db"
	"school-oj/apps/api/internal/handlers"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/services"
	"school-oj/apps/api/internal/streams"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "apply database migrations and exit")
	flag.Parse()
	ctx := context.Background()
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("configuration: %v", err)
	}

	gdb, err := db.Connect(ctx, cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	if cfg.AutoMigrate || *migrateOnly {
		if err := db.Migrate(gdb); err != nil {
			log.Fatalf("migrate: %v", err)
		}
	}
	if *migrateOnly {
		log.Print("database migrations complete")
		return
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
	outboxDispatcher := services.SubmissionOutboxDispatcher{DB: gdb, Redis: redisClient}
	go outboxDispatcher.Run(ctx)
	go services.RunAuditRetention(ctx, gdb)
	go middleware.ObserveAPICoverage(ctx, redisClient)
	server := handlers.Server{DB: gdb, Redis: redisClient, MinIO: minioClient, Cfg: cfg}
	// Bound slow headers and idle keep-alives without imposing a write deadline
	// on long-lived submission SSE or legitimate large multipart uploads.
	httpServer := &http.Server{
		Addr: cfg.Addr, Handler: server.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}
