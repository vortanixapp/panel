package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/vortanixapp/panel/internal/worker/jobs"
	"github.com/vortanixapp/panel/internal/worker/jobwake"
	"github.com/vortanixapp/panel/internal/worker/mail"
	"github.com/vortanixapp/panel/internal/worker/relay"
	"github.com/vortanixapp/panel/pkg/netaddr"
	"github.com/vortanixapp/panel/pkg/secretbox"
	"github.com/vortanixapp/panel/pkg/tenantpools"
	"strings"
)

func main() {
	_ = godotenv.Load()
	dbURL := env("DATABASE_URL", "postgres://vortanix:vortanix@localhost:5432/vortanix?sslmode=disable")
	relayURL := env("RELAY_URL", "http://localhost:8082")
	secret := env("INTERNAL_SECRET", "dev-internal-secret")
	natsURL := env("NATS_URL", "")
	mailCfg := mail.Config{
		Host: env("SMTP_HOST", ""),
		Port: env("SMTP_PORT", "587"),
		User: env("SMTP_USER", ""),
		Pass: env("SMTP_PASS", ""),
		From: env("MAIL_FROM", "noreply@vortanix.app"),
	}
	// Доставка оповещений наружу: токен общего бота панели и адрес самой панели
	// для кнопок в письмах и сообщениях.
	telegramBotToken := env("TELEGRAM_BOT_TOKEN", "")
	panelURL := strings.TrimRight(env("FRONTEND_URL", env("APP_URL", "")), "/")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	secrets, err := secretbox.New(env("SECRETS_KEY", ""))
	if err != nil {
		log.Fatalf("secrets key: %v", err)
	}

	tenants := tenantpools.New(pool, dbURL)
	defer tenants.Close()

	relayClient := relay.New(relayURL, secret)
	runner := jobs.New(pool, relayClient, mailCfg, secrets).WithNotify(telegramBotToken, panelURL)
	wake := fanout(ctx, jobwake.Subscribe(ctx, natsURL))
	log.Println("worker started: provision, backup, node_setup, daemon, mailing, ftp, migrate, expiry, backup schedule processors")

	startLoops(ctx, runner, wake.subscribe())

	// Задачи арендатора лежат в его базе, а не в центральной: обработчики
	// core-api пишут их туда, куда указывает заголовок запроса. Поэтому циклы
	// поднимаются на каждую базу, иначе очередь арендатора никто не разбирает.
	go superviseTenants(ctx, tenants, relayClient, mailCfg, secrets, telegramBotToken, panelURL, wake)

	healthSrv := startHealthServer(ctx, env("PORT", "8086"), pool, runner)
	defer func() {
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelShutdown()
		_ = healthSrv.Shutdown(shutdownCtx)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	cancel()
	log.Println("worker stopped")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func startHealthServer(ctx context.Context, port string, pool *pgxpool.Pool, runner *jobs.Runner) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		loops, healthy := runner.Heartbeat().Status()

		pingCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		dbOK := pool.Ping(pingCtx) == nil

		status := "ok"
		code := http.StatusOK
		if !dbOK || !healthy {
			status = "unhealthy"
			code = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":   status,
			"database": dbOK,
			"loops":    loops,
		})
	})

	srv := &http.Server{
		Addr:              netaddr.Listen(port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		log.Printf("worker health listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("worker health server: %v", err)
		}
	}()
	return srv
}
