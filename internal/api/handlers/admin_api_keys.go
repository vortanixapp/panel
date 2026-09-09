package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/internal/api/paneljwt"
)

const apiKeyPrefix = "vtx_"

type apiKeyBody struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	ExpiresAt string   `json:"expires_at"`
}

func generateAPIKey() (key, prefix, hash string, err error) {
	raw := make([]byte, 24)
	if _, err = rand.Read(raw); err != nil {
		return "", "", "", err
	}
	key = apiKeyPrefix + hex.EncodeToString(raw)
	sum := sha256.Sum256([]byte(key))
	return key, key[:len(apiKeyPrefix)+6], hex.EncodeToString(sum[:]), nil
}

func hashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(key)))
	return hex.EncodeToString(sum[:])
}

func (h *Handler) AdminAPIKeysList(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT k.id::text, k.name, k.prefix, k.scopes, k.last_used_at, k.expires_at,
		       k.revoked_at, k.created_at, COALESCE(u.email, '')
		FROM core.api_keys k
		LEFT JOIN core.users u ON u.id = k.user_id
		WHERE k.tenant_id = $1
		ORDER BY k.created_at DESC
	`, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var id, name, prefix, email string
		var scopes []byte
		var lastUsed, expires, revoked *time.Time
		var createdAt time.Time
		if rows.Scan(&id, &name, &prefix, &scopes, &lastUsed, &expires, &revoked, &createdAt, &email) != nil {
			continue
		}
		list = append(list, map[string]any{
			"id": id, "name": name, "prefix": prefix,
			"scopes":       json.RawMessage(scopes),
			"last_used_at": timeOrNil(lastUsed),
			"expires_at":   timeOrNil(expires),
			"revoked_at":   timeOrNil(revoked),
			"created_at":   createdAt.Format(time.RFC3339),
			"created_by":   email,
			"active":       revoked == nil && (expires == nil || expires.After(time.Now())),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"keys":       list,
		"all_scopes": rbacAllPermissionKeys(),
	})
}

func (h *Handler) AdminAPIKeyCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body apiKeyBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "укажите название ключа")
		return
	}

	allowed := map[string]bool{}
	for _, k := range rbacAllPermissionKeys() {
		allowed[k] = true
	}
	scopes := []string{}
	for _, s := range body.Scopes {
		s = strings.TrimSpace(s)
		if !allowed[s] {
			writeError(w, http.StatusBadRequest, "неизвестное право: "+s)
			return
		}
		scopes = append(scopes, s)
	}
	if len(scopes) == 0 {
		writeError(w, http.StatusBadRequest, "выберите хотя бы одно право")
		return
	}

	var expires any
	if s := strings.TrimSpace(body.ExpiresAt); s != "" {
		parsed, ok := parseFlexibleTime(s)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid expires_at")
			return
		}
		expires = parsed
	}

	key, prefix, hash, err := generateAPIKey()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось создать ключ")
		return
	}
	scopesJSON, _ := json.Marshal(scopes)

	var id string
	if h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.api_keys (tenant_id, user_id, name, prefix, key_hash, scopes, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7)
		RETURNING id::text
	`, claims.TenantID, nullableUUID(claims.UserID), name, prefix, hash, scopesJSON, expires).Scan(&id) != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "api_key.create", "api_key:"+id,
		map[string]any{"name": name, "scopes": scopes})
	h.auditAlert(r.Context(), claims.TenantID, claims.UserID, claims.Email, "api_key.create", name)

	writeJSON(w, http.StatusCreated, map[string]any{
		"id": id, "name": name, "prefix": prefix,
		"key": key,
	})
}

func (h *Handler) AdminAPIKeyRevoke(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.api_keys SET revoked_at = now()
		WHERE id = $1 AND tenant_id = $2 AND revoked_at IS NULL
	`, id, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "ключ не найден или уже отозван")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "api_key.revoke", "api_key:"+id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "revoked"})
}

type apiKeyClaims struct {
	ID       string
	TenantID string
	UserID   string
	Scopes   map[string]bool
}

func (h *Handler) authenticateAPIKey(ctx context.Context, key string) (*apiKeyClaims, bool) {
	key = strings.TrimSpace(key)
	if !strings.HasPrefix(key, apiKeyPrefix) {
		return nil, false
	}
	var id, tenantID, userID string
	var scopes []byte
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT id::text, tenant_id::text, COALESCE(user_id::text, ''), scopes
		FROM core.api_keys
		WHERE key_hash = $1
		  AND revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > now())
	`, hashAPIKey(key)).Scan(&id, &tenantID, &userID, &scopes)
	if err != nil {
		return nil, false
	}

	var list []string
	_ = json.Unmarshal(scopes, &list)
	set := map[string]bool{}
	for _, s := range list {
		set[s] = true
	}

	pool := h.dbOf(ctx)
	go func(keyID string) {
		bg, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = pool.Exec(bg, `UPDATE core.api_keys SET last_used_at = now() WHERE id = $1`, keyID)
	}(id)

	return &apiKeyClaims{ID: id, TenantID: tenantID, UserID: userID, Scopes: set}, true
}

type apiKeyContextKeyType struct{}

var apiKeyContextKey apiKeyContextKeyType

func (h *Handler) authWithAPIKey(next http.Handler) http.Handler {
	jwtChain := AuthMiddleware(h.tokens)(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimSpace(r.Header.Get("X-API-Key"))
		if raw == "" {
			jwtChain.ServeHTTP(w, r)
			return
		}
		key, ok := h.authenticateAPIKey(r.Context(), raw)
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid api key")
			return
		}
		if !strings.HasPrefix(r.URL.Path, "/v1/admin/") {
			writeError(w, http.StatusForbidden, "ключ API работает только с /v1/admin/*")
			return
		}
		claims := &paneljwt.Claims{
			TenantID: key.TenantID,
			UserID:   key.UserID,
			Role:     rbacRoleAdmin,
			Email:    "api-key:" + key.ID,
		}
		ctx := context.WithValue(r.Context(), authContextKey, claims)
		ctx = context.WithValue(ctx, apiKeyContextKey, key)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func apiKeyFromContext(ctx context.Context) (*apiKeyClaims, bool) {
	key, ok := ctx.Value(apiKeyContextKey).(*apiKeyClaims)
	return key, ok
}
