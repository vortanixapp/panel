package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/pquerna/otp/totp"
	"github.com/vortanixapp/panel/pkg/notify"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) adminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := tenantClaims(r.Context())
		if !ok || !isStaffRole(claims.Role) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	var profile struct {
		DisplayName *string `json:"display_name"`
		FirstName   *string `json:"first_name"`
		LastName    *string `json:"last_name"`
		Phone       *string `json:"phone"`
		Locale      string  `json:"locale"`
		AvatarURL   *string `json:"avatar_url"`
	}
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT display_name, first_name, last_name, phone, locale, avatar_url
		FROM core.user_profiles WHERE user_id = $1
	`, claims.UserID).Scan(&profile.DisplayName, &profile.FirstName, &profile.LastName, &profile.Phone, &profile.Locale, &profile.AvatarURL)

	var twoFA bool
	var emailVerified *time.Time
	_ = h.dbOf(ctx).QueryRow(ctx, `SELECT two_factor_enabled, email_verified_at FROM core.users WHERE id = $1`, claims.UserID).Scan(&twoFA, &emailVerified)

	linked := []string{}
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT provider FROM core.user_social_accounts WHERE user_id = $1 ORDER BY provider
	`, claims.UserID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p string
			if rows.Scan(&p) == nil && p != "" {
				linked = append(linked, publicProviderKey(p))
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":                 claims.UserID,
			"email":              claims.Email,
			"role":               claims.Role,
			"two_factor_enabled": twoFA,
			"email_verified":     emailVerified != nil,
			"email_verified_at":  emailVerified,
			"display_name":       profile.DisplayName,
			"first_name":         profile.FirstName,
			"last_name":          profile.LastName,
			"phone":              profile.Phone,
			"locale":             profile.Locale,
			"avatar_url":         h.avatarPublicURL(r, profile.AvatarURL),
			"linked_providers":   linked,
		},
	})
}

