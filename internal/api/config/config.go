package config

import (
	"os"
	"strconv"
	"strings"
)

type OAuthProvider struct {
	ClientID     string
	ClientSecret string
}

type Config struct {
	Port                 string
	DatabaseURL          string
	DatabaseReadURL      string
	RedisURL             string
	LicensePublicKeyPath string
	LicensePublicKeyPEM  string
	LicenseIssuer        string
	LicenseServiceURL    string
	LicenseGraceHours    int
	LicensePollSeconds   int
	PanelVersion         string
	JWTSecret            string
	AccessTokenTTLMin    int
	RefreshTokenTTLDays  int
	RelayURL             string
	InternalSecret       string
	MigrationsDir        string
	DBPublicHost         string
	DBPublicPort         string
	StripeWebhookSecret  string
	EggCDNPrefix         string
	NATSURL              string
	CORSOrigins          []string
	FrontendURL          string
	GoogleOAuth          OAuthProvider
	DiscordOAuth         OAuthProvider
	VKOAuth              OAuthProvider
	TelegramBotToken     string
	SMTPHost             string
	SMTPPort             string
	SMTPUser             string
	SMTPPass             string
	MailFrom             string
	MailDevExposeURL     bool
	UploadDir            string
	SecretsKey           string
}

func Load() Config {
	accessMin, _ := strconv.Atoi(getEnv("ACCESS_TOKEN_TTL_MIN", "60"))
	refreshDays, _ := strconv.Atoi(getEnv("REFRESH_TOKEN_TTL_DAYS", "7"))
	graceHours, err := strconv.Atoi(getEnv("LICENSE_GRACE_HOURS", "24"))
	if err != nil || graceHours < 1 || graceHours > 720 {
		graceHours = 24
	}
	pollSeconds, err := strconv.Atoi(getEnv("LICENSE_STATE_POLL_SECONDS", "60"))
	if err != nil || pollSeconds < 15 || pollSeconds > 3600 {
		pollSeconds = 60
	}
	return Config{
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://vortanix:vortanix@localhost:5432/vortanix?sslmode=disable"),
		DatabaseReadURL:      getEnv("DATABASE_READ_URL", ""),
		RedisURL:             getEnv("REDIS_URL", "redis://localhost:6379/0"),
		LicensePublicKeyPath: getEnv("LICENSE_PUBLIC_KEY_PATH", "../license-service/keys/public.pem"),
		LicensePublicKeyPEM:  getEnv("LICENSE_PUBLIC_KEY_PEM", ""),
		LicenseIssuer:        getEnv("LICENSE_ISSUER", "license.vortanix.app"),
		LicenseServiceURL:    getEnv("LICENSE_SERVICE_URL", "http://localhost:8081"),
		LicenseGraceHours:    graceHours,
		LicensePollSeconds:   pollSeconds,
		PanelVersion:         getEnv("PANEL_VERSION", "dev"),
		JWTSecret:            getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		AccessTokenTTLMin:    accessMin,
		RefreshTokenTTLDays:  refreshDays,
		RelayURL:             getEnv("RELAY_URL", "http://localhost:8082"),
		InternalSecret:       getEnv("INTERNAL_SECRET", "dev-internal-secret"),
		MigrationsDir:        getEnv("CORE_MIGRATIONS_DIR", "migrations/core"),
		DBPublicHost:         getEnv("DB_PUBLIC_HOST", ""),
		DBPublicPort:         getEnv("DB_PUBLIC_PORT", "5432"),
		StripeWebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
		EggCDNPrefix:         getEnv("EGG_CDN_PREFIX", ""),
		NATSURL:              getEnv("NATS_URL", ""),
		CORSOrigins:          parseCORSOrigins(),
		FrontendURL:          getEnv("FRONTEND_URL", "http://localhost:3000"),
		GoogleOAuth: OAuthProvider{
			ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		},
		DiscordOAuth: OAuthProvider{
			ClientID:     getEnv("DISCORD_CLIENT_ID", ""),
			ClientSecret: getEnv("DISCORD_CLIENT_SECRET", ""),
		},
		VKOAuth: OAuthProvider{
			ClientID:     getEnv("VK_CLIENT_ID", ""),
			ClientSecret: getEnv("VK_CLIENT_SECRET", ""),
		},
		TelegramBotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
		SMTPHost:         getEnv("SMTP_HOST", ""),
		SMTPPort:         getEnv("SMTP_PORT", "587"),
		SMTPUser:         getEnv("SMTP_USER", ""),
		SMTPPass:         getEnv("SMTP_PASS", ""),
		MailFrom:         getEnv("MAIL_FROM", "noreply@vortanix.app"),
		MailDevExposeURL: getEnv("MAIL_DEV_EXPOSE_URL", "false") == "true",
		UploadDir:        getEnv("UPLOAD_DIR", "data/uploads"),
		SecretsKey:       getEnv("SECRETS_KEY", ""),
	}
}

func parseCORSOrigins() []string {
	raw := getEnv("CORS_ORIGINS", "*")
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
