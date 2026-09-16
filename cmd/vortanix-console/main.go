package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/vortanixapp/panel/internal/console/handlers"
	"github.com/vortanixapp/panel/internal/console/relayclient"
	"github.com/vortanixapp/panel/pkg/httplog"
	"github.com/vortanixapp/panel/pkg/netaddr"
	"github.com/vortanixapp/panel/pkg/paneljwt"
	"github.com/vortanixapp/panel/pkg/panelsecret"
)

func main() {
	_ = godotenv.Load()
	port := env("PORT", "8083")
	redisURL := env("REDIS_URL", "redis://localhost:6379/0")
	relayURL := env("RELAY_URL", "http://localhost:8082")
	secret := env("INTERNAL_SECRET", "dev-internal-secret")
	jwtSecret := env("JWT_SECRET", panelsecret.DevJWTSecret)

	ctx := context.Background()
	if dbURL := env("DATABASE_URL", ""); dbURL != "" {
		pool, err := pgxpool.New(ctx, dbURL)
		if err != nil {
			log.Fatalf("database: %v", err)
		}
		defer pool.Close()
		resolved, err := panelsecret.JWT(ctx, pool, jwtSecret)
		if err != nil {
			log.Fatalf("ключ подписи токенов: %v", err)
		}
		jwtSecret = resolved
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	rdb := redis.NewClient(opts)
	defer rdb.Close()

	h := handlers.New(rdb, relayclient.New(relayURL, secret), paneljwt.NewVerifier(jwtSecret))

	r := chi.NewRouter()
	r.Use(httplog.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   parseCORSOrigins(),
		AllowedMethods:   []string{"GET", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization"},
		AllowCredentials: true,
	}))
	r.Mount("/", h.Routes())

	srv := &http.Server{Addr: netaddr.Listen(port), Handler: r}
	go func() {
		log.Printf("console-gateway listening on :%s", port)
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

func parseCORSOrigins() []string {
	raw := env("CORS_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000")
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}
