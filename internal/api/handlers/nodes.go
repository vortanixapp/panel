package handlers

import (
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vortanixapp/panel/internal/api/cache"
	"github.com/vortanixapp/panel/internal/api/mail"
	"github.com/vortanixapp/panel/internal/api/paneljwt"
	"github.com/vortanixapp/panel/internal/api/relay"
	"github.com/vortanixapp/panel/internal/api/storage"
	"github.com/vortanixapp/panel/pkg/oauth"
	"github.com/vortanixapp/panel/pkg/secretbox"
)

type Handler struct {
	db               *pgxpool.Pool
	readDB           *pgxpool.Pool
	tokens           *paneljwt.Manager
	cache            *cache.Cache
	relay            *relay.Client
	eggCDN           *storage.EggCDN
	oauth            *oauth.Registry
	mail             mail.Config
	frontendURL      string
	apiPublicURL     string
	jwtSecret        string
	telegramBotToken string
	uploadDir        string
	secrets          *secretbox.Box

	tgBotMu       sync.Mutex
	tgBotUsername string
	tgBotUntil    time.Time
}

func New(db, readDB *pgxpool.Pool, tokens *paneljwt.Manager, c *cache.Cache, relayClient *relay.Client, eggCDN *storage.EggCDN, deps HandlerDeps) *Handler {
	if readDB == nil {
		readDB = db
	}
	return &Handler{
		db: db, readDB: readDB, tokens: tokens, cache: c, relay: relayClient, eggCDN: eggCDN,
		oauth: deps.OAuth, mail: deps.Mail, frontendURL: deps.FrontendURL, apiPublicURL: deps.APIPublicURL,
		jwtSecret: deps.JWTSecret, telegramBotToken: deps.TelegramBotToken, uploadDir: deps.UploadDir,
		secrets: deps.Secrets,
	}
}

type HandlerDeps struct {
	OAuth            *oauth.Registry
	Mail             mail.Config
	FrontendURL      string
	APIPublicURL     string
	JWTSecret        string
	TelegramBotToken string
	UploadDir        string
	Secrets          *secretbox.Box
}

func (h *Handler) reader() *pgxpool.Pool {
	if h.readDB != nil {
		return h.readDB
	}
	return h.db
}
