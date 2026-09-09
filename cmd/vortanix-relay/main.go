package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/vortanixapp/panel/internal/relay/handlers"
	"github.com/vortanixapp/panel/internal/relay/hub"
	"github.com/vortanixapp/panel/pkg/httplog"
	"github.com/vortanixapp/panel/pkg/httpprom"
	"github.com/vortanixapp/panel/pkg/netaddr"
	"github.com/vortanixapp/panel/pkg/tenantpools"
)

func main() {
	_ = godotenv.Load()
	port := env("PORT", "8082")
	dbURL := env("DATABASE_URL", "postgres://vortanix:vortanix@localhost:5432/vortanix?sslmode=disable")
	redisURL := env("REDIS_URL", "redis://localhost:6379/0")
	secret := env("INTERNAL_SECRET", "dev-internal-secret")
	metricsURL := env("METRICS_INGEST_URL", "")

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

	h := handlers.New(pool, pools, rdb, hub.New(), secret, metricsURL).
		WithPanelURL(env("FRONTEND_URL", env("APP_URL", "")))
	prom := httpprom.New("agent-relay")
	r := chi.NewRouter()
	r.Get("/v1/agent/connect", h.AgentConnect)
	r.Handle("/metrics", httpprom.MetricsHandler())
	r.Group(func(r chi.Router) {
		r.Use(prom.Handler)
		r.Use(httplog.Logger)
		r.Use(middleware.Recoverer)
		r.Mount("/", h.Routes())
	})

	srv := &http.Server{Addr: netaddr.Listen(port), Handler: r}
	go func() {
		log.Printf("agent-relay listening on :%s", port)
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

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
