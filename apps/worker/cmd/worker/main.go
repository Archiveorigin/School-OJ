package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"school-oj/apps/worker/internal/config"
	"school-oj/apps/worker/internal/db"
	"school-oj/apps/worker/internal/runner"
	"school-oj/apps/worker/internal/streams"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--healthcheck" {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("http://127.0.0.1:9091/healthz")
		if err != nil {
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		return
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("configuration: %v", err)
	}
	dockerRunner := runner.DockerRunner{Cfg: cfg}
	if err := dockerRunner.Prepare(); err != nil {
		log.Fatalf("sandbox: %v", err)
	}
	if len(os.Args) == 2 && os.Args[1] == "--cleanup" {
		cleanupCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		defer cancel()
		if err := dockerRunner.CleanupStale(cleanupCtx); err != nil {
			log.Fatal(err)
		}
		return
	}
	warmCtx, cancelWarm := context.WithTimeout(ctx, 15*time.Minute)
	err := dockerRunner.WarmImages(warmCtx)
	cancelWarm()
	if err != nil {
		log.Fatalf("sandbox images: %v", err)
	}
	cleanupCtx, cancelCleanup := context.WithTimeout(ctx, 45*time.Second)
	err = dockerRunner.CleanupStale(cleanupCtx)
	cancelCleanup()
	if err != nil {
		log.Fatalf("sandbox cleanup: %v", err)
	}
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
		Runner: &dockerRunner,
	}
	monitor := &http.Server{Addr: cfg.MonitorAddr, Handler: consumer.Monitor(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := monitor.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("worker monitor: %v", err)
			stop()
		}
	}()
	defer monitor.Close()
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanupCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
				if err := dockerRunner.CleanupStale(cleanupCtx); err != nil {
					log.Printf("sandbox cleanup: %v", err)
				}
				cancel()
			}
		}
	}()
	log.Printf("judge-worker consumer=%s stream=%s group=%s", cfg.Consumer, cfg.Stream, cfg.Group)
	if err := consumer.Run(ctx); err != nil && ctx.Err() == nil {
		log.Fatalf("consumer: %v", err)
	}
}
