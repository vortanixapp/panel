package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/internal/api/paneljwt"
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

const (
	apiKeyKindAdmin    = "admin"
	apiKeyKindPersonal = "personal"
	personalTokenLimit = 10
)

var personalTokenScopes = []string{
	"servers.read",
	"servers.power",
	"servers.console",
	"servers.files.read",
	"servers.files.write",
	"servers.backups",
	"billing.read",
}

var personalTokenDays = map[int]bool{0: true, 30: true, 90: true, 365: true}

type personalTokenRoute struct {
	method  string
	pattern string
	scope   string
}

var personalTokenRoutes = []personalTokenRoute{
	{http.MethodGet, "/v1/me", ""},
	{http.MethodGet, "/v1/servers", "servers.read"},
	{http.MethodGet, "/v1/my-servers", "servers.read"},
	{http.MethodGet, "/v1/servers/*", "servers.read"},
	{http.MethodGet, "/v1/servers/*/detail", "servers.read"},
	{http.MethodGet, "/v1/servers/*/status", "servers.read"},
	{http.MethodGet, "/v1/servers/*/metrics", "servers.read"},
	{http.MethodGet, "/v1/servers/*/logs", "servers.read"},
	{http.MethodPost, "/v1/servers/*/power", "servers.power"},
	{http.MethodPost, "/v1/servers/*/console-command", "servers.console"},
	{http.MethodPost, "/v1/servers/*/files/list", "servers.files.read"},
	{http.MethodPost, "/v1/servers/*/files/read", "servers.files.read"},
	{http.MethodGet, "/v1/servers/*/files/download", "servers.files.read"},
	{http.MethodPost, "/v1/servers/*/files/write", "servers.files.write"},
	{http.MethodPost, "/v1/servers/*/files/mkdir", "servers.files.write"},
	{http.MethodPost, "/v1/servers/*/files/delete", "servers.files.write"},
	{http.MethodPost, "/v1/servers/*/files/upload", "servers.files.write"},
	{http.MethodGet, "/v1/servers/*/backups", "servers.backups"},
	{http.MethodPost, "/v1/servers/*/backups", "servers.backups"},
	{http.MethodDelete, "/v1/servers/*/backups", "servers.backups"},
	{http.MethodPost, "/v1/servers/*/backups/restore", "servers.backups"},
	{http.MethodGet, "/v1/servers/*/backups/remote", "servers.backups"},
	{http.MethodPost, "/v1/servers/*/backups/remote/restore", "servers.backups"},
	{http.MethodPost, "/v1/servers/*/backups/remote/delete", "servers.backups"},
	{http.MethodGet, "/v1/billing", "billing.read"},
}

func personalTokenScope(method, path string) (string, bool) {
	if len(path) > 1 {
		path = strings.TrimSuffix(path, "/")
	}
	parts := strings.Split(path, "/")
	for _, route := range personalTokenRoutes {
		if route.method != method {
			continue
		}
		pattern := strings.Split(route.pattern, "/")
		if len(pattern) != len(parts) {
			continue
		}
		match := true
		for i, segment := range pattern {
			if segment == "*" {
				if parts[i] == "" {
					match = false
					break
				}
				continue
			}
			if segment != parts[i] {
				match = false
				break
			}
		}
		if match {
			return route.scope, true
		}
	}
	return "", false
}

func (h *Handler) personalTokensEnabled(ctx context.Context) bool {
	v := strings.TrimSpace(h.tenantSettingString(ctx, "security.user_api_tokens"))
	return v == "" || truthySetting(v)
}

func (h *Handler) apiTokensVisible(ctx context.Context, userID string) bool {
	if h.personalTokensEnabled(ctx) {
		return true
	}
	var exists bool
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM core.api_keys WHERE user_id = $1 AND kind = 'personal' AND revoked_at IS NULL)
	`, userID).Scan(&exists)
	return exists
}

func (h *Handler) servePersonalToken(w http.ResponseWriter, r *http.Request, key *apiKeyClaims, next http.Handler) {
	if !h.personalTokensEnabled(r.Context()) {
		writeError(w, http.StatusForbidden, "Личные API-токены отключены администратором")
		return
	}
	path := r.URL.RawPath
	if path == "" {
		path = r.URL.Path
	}
	scope, known := personalTokenScope(r.Method, path)
	if !known {
		writeError(w, http.StatusForbidden, "Токен не даёт доступа к этому разделу")
		return
	}
	if scope != "" && !key.Scopes[scope] {
		writeError(w, http.StatusForbidden, "У токена нет права "+scope)
		return
	}
	claims := &paneljwt.Claims{
		UserID: key.UserID,
		Role:   rbacRoleUser,
		Email:  key.Email,
	}
	ctx := context.WithValue(r.Context(), authContextKey, claims)
	ctx = context.WithValue(ctx, apiKeyContextKey, key)
	next.ServeHTTP(w, r.WithContext(ctx))
}

func personalTokenView(id, name, prefix string, scopes []byte, createdAt time.Time, lastUsed *time.Time, lastIP string, expires *time.Time) map[string]any {
	list := []string{}
	_ = json.Unmarshal(scopes, &list)
	return map[string]any{
		"id":           id,
		"name":         name,
		"prefix":       prefix,
		"scopes":       list,
		"created_at":   createdAt.Format(time.RFC3339),
		"last_used_at": timeOrNil(lastUsed),
		"last_used_ip": lastIP,
		"expires_at":   timeOrNil(expires),
		"expired":      expires != nil && !expires.After(time.Now()),
	}
}

func (h *Handler) ListAPITokens(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT id::text, name, prefix, scopes, created_at, last_used_at, COALESCE(last_used_ip, ''), expires_at
		FROM core.api_keys
		WHERE user_id = $1 AND kind = 'personal' AND revoked_at IS NULL
		ORDER BY created_at DESC
	`, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	tokens := []map[string]any{}
	for rows.Next() {
		var id, name, prefix, lastIP string
		var scopes []byte
		var createdAt time.Time
		var lastUsed, expires *time.Time
		if rows.Scan(&id, &name, &prefix, &scopes, &createdAt, &lastUsed, &lastIP, &expires) != nil {
			continue
		}
		tokens = append(tokens, personalTokenView(id, name, prefix, scopes, createdAt, lastUsed, lastIP, expires))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tokens":  tokens,
		"scopes":  personalTokenScopes,
		"limit":   personalTokenLimit,
		"enabled": h.personalTokensEnabled(ctx),
	})
}

