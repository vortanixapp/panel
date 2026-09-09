package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/vortanixapp/panel/internal/api/paneljwt"
)

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/health", h.Health)
	r.Get("/v1/branding", h.GetBranding)
	r.Get("/v1/home", h.Home)
	r.Get("/v1/uploads/avatars/{filename}", h.ServeAvatar)
	r.Get("/v1/plugins/images/{id}", h.ServePluginImage)
	r.Get("/v1/internal/catalog-archive", h.ServeCatalogArchive)
	r.Get("/v1/news/images/{id}", h.ServeNewsImage)
	r.Get("/v1/uploads/branding/{filename}", h.ServeBranding)
	r.Get("/v1/tenants/status", h.TenantStatus)
	r.Post("/v1/tenants/bootstrap", h.Bootstrap)
	r.Get("/v1/setup/status", h.SetupStatus)
	r.Post("/v1/auth/login", h.Login)
	r.Post("/v1/auth/register", h.Register)
	r.Post("/v1/webhooks/stripe", h.StripeWebhook)
	r.Post("/v1/webhooks/yookassa", h.YooKassaWebhook)
	r.HandleFunc("/v1/webhooks/freekassa", h.FreekassaWebhook)
	r.HandleFunc("/v1/webhooks/robokassa", h.RobokassaWebhook)
	r.Post("/v1/webhooks/paypal", h.PayPalWebhook)
	r.Post("/v1/webhooks/nowpayments", h.NowPaymentsWebhook)
	h.mountPublicMonitoring(r)
	h.mountPublicAuth(r)
	h.mountProtected(r)
	return r
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	if err := h.cache.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "redis unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type bootstrapRequest struct {
	OwnerEmail    string `json:"owner_email"`
	OwnerPassword string `json:"owner_password"`
	PanelName     string `json:"panel_name"`
}

func (h *Handler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	var req bootstrapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.OwnerEmail == "" || req.OwnerPassword == "" {
		writeError(w, http.StatusBadRequest, "owner_email and owner_password are required")
		return
	}
	if len(req.OwnerPassword) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	if h.tenantExists(r) {
		writeError(w, http.StatusConflict, "панель уже настроена")
		return
	}

	ctx := r.Context()

	tenantID := uuid.NewString()
	panelName := strings.TrimSpace(req.PanelName)
	if panelName == "" {
		panelName = "Vortanix"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.OwnerPassword), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	tx, err := h.db.Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO core.tenants (id, slug, name, settings)
		VALUES ($1, $2, $3, '{}'::jsonb)
	`, tenantID, singleTenantSlug, panelName); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create tenant")
		return
	}

	var userID string
	err = tx.QueryRow(ctx, `
		INSERT INTO core.users (tenant_id, email, password_hash, role)
		VALUES ($1, $2, $3, 'owner')
		RETURNING id::text
	`, tenantID, strings.ToLower(req.OwnerEmail), string(hash)).Scan(&userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create owner user")
		return
	}

	if err := seedDefaultCatalog(ctx, tx, tenantID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to seed catalog")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "commit failed")
		return
	}

	access, refresh, err := h.issueAuthTokens(r, tenantID, singleTenantSlug, userID, req.OwnerEmail, "owner", "", rememberRefreshTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"tenant_id":     tenantID,
		"tenant_slug":   singleTenantSlug,
		"access_token":  access,
		"refresh_token": refresh,
		"user": map[string]string{
			"id":    userID,
			"email": req.OwnerEmail,
			"role":  "owner",
		},
	})
}

type loginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	TenantSlug string `json:"tenant_slug"`
	Remember   bool   `json:"remember"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	req.TenantSlug = singleTenantSlug

	ctx := r.Context()
	if blocked, reason := h.ipBlocked(ctx, clientIP(r)); blocked {
		h.recordLoginAttempt(ctx, r, "", "", req.Email, "адрес заблокирован", false)
		msg := "доступ с этого адреса заблокирован"
		if reason != "" {
			msg += ": " + reason
		}
		writeError(w, http.StatusForbidden, msg)
		return
	}

	var userID, tenantID, role, hash string
	var twoFAEnabled bool
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT u.id::text, u.tenant_id::text, u.role, u.password_hash, COALESCE(u.two_factor_enabled, false)
		FROM core.users u
		JOIN core.tenants t ON t.id = u.tenant_id
		WHERE t.slug = $1 AND u.email = $2 AND u.status = 'active' AND t.status = 'active'
	`, req.TenantSlug, strings.ToLower(req.Email)).Scan(&userID, &tenantID, &role, &hash, &twoFAEnabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			h.recordLoginAttempt(ctx, r, "", "", req.Email, "нет такого пользователя", false)
			h.autoBlockIfNeeded(ctx, h.tenantIDBySlug(ctx, req.TenantSlug), clientIP(r))
			writeError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)); err != nil {
		h.recordLoginAttempt(ctx, r, tenantID, userID, req.Email, "неверный пароль", false)
		h.autoBlockIfNeeded(ctx, tenantID, clientIP(r))
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if !twoFAEnabled && isStaffRole(role) && h.staffRequires2FA(ctx, tenantID) {
		h.recordLoginAttempt(ctx, r, tenantID, userID, req.Email, "требуется 2FA", false)
		writeError(w, http.StatusForbidden,
			"для сотрудников включена обязательная двухфакторная аутентификация — обратитесь к владельцу панели, чтобы её настроить")
		return
	}

	if twoFAEnabled {
		challenge := randomToken(16)
		_ = h.cache.SetJSON(ctx, "2fa:"+challenge, map[string]string{
			"tenant_id": tenantID, "user_id": userID, "email": req.Email,
			"role": role, "tenant_slug": req.TenantSlug,
		}, 5*time.Minute)
		writeJSON(w, http.StatusOK, map[string]any{
			"requires_2fa":     true,
			"two_factor_token": challenge,
		})
		return
	}

	access, refresh, err := h.issueAuthTokens(r, tenantID, req.TenantSlug, userID, req.Email, role, "", refreshTTLForRemember(req.Remember))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}

	h.recordLoginAttempt(ctx, r, tenantID, userID, req.Email, "", true)
	h.notifyNewLogin(ctx, r, tenantID, userID, req.Email)
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  access,
		"refresh_token": refresh,
		"user": map[string]string{
			"id":    userID,
			"email": req.Email,
			"role":  role,
		},
	})
}

func (h *Handler) tenantIDBySlug(ctx context.Context, slug string) string {
	var id string
	if h.dbOf(ctx).QueryRow(ctx, `SELECT id::text FROM core.tenants WHERE slug = $1`, slug).Scan(&id) != nil {
		return ""
	}
	return id
}

func (h *Handler) TenantStatus(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimSpace(r.URL.Query().Get("slug"))
	if slug == "" {
		slug = singleTenantSlug
	}

	ctx := r.Context()

	var exists bool
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM core.tenants WHERE slug = $1 AND status = 'active')
	`, slug).Scan(&exists); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"bootstrapped": exists})
}

type registerRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	TenantSlug string `json:"tenant_slug"`
	Name       string `json:"name"`
	LastName   string `json:"last_name"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	req.TenantSlug = singleTenantSlug
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	ctx := r.Context()
	var tenantID string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT id::text FROM core.tenants WHERE slug = $1 AND status = 'active'
	`, req.TenantSlug).Scan(&tenantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "tenant not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	email := strings.ToLower(req.Email)
	var emailTaken bool
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM core.users u
			JOIN core.tenants t ON t.id = u.tenant_id
			WHERE t.slug = $1 AND u.email = $2
		)
	`, req.TenantSlug, email).Scan(&emailTaken); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if emailTaken {
		writeError(w, http.StatusConflict, "email already registered")
		return
	}

	var userID string
	err = h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.users (tenant_id, email, password_hash, role)
		VALUES ($1, $2, $3, 'user')
		RETURNING id::text
	`, tenantID, email, string(hash)).Scan(&userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	firstName := strings.TrimSpace(req.Name)
	lastName := strings.TrimSpace(req.LastName)
	if firstName != "" || lastName != "" {
		displayName := strings.TrimSpace(firstName + " " + lastName)
		var dn, fn, ln *string
		if displayName != "" {
			dn = &displayName
		}
		if firstName != "" {
			fn = &firstName
		}
		if lastName != "" {
			ln = &lastName
		}
		_, _ = h.dbOf(ctx).Exec(ctx, `
			INSERT INTO core.user_profiles (user_id, display_name, first_name, last_name, updated_at)
			VALUES ($1, $2, $3, $4, now())
		`, userID, dn, fn, ln)
	}

	access, refresh, err := h.issueAuthTokens(r, tenantID, req.TenantSlug, userID, email, "user", "", 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}

	h.maybeSendVerificationEmail(userID, email)

	writeJSON(w, http.StatusCreated, map[string]any{
		"access_token":  access,
		"refresh_token": refresh,
		"user": map[string]string{
			"id":    userID,
			"email": email,
			"role":  "user",
		},
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id":     claims.UserID,
		"email":       claims.Email,
		"role":        claims.Role,
		"tenant_id":   claims.TenantID,
		"tenant_slug": claims.TenantSlug,
	})
}

type contextKey string

const authContextKey contextKey = "auth"

func AuthMiddleware(tokens *paneljwt.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "missing bearer token")
				return
			}
			claims, err := tokens.ParseAccess(strings.TrimPrefix(header, "Bearer "))
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			if _, err := uuid.Parse(claims.TenantID); err != nil {
				writeError(w, http.StatusUnauthorized, "invalid tenant in token")
				return
			}
			ctx := context.WithValue(r.Context(), authContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
