package main

import (
	"context"
	"github.com/vortanix/vortanix/internal/api/tenantdb"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/vortanix/vortanix/internal/api/cache"
	"github.com/vortanix/vortanix/internal/api/config"
	"github.com/vortanix/vortanix/internal/api/db"
	"github.com/vortanix/vortanix/internal/api/handlers"
	"github.com/vortanix/vortanix/internal/api/jobwake"
	"github.com/vortanix/vortanix/internal/api/licenseclient"
	"github.com/vortanix/vortanix/internal/api/licensejwt"
	"github.com/vortanix/vortanix/internal/api/licensestate"
	"github.com/vortanix/vortanix/internal/api/mail"
	"github.com/vortanix/vortanix/internal/api/paneljwt"
	"github.com/vortanix/vortanix/internal/api/relay"
	"github.com/vortanix/vortanix/internal/api/storage"
	"github.com/vortanix/vortanix/pkg/httplog"
	"github.com/vortanix/vortanix/pkg/httpprom"
	"github.com/vortanix/vortanix/pkg/netaddr"
	"github.com/vortanix/vortanix/pkg/oauth"
	"github.com/vortanix/vortanix/pkg/secretbox"
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

	if err := handlers.SyncCatalogForAllTenants(ctx, pools.Write); err != nil {
		log.Printf("catalog sync: %v", err)
	}

	redisCache, err := cache.New(cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer redisCache.Close()

	licenseVerifier, err := loadLicenseVerifier(ctx, pools.Write, cfg)
	if err != nil {
		log.Fatalf("license verifier: %v (запустите license-service, чтобы он создал ключи)", err)
	}

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

	tenants := tenantdb.New(pools.Write, cfg.DatabaseURL, cfg.MigrationsDir)
	defer tenants.Close()

	// Базы арендаторов заведены раньше нынешнего кода, поэтому схему в них
	// докатываем на старте, а каталог засеваем в каждой отдельно: он лежит
	// в базе арендатора, а не в общей.
	tenants.MigrateAll(ctx)
	if slugs, err := tenants.Slugs(ctx); err == nil {
		for _, slug := range slugs {
			pool, err := tenants.Pool(ctx, slug)
			if err != nil {
				continue
			}
			if err := handlers.SyncCatalogForAllTenants(ctx, pool); err != nil {
				log.Printf("каталог арендатора %s не засеян: %v", slug, err)
			}
		}
	}

	h := handlers.New(pools.Write, pools.Reader(), licenseVerifier, tokens, redisCache, relayClient, eggCDN, handlers.HandlerDeps{
		OAuth: oauthRegistry,
		Mail: mail.Config{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort, User: cfg.SMTPUser,
			Pass: cfg.SMTPPass, From: cfg.MailFrom, DevExpose: cfg.MailDevExposeURL,
		},
		FrontendURL:       cfg.FrontendURL,
		APIPublicURL:      apiPublicURL,
		JWTSecret:         cfg.JWTSecret,
		Tenants:           tenants,
		InternalSecret:    cfg.InternalSecret,
		DBPublicHost:      cfg.DBPublicHost,
		DBPublicPort:      cfg.DBPublicPort,
		LicenseGraceHours: cfg.LicenseGraceHours,
		TelegramBotToken:  cfg.TelegramBotToken,
		UploadDir:         cfg.UploadDir,
		Secrets:           secrets,
	})

	licenseStore := licensestate.NewStore()
	licenseRefresher := licensestate.New(
		pools.Write, licenseStore,
		licenseclient.New(cfg.LicenseServiceURL),
		licenseVerifier,
		licensestate.Config{
			PollInterval: time.Duration(cfg.LicensePollSeconds) * time.Second,
			Grace:        time.Duration(cfg.LicenseGraceHours) * time.Hour,
			Version:      cfg.PanelVersion,
		},
		h.CountUsage,
	)
	h.AttachLicense(licenseStore, licenseRefresher)
	licenseRefresher.SetTenantPools(tenants.Pools)
	licenseRefresher.SetUpdateNotice(h.NotifyPanelUpdate)
	licenseRefresher.Start(ctx)

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

func loadLicenseVerifier(ctx context.Context, db *pgxpool.Pool, cfg config.Config) (*licensejwt.Verifier, error) {
	if pemText := strings.TrimSpace(cfg.LicensePublicKeyPEM); pemText != "" {
		v, err := licensejwt.LoadVerifierFromPEM(pemText, cfg.LicenseIssuer)
		if err == nil {
			cacheLicenseKey(ctx, db, v)
			return v, nil
		}
		log.Printf("license verifier: LICENSE_PUBLIC_KEY_PEM непригоден: %v", err)
	}

	var lastErr error
	for attempt := 1; attempt <= 5; attempt++ {
		v, err := licensejwt.LoadVerifier(cfg.LicensePublicKeyPath, cfg.LicenseIssuer)
		if err == nil {
			cacheLicenseKey(ctx, db, v)
			return v, nil
		}
		lastErr = err
		time.Sleep(time.Second)
	}
	log.Printf("license verifier: файл ключа недоступен (%v), пробую кэш в базе", lastErr)

	if row, err := licensestate.EnsureRow(ctx, db); err == nil && row.PublicKeyPEM != "" {
		if v, err := licensejwt.LoadVerifierFromPEM(row.PublicKeyPEM, cfg.LicenseIssuer); err == nil {
			log.Printf("license verifier: взят из кэша в базе")
			return v, nil
		}
	}

	client := licenseclient.New(cfg.LicenseServiceURL)
	for attempt := 1; attempt <= 5; attempt++ {
		pemText, err := client.PublicKey(ctx)
		if err == nil {
			v, err := licensejwt.LoadVerifierFromPEM(pemText, cfg.LicenseIssuer)
			if err == nil {
				log.Printf("license verifier: получен от сервиса лицензий")
				cacheLicenseKey(ctx, db, v)
				return v, nil
			}
		}
		lastErr = err
		time.Sleep(2 * time.Second)
	}

	return nil, lastErr
}

func cacheLicenseKey(ctx context.Context, db *pgxpool.Pool, v *licensejwt.Verifier) {
	if _, err := licensestate.EnsureRow(ctx, db); err != nil {
		return
	}
	if err := licensestate.SavePublicKey(ctx, db, v.PEM()); err != nil {
		log.Printf("license verifier: не удалось закэшировать ключ: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