type apiTokenBody struct {
	Name          string   `json:"name"`
	Scopes        []string `json:"scopes"`
	ExpiresInDays int      `json:"expires_in_days"`
}

func (h *Handler) CreateAPIToken(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	if !h.personalTokensEnabled(ctx) {
		writeError(w, http.StatusForbidden, "Личные API-токены отключены администратором")
		return
	}
	var body apiTokenBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeError(w, http.StatusUnprocessableEntity, "Укажите название токена")
		return
	}
	if utf8.RuneCountInString(name) > 64 {
		writeError(w, http.StatusUnprocessableEntity, "Название токена — не длиннее 64 символов")
		return
	}
	allowed := map[string]bool{}
	for _, s := range personalTokenScopes {
		allowed[s] = true
	}
	seen := map[string]bool{}
	scopes := []string{}
	for _, s := range body.Scopes {
		s = strings.TrimSpace(s)
		if !allowed[s] {
			writeError(w, http.StatusUnprocessableEntity, "Неизвестное право: "+s)
			return
		}
		if seen[s] {
			continue
		}
		seen[s] = true
		scopes = append(scopes, s)
	}
	if len(scopes) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "Выберите хотя бы одно право")
		return
	}
	if !personalTokenDays[body.ExpiresInDays] {
		writeError(w, http.StatusUnprocessableEntity, "Срок действия: 30, 90 или 365 дней либо без срока")
		return
	}
	var expires *time.Time
	if body.ExpiresInDays > 0 {
		at := time.Now().Add(time.Duration(body.ExpiresInDays) * 24 * time.Hour)
		expires = &at
	}

	key, prefix, hash, err := generateAPIKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось создать токен")
		return
	}
	scopesJSON, _ := json.Marshal(scopes)
	db := h.dbOf(ctx)
	var id string
	var createdAt time.Time
	err = db.QueryRow(ctx, `
		INSERT INTO core.api_keys (user_id, name, prefix, key_hash, scopes, expires_at, kind)
		SELECT $1::uuid, $2::text, $3::text, $4::text, $5::jsonb, $6::timestamptz, 'personal'
		WHERE (
			SELECT count(*) FROM core.api_keys
			WHERE user_id = $1::uuid AND kind = 'personal' AND revoked_at IS NULL
			  AND (expires_at IS NULL OR expires_at > now())
		) < $7::int
		RETURNING id::text, created_at
	`, claims.UserID, name, prefix, hash, scopesJSON, expires, personalTokenLimit).Scan(&id, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusConflict, "Действующих токенов может быть не больше "+strconv.Itoa(personalTokenLimit)+" — отзовите ненужные")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	ip := clientIP(r)
	audit(ctx, db, claims.UserID, "api_token.create", "api_key:"+id, map[string]any{
		"name": name, "scopes": scopes, "expires_in_days": body.ExpiresInDays, "ip": ip,
	})
	if ip == "" {
		ip = "—"
	}
	h.notifyUser(ctx, claims.UserID, notify.Event{
		Kind:   notify.KindAPIToken,
		Title:  i18n.Key("notify.api_token_created.title"),
		Body:   i18n.Key("notify.api_token_created.body", i18n.Params{"name": name, "ip": ip}),
		Action: h.panelAction("notify.action.security", "/settings?tab=tokens"),
	})
	writeJSON(w, http.StatusCreated, map[string]any{
		"token": personalTokenView(id, name, prefix, scopesJSON, createdAt, nil, "", expires),
		"key":   key,
	})
}

func (h *Handler) RevokeAPIToken(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	id := chi.URLParam(r, "id")
	db := h.dbOf(ctx)
	var name string
	err := db.QueryRow(ctx, `
		UPDATE core.api_keys SET revoked_at = now()
		WHERE id::text = $1 AND user_id = $2 AND kind = 'personal' AND revoked_at IS NULL
		RETURNING name
	`, id, claims.UserID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Токен не найден или уже отозван")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(ctx, db, claims.UserID, "api_token.revoke", "api_key:"+id, map[string]any{"name": name})
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}
