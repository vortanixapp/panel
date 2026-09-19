package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
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
	"github.com/vortanixapp/panel/pkg/relaytls"
	"github.com/vortanixapp/panel/pkg/secretbox"
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

	h := handlers.New(pool, rdb, hub.New(), secret, metricsURL).
		WithPanelURL(env("FRONTEND_URL", env("APP_URL", "")))
	if metricsURL == "" {
		go purgeMetricPoints(ctx, pool, env("METRICS_RETENTION_DAYS", "7"))
	}
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

	tlsSrv := startAgentTLS(ctx, pool, h)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	if tlsSrv != nil {
		_ = tlsSrv.Shutdown(shutdownCtx)
	}
}

func startAgentTLS(ctx context.Context, pool *pgxpool.Pool, h *handlers.Handler) *http.Server {
	port := env("RELAY_TLS_PORT", "8443")
	if port == "0" || strings.EqualFold(port, "off") {
		return nil
	}
	hosts := relaytls.HostsFrom(
		env("RELAY_TLS_HOSTS", ""),
		env("RELAY_PUBLIC_URL", ""),
		env("FRONTEND_URL", ""),
		env("SITE_ADDRESS", ""),
	)
	if len(hosts) == 0 {
		log.Printf("relay: защищённый порт для агентов не поднят — неизвестен внешний адрес (RELAY_PUBLIC_URL)")
		return nil
	}
	box, err := secretbox.New(env("SECRETS_KEY", ""))
	if err != nil {
		log.Printf("relay: ключ шифрования недоступен: %v", err)
		box = nil
	}
	material, err := relaytls.Ensure(ctx, pool, box, hosts)
	if err != nil {
		log.Printf("relay: сертификат для агентов не готов: %v", err)
		return nil
	}

	mux := chi.NewRouter()
	mux.Get("/health", h.Health)
	mux.Get("/v1/agent/connect", h.AgentConnect)

	srv := &http.Server{
		Addr:      netaddr.Listen(port),
		Handler:   mux,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{material.Certificate}, MinVersion: tls.VersionTLS12},
	}
	go func() {
		log.Printf("agent-relay tls listening on :%s (%s, отпечаток %s)", port, strings.Join(material.Hosts, ", "), material.Pin)
		if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			log.Printf("relay tls: %v", err)
		}
	}()
	return srv
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func purgeMetricPoints(ctx context.Context, pool *pgxpool.Pool, days string) {
	if n, err := strconv.Atoi(days); err != nil || n <= 0 {
		days = "7"
	}
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		purgeCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		tag, err := pool.Exec(purgeCtx, `
			DELETE FROM core.server_metric_points
			WHERE ts < now() - ($1::text || ' days')::interval
		`, days)
		cancel()
		if err != nil {
			log.Printf("relay: очистка старых метрик: %v", err)
		} else if tag.RowsAffected() > 0 {
			log.Printf("relay: удалено %d точек метрик старше %s дней", tag.RowsAffected(), days)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
