package handlers

import (
	"context"
	"github.com/vortanix/vortanix/internal/api/tenantdb"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vortanix/vortanix/internal/api/cache"
	"github.com/vortanix/vortanix/internal/api/licenseclient"
	"github.com/vortanix/vortanix/internal/api/licensejwt"
	"github.com/vortanix/vortanix/internal/api/licensestate"
	"github.com/vortanix/vortanix/internal/api/mail"
	"github.com/vortanix/vortanix/internal/api/paneljwt"
	"github.com/vortanix/vortanix/internal/api/relay"
	"github.com/vortanix/vortanix/internal/api/storage"
	"github.com/vortanix/vortanix/pkg/oauth"
	"github.com/vortanix/vortanix/pkg/secretbox"
)

type Handler struct {
	db                *pgxpool.Pool
	readDB            *pgxpool.Pool
	license           *licensejwt.Verifier
	tokens            *paneljwt.Manager
	cache             *cache.Cache
	relay             *relay.Client
	eggCDN            *storage.EggCDN
	tenants           *tenantdb.Registry
	internalSecret    string
	dbPublicHost      string
	dbPublicPort      string
	licenseGraceHours int
	oauth             *oauth.Registry
	mail              mail.Config
	frontendURL       string
	apiPublicURL      string
	jwtSecret         string
	telegramBotToken  string
	uploadDir         string
	secrets           *secretbox.Box

	tgBotMu       sync.Mutex
	tgBotUsername string
	tgBotUntil    time.Time

	licenseState *licensestate.Store
	licenseOps   LicenseOps
}

type LicenseOps interface {
	Activate(ctx context.Context, licenseKey, domain string) error
	ActivateIn(ctx context.Context, db *pgxpool.Pool, licenseKey, domain string) error
	RefreshNow(ctx context.Context)
	RefreshTenant(ctx context.Context, db *pgxpool.Pool)
	UpdateEvents(ctx context.Context, licenseKey, component string) ([]licenseclient.UpdateEvent, error)
	UpdateReleases(ctx context.Context, licenseKey, component string) (licenseclient.ComponentReleases, error)
	ReportBug(ctx context.Context, report licenseclient.BugReport) (int64, error)
}

func (h *Handler) AttachLicense(store *licensestate.Store, ops LicenseOps) {
	h.licenseState = store
	h.licenseOps = ops
}

func (h *Handler) licenseNow() *licensestate.State {
	if h.licenseState == nil {
		return licensestate.NewStore().Load()
	}
	return h.licenseState.Load()
}

func New(db, readDB *pgxpool.Pool, license *licensejwt.Verifier, tokens *paneljwt.Manager, c *cache.Cache, relayClient *relay.Client, eggCDN *storage.EggCDN, deps HandlerDeps) *Handler {
	if readDB == nil {
		readDB = db
	}
	return &Handler{
		db: db, readDB: readDB, license: license, tokens: tokens, cache: c, relay: relayClient, eggCDN: eggCDN,
		oauth: deps.OAuth, mail: deps.Mail, frontendURL: deps.FrontendURL, apiPublicURL: deps.APIPublicURL,
		jwtSecret: deps.JWTSecret, telegramBotToken: deps.TelegramBotToken, uploadDir: deps.UploadDir,
		secrets: deps.Secrets, tenants: deps.Tenants, internalSecret: deps.InternalSecret,
		dbPublicHost: deps.DBPublicHost, dbPublicPort: deps.DBPublicPort,
		licenseGraceHours: deps.LicenseGraceHours,
	}
}

type HandlerDeps struct {
	OAuth             *oauth.Registry
	Mail              mail.Config
	FrontendURL       string
	APIPublicURL      string
	JWTSecret         string
	TelegramBotToken  string
	UploadDir         string
	Secrets           *secretbox.Box
	Tenants           *tenantdb.Registry
	InternalSecret    string
	DBPublicHost      string
	DBPublicPort      string
	LicenseGraceHours int
}

func (h *Handler) reader() *pgxpool.Pool {
	if h.readDB != nil {
		return h.readDB
	}
	return h.db
}
