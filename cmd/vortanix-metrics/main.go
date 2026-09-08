package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/vortanix/vortanix/internal/metrics/handlers"
	"github.com/vortanix/vortanix/internal/metrics/store"
	"github.com/vortanix/vortanix/pkg/httplog"
	"github.com/vortanix/vortanix/pkg/httpprom"
	"github.com/vortanix/vortanix/pkg/netaddr"
	"github.com/vortanix/vortanix/pkg/tenantpools"
)

func main() {
	_ = godotenv.Load()
	port := env("PORT", "8084")
	dbURL := env("DATABASE_URL", "postgres://vortanix:vortanix@localhost:5432/vortanix?sslmode=disable")
	redisURL := env("REDIS_URL", "redis://localhost:6379/0")
	secret := env("INTERNAL_SECRET", "dev-internal-secret")
	retentionDays := envInt("METRICS_RETENTION_DAYS", 7)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("redis url: %v", err)
	}
	rdb := redis.NewClient(opts)
	defer rdb.Close()

	pools := tenantpools.New(pool, dbURL)
	defer pools.Close()

	st := store.New(pool, rdb, pools)
	h := handlers.New(st, secret)

	go retentionLoop(ctx, st, retentionDays)

	prom := httpprom.New("metrics-ingest")
	r := chi.NewRouter()
	r.Use(prom.Handler)
	r.Use(httplog.Logger)
	r.Use(middleware.Recoverer)
	r.Handle("/metrics", httpprom.MetricsHandler())
	r.Mount("/", h.Routes())

	srv := &http.Server{Addr: netaddr.Listen(port), Handler: r}
	go func() {
		log.Printf("metrics-ingest listening on :%s (retention %d days)", port, retentionDays)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func retentionLoop(ctx context.Context, st *store.Store, days int) {
	if os.Getenv("TIMESCALE_ENABLED") == "true" {
		log.Printf("metrics retention: using TimescaleDB policy, skipping SQL purge loop")
		return
	}

	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	runPurge := func() {
		purgeCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
		n, err := st.PurgeOlderThan(purgeCtx, days)
		if err != nil {
			log.Printf("metrics retention purge failed: %v", err)
			return
		}
		if n > 0 {
			log.Printf("metrics retention: deleted %d rows older than %d days", n, days)
		}
	}

	runPurge()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runPurge()
		}
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}
