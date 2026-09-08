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
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"

	"github.com/vortanixapp/panel/internal/api/cache"
	"github.com/vortanixapp/panel/internal/api/config"
	"github.com/vortanixapp/panel/internal/api/db"
	"github.com/vortanixapp/panel/internal/api/handlers"
	"github.com/vortanixapp/panel/internal/api/jobwake"
	"github.com/vortanixapp/panel/internal/api/mail"
	"github.com/vortanixapp/panel/internal/api/paneljwt"
	"github.com/vortanixapp/panel/internal/api/relay"
	"github.com/vortanixapp/panel/internal/api/storage"
	"github.com/vortanixapp/panel/pkg/httplog"
	"github.com/vortanixapp/panel/pkg/httpprom"
	"github.com/vortanixapp/panel/pkg/netaddr"
	"github.com/vortanixapp/panel/pkg/oauth"
	"github.com/vortanixapp/panel/pkg/secretbox"
)

// version вшивается при сборке (-X main.version). Без него панель клиента
// показывала ядро как «dev» даже на боевом сервере.
var version = "dev"

func main() {
	_ = godotenv.Load()
	if os.Getenv("VORTANIX_VERSION") == "" {
		_ = os.Setenv("VORTANIX_VERSION", version)
	}
	cfg := config.Load()
	ctx := context.Background()

	pools, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DatabaseReadURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pools.Close()
	if cfg.DatabaseReadURL != "" && cfg.DatabaseReadURL != cfg.DatabaseURL {
		log.Printf("database: read replica enabled")
	}

	if err := db.Migrate(ctx, pools.Write, cfg.MigrationsDir); err != nil {
		log.Fatalf("схема базы: %v", err)
	}

	if err := handlers.SyncCatalog(ctx, pools.Write); err != nil {
		log.Printf("catalog sync: %v", err)
	}

	redisCache, err := cache.New(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer redisCache.Close()

	tokens := paneljwt.New(cfg.JWTSecret, cfg.AccessTokenTTLMin, cfg.RefreshTokenTTLDays)
	relayClient := relay.New(cfg.RelayURL, cfg.InternalSecret)
	closeNATS := jobwake.ConnectNATS(cfg.NATSURL)
	defer closeNATS()
	eggCDN := storage.NewEggCDN(cfg.EggCDNPrefix)
	oauthRegistry := oauth.NewRegistry(
		cfg.FrontendURL,
		cfg.GoogleOAuth.ClientID, cfg.GoogleOAuth.ClientSecret,
		cfg.DiscordOAuth.ClientID, cfg.DiscordOAuth.ClientSecret,
		cfg.VKOAuth.ClientID, cfg.VKOAuth.ClientSecret,
	)
	secrets, err := secretbox.New(cfg.SecretsKey)
	if err != nil {
		log.Fatalf("secrets key: %v", err)
	}
	if !secrets.Enabled() {
		log.Printf("warning: SECRETS_KEY не задан — ключи платёжных шлюзов, токены хостинг-панелей и SSH-пароли пишутся в БД открытым текстом")
	}
	apiPublicURL := getEnv("API_PUBLIC_URL", "http://localhost:"+cfg.Port)

	h := handlers.New(pools.Write, pools.Reader(), tokens, redisCache, relayClient, eggCDN, handlers.HandlerDeps{
		OAuth: oauthRegistry,
		Mail: mail.Config{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort, User: cfg.SMTPUser,
			Pass: cfg.SMTPPass, From: cfg.MailFrom, DevExpose: cfg.MailDevExposeURL,
		},
		FrontendURL:      cfg.FrontendURL,
		APIPublicURL:     apiPublicURL,
		JWTSecret:        cfg.JWTSecret,
		TelegramBotToken: cfg.TelegramBotToken,
		UploadDir:        cfg.UploadDir,
		Secrets:          secrets,
	})

	samplerCtx, stopSampler := context.WithCancel(ctx)
	defer stopSampler()
	h.StartMonitoringSampler(samplerCtx)
	h.StartHostingExpirySweeper(samplerCtx)
	h.StartServerDunningSweeper(samplerCtx)

	prom := httpprom.New("core-api")
	r := chi.NewRouter()
	r.Use(prom.Handler)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(httplog.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	// Панели клиентов живут на своих доменах и IP, заранее нам неизвестных,
	// поэтому список источников не закрыт. Это безопасно: cookie мы не выдаём,
	// вход идёт по Bearer-токену, и браузер не приложит его к запросу чужого
	// сайта. Ровно поэтому же выключены credentials — с ними браузер запрещает
	// «*», а держать точный перечень значило бы ломать каждую новую установку.
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Tenant-Slug"},
		AllowCredentials: false,
	}))

	r.Handle("/metrics", httpprom.MetricsHandler())
	r.Mount("/", h.Routes())

	srv := &http.Server{
		Addr:         netaddr.Listen(cfg.Port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Printf("core-api listening on :%s", cfg.Port)
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

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
