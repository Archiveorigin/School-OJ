package streams

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"school-oj/apps/worker/internal/runner"
)

var durationBuckets = []int64{1000, 5000, 15000, 30000, 60000, 120000, 300000, 3600000}

func (c *Consumer) metric(name string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := c.Redis.HIncrBy(ctx, "oj:metrics:worker", name, 1).Err(); err != nil {
		log.Printf("worker metric persistence: %v", err)
	}
}

func (c *Consumer) observeJudge(result runner.JudgeResult, duration time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := c.Redis.TxPipelined(ctx, func(p redis.Pipeliner) error {
		key := "oj:metrics:worker"
		p.HIncrBy(ctx, key, "status:"+string(result.Status), 1)
		p.HIncrBy(ctx, key, "duration_count", 1)
		p.HIncrBy(ctx, key, "duration_sum_ms", duration.Milliseconds())
		for _, bound := range durationBuckets {
			if duration.Milliseconds() <= bound {
				p.HIncrBy(ctx, key, fmt.Sprintf("bucket:%d", bound), 1)
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("judge metric persistence: %v", err)
	}
}

func (c *Consumer) Monitor() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		budget := c.Cfg.SubmissionTimeout
		if budget <= 0 {
			budget = 5 * time.Minute
		}
		last := atomic.LoadInt64(&c.progress)
		database, err := c.DB.DB()
		if last == 0 || time.Since(time.Unix(last, 0)) > budget+time.Minute || err != nil || database.PingContext(ctx) != nil || c.Redis.Ping(ctx).Err() != nil {
			http.Error(w, "worker not ready", 503)
			return
		}
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		groups, err := c.Redis.XInfoGroups(ctx, c.Cfg.Stream).Result()
		if err != nil {
			http.Error(w, "queue unavailable", 503)
			return
		}
		values, err := c.Redis.HGetAll(ctx, "oj:metrics:worker").Result()
		if err != nil {
			http.Error(w, "metrics unavailable", 503)
			return
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		for _, group := range groups {
			if group.Name == c.Cfg.Group {
				fmt.Fprintf(w, "oj_queue_pending %d\noj_queue_lag %d\n", group.Pending, group.Lag)
			}
		}
		for key, value := range values {
			n, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				continue
			}
			switch {
			case strings.HasPrefix(key, "status:"):
				fmt.Fprintf(w, "oj_judge_results_total{status=%q} %d\n", strings.TrimPrefix(key, "status:"), n)
			case key == "duration_sum_ms":
				fmt.Fprintf(w, "oj_judge_duration_seconds_sum %.3f\n", float64(n)/1000)
			case key == "duration_count":
				fmt.Fprintf(w, "oj_judge_duration_seconds_count %d\noj_judge_duration_seconds_bucket{le=\"+Inf\"} %d\n", n, n)
			case key == "reclaimed" || key == "lease_lost" || key == "retries":
				fmt.Fprintf(w, "oj_worker_%s_total %d\n", key, n)
			}
		}
		for _, bound := range durationBuckets {
			n, _ := strconv.ParseInt(values[fmt.Sprintf("bucket:%d", bound)], 10, 64)
			fmt.Fprintf(w, "oj_judge_duration_seconds_bucket{le=\"%g\"} %d\n", float64(bound)/1000, n)
		}
		fmt.Fprintf(w, "oj_worker_last_progress_timestamp_seconds %d\n", atomic.LoadInt64(&c.progress))
	})
	return mux
}