func (h *Handler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx := r.Context()
	_, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.user_profiles (user_id, display_name, first_name, last_name, phone, locale, updated_at)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6, 'ru'), now())
		ON CONFLICT (user_id) DO UPDATE SET
			display_name = COALESCE(EXCLUDED.display_name, core.user_profiles.display_name),
			first_name = COALESCE(EXCLUDED.first_name, core.user_profiles.first_name),
			last_name = COALESCE(EXCLUDED.last_name, core.user_profiles.last_name),
			phone = COALESCE(EXCLUDED.phone, core.user_profiles.phone),
			locale = COALESCE(EXCLUDED.locale, core.user_profiles.locale),
			updated_at = now()
	`, claims.UserID, body["display_name"], body["first_name"], body["last_name"], body["phone"], body["locale"])
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

type destroySessionRequest struct {
	SessionID string `json:"session_id"`
	AllOthers bool   `json:"all_others"`
}

func (h *Handler) ListAccountSessions(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, ip_address, user_agent, last_active::text, created_at::text
		FROM core.user_sessions
		WHERE user_id = $1 AND tenant_id = $2
		ORDER BY last_active DESC
		LIMIT 20
	`, claims.UserID, claims.TenantID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"sessions": []any{}, "current_session_id": claims.SessionID})
		return
	}
	defer rows.Close()
	sessions := []map[string]any{}
	for rows.Next() {
		var id, ip, ua, lastActive, created string
		if rows.Scan(&id, &ip, &ua, &lastActive, &created) == nil {
			sessions = append(sessions, map[string]any{
				"id":            id,
				"ip_address":    ip,
				"user_agent":    ua,
				"last_activity": lastActive,
				"created_at":    created,
				"is_current":    id == claims.SessionID,
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sessions":           sessions,
		"current_session_id": claims.SessionID,
	})
}

func (h *Handler) DestroySession(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req destroySessionRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	ctx := r.Context()
	if req.AllOthers {
		_, _ = h.dbOf(ctx).Exec(ctx, `
			DELETE FROM core.user_sessions
			WHERE user_id = $1 AND tenant_id = $2 AND id != $3
		`, claims.UserID, claims.TenantID, claims.SessionID)
	} else if req.SessionID != "" && req.SessionID != claims.SessionID {
		_, _ = h.dbOf(ctx).Exec(ctx, `
			DELETE FROM core.user_sessions WHERE id = $1 AND user_id = $2 AND tenant_id = $3
		`, req.SessionID, claims.UserID, claims.TenantID)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type changeEmailRequest struct {
	Email           string `json:"email"`
	CurrentPassword string `json:"current_password"`
}

func (h *Handler) ChangeEmail(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req changeEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	newEmail := strings.ToLower(strings.TrimSpace(req.Email))
	if newEmail == "" || !strings.Contains(newEmail, "@") {
		writeError(w, http.StatusBadRequest, "valid email is required")
		return
	}
	if req.CurrentPassword == "" {
		writeError(w, http.StatusBadRequest, "current_password is required")
		return
	}

	ctx := r.Context()
	var hash string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT password_hash FROM core.users
		WHERE id = $1 AND tenant_id = $2 AND status = 'active'
	`, claims.UserID, claims.TenantID).Scan(&hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.CurrentPassword)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid current password")
		return
	}

	var taken bool
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM core.users
			WHERE tenant_id = $1 AND email = $2 AND id != $3
		)
	`, claims.TenantID, newEmail, claims.UserID).Scan(&taken); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if taken {
		writeError(w, http.StatusConflict, "email already in use")
		return
	}

	_, err = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.users SET email = $1, email_verified_at = NULL WHERE id = $2 AND tenant_id = $3
	`, newEmail, claims.UserID, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}

	h.maybeSendVerificationEmail(claims.UserID, newEmail)

	resp := map[string]any{"status": "email-updated", "email": newEmail}
	if !h.mail.Enabled() && h.mail.DevExpose {
		resp["verification_url"] = h.buildSignedVerifyURL(claims.UserID, newEmail)
	}
	writeJSON(w, http.StatusOK, resp)
}

var allowedAvatarTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

type avatarURLRequest struct {
	AvatarURL string `json:"avatar_url"`
}

func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	reqContentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(reqContentType, "application/json") {
		var req avatarURLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid json")
			return
		}
		avatarURL := strings.TrimSpace(req.AvatarURL)
		if avatarURL == "" || !strings.HasPrefix(avatarURL, "http://") && !strings.HasPrefix(avatarURL, "https://") {
			writeError(w, http.StatusBadRequest, "valid avatar_url is required")
			return
		}
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
			INSERT INTO core.user_profiles (user_id, avatar_url, updated_at)
			VALUES ($1, $2, now())
			ON CONFLICT (user_id) DO UPDATE SET avatar_url = EXCLUDED.avatar_url, updated_at = now()
		`, claims.UserID, avatarURL)
		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "avatar-updated",
			"avatar_url": avatarURL,
		})
		return
	}

	if err := r.ParseMultipartForm(4 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}
	file, header, err := r.FormFile("avatar")
	if err != nil {
		writeError(w, http.StatusBadRequest, "avatar file is required")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	ext, allowed := allowedAvatarTypes[contentType]
	if !allowed {
		lower := strings.ToLower(header.Filename)
		switch {
		case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
			ext = ".jpg"
			contentType = "image/jpeg"
		case strings.HasSuffix(lower, ".png"):
			ext = ".png"
			contentType = "image/png"
		case strings.HasSuffix(lower, ".webp"):
			ext = ".webp"
			contentType = "image/webp"
		default:
			writeError(w, http.StatusBadRequest, "avatar must be jpg, png, or webp")
			return
		}
	}

	avatarDir := filepath.Join(h.uploadDir, "avatars")
	if err := os.MkdirAll(avatarDir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "storage unavailable")
		return
	}

	filename := claims.UserID + ext
	destPath := filepath.Join(avatarDir, filename)
	out, err := os.Create(destPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save avatar")
		return
	}
	if _, err := io.Copy(out, file); err != nil {
		out.Close()
		writeError(w, http.StatusInternalServerError, "failed to save avatar")
		return
	}
	out.Close()

	avatarURL := h.publicBaseURL(r) + "/v1/uploads/avatars/" + filename
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.user_profiles (user_id, avatar_url, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (user_id) DO UPDATE SET avatar_url = EXCLUDED.avatar_url, updated_at = now()
	`, claims.UserID, avatarURL)

	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "avatar-updated",
		"avatar_url": avatarURL,
	})
}

func (h *Handler) ServeAvatar(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	if filename == "" || strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		writeError(w, http.StatusBadRequest, "invalid filename")
		return
	}
	path := filepath.Join(h.uploadDir, "avatars", filename)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	ext := strings.ToLower(filepath.Ext(filename))
	contentType := "application/octet-stream"
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".webp":
		contentType = "image/webp"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=300")
	http.ServeFile(w, r, path)
}

func (h *Handler) Generate2FA(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Vortanix",
		AccountName: claims.Email,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to generate 2fa secret")
		return
	}
	secret := key.Secret()
	sealed, err := h.secrets.Encrypt(secret)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encrypt 2fa secret")
		return
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.two_factor_secrets (user_id, secret)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET secret = EXCLUDED.secret
	`, claims.UserID, sealed)
	writeJSON(w, http.StatusOK, map[string]string{
		"secret": secret,
		"uri":    key.URL(),
	})
}

