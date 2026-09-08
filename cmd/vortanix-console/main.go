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
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/vortanix/vortanix/internal/console/handlers"
	"github.com/vortanix/vortanix/internal/console/relayclient"
	"github.com/vortanix/vortanix/pkg/httplog"
	"github.com/vortanix/vortanix/pkg/netaddr"
	"github.com/vortanix/vortanix/pkg/paneljwt"
)

func main() {
	_ = godotenv.Load()
	port := env("PORT", "8083")
	redisURL := env("REDIS_URL", "redis://localhost:6379/0")
	relayURL := env("RELAY_URL", "http://localhost:8082")
	secret := env("INTERNAL_SECRET", "dev-internal-secret")
	jwtSecret := env("JWT_SECRET", "dev-secret-change-in-production")

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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
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