func (h *Handler) Enable2FA(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `UPDATE core.users SET two_factor_enabled = true WHERE id = $1`, claims.UserID)
	h.notifyUser(r.Context(), claims.TenantID, claims.UserID, notify.Event{
		Kind:  notify.KindTwoFactor,
		Title: "Двухфакторная защита включена",
		Body:  "Для входа в учётную запись теперь требуется код из приложения.",
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "enabled"})
}

func (h *Handler) Disable2FA(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()

	var body struct {
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if strings.TrimSpace(body.Password) == "" {
		writeError(w, http.StatusBadRequest, "нужен пароль для отключения второго фактора")
		return
	}
	var hash string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT password_hash FROM core.users WHERE id = $1 AND tenant_id = $2
	`, claims.UserID, claims.TenantID).Scan(&hash); err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(body.Password)) != nil {
		h.recordLoginAttempt(ctx, r, claims.TenantID, claims.UserID, claims.Email, "отключение 2FA: неверный пароль", false)
		writeError(w, http.StatusUnauthorized, "неверный пароль")
		return
	}

	_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.users SET two_factor_enabled = false WHERE id = $1`, claims.UserID)
	_, _ = h.dbOf(ctx).Exec(ctx, `DELETE FROM core.two_factor_secrets WHERE user_id = $1`, claims.UserID)
	audit(ctx, h.dbOf(ctx), claims.TenantID, claims.UserID, "user.2fa_disable", "user:"+claims.UserID, nil)
	h.notifyUser(r.Context(), claims.TenantID, claims.UserID, notify.Event{
		Kind:     notify.KindTwoFactor,
		Severity: notify.SeverityCritical,
		Title:    "Двухфакторная защита отключена",
		Body:     "Второй фактор для входа отключён. Если это были не вы, включите его снова и смените пароль.",
		Action:   h.panelAction("Настройки безопасности", "/settings"),
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
}

func (h *Handler) Translations(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	locale := "ru"
	if ok {
		var l string
		if err := h.dbOf(r.Context()).QueryRow(r.Context(), `SELECT locale FROM core.user_profiles WHERE user_id = $1`, claims.UserID).Scan(&l); err == nil && l != "" {
			locale = l
		}
	}
	messages := map[string]string{}
	if ok {
		rows, err := h.dbOf(r.Context()).Query(r.Context(), `
			SELECT key, value FROM core.translation_keys WHERE tenant_id = $1 AND locale = $2
		`, claims.TenantID, locale)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var k, v string
				if rows.Scan(&k, &v) == nil {
					messages[k] = v
				}
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"locale": locale, "messages": messages})
}

func (h *Handler) SetLocale(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	locale := chi.URLParam(r, "locale")
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.user_profiles (user_id, locale) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET locale = EXCLUDED.locale
	`, claims.UserID, locale)
	writeJSON(w, http.StatusOK, map[string]string{"locale": locale})
}
