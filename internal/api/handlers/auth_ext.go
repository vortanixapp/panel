package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/vortanix/vortanix/internal/api/mail"
)

// Logout закрывает сессию на сервере. Раньше он только отвечал "logged_out":
// запись о сессии оставалась, и токен обновления продолжал работать — выход
// был виден лишь в браузере, который забывал токены.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if ok && claims.SessionID != "" {
		if _, err := h.dbOf(r.Context()).Exec(r.Context(),
			`DELETE FROM core.user_sessions WHERE id = $1 AND user_id = $2`,
			claims.SessionID, claims.UserID,
		); err != nil {
			log.Printf("сессия %s не закрыта: %v", claims.SessionID, err)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	tenantID, userID, sessionID, expiresAt, err := h.tokens.ParseRefreshDetails(req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	refreshTTL := refreshTTLFromRemaining(remaining, h.tokens.DefaultRefreshTTL())
	ctx := r.Context()

	// Закрытая сессия не должна воскресать обновлением токена. Иначе выход и
	// кнопка "завершить сессию" в настройках оставались бы жестом: токен жил
	// бы до истечения срока и продлевался дальше.
	if sessionID != "" {
		var alive bool
		if err := h.dbOf(ctx).QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM core.user_sessions WHERE id = $1 AND user_id = $2)`,
			sessionID, userID,
		).Scan(&alive); err != nil || !alive {
			writeError(w, http.StatusUnauthorized, "сессия закрыта")
			return
		}
	}
	var email, role, slug string
	err = h.dbOf(ctx).QueryRow(ctx, `
		SELECT u.email, u.role, t.slug
		FROM core.users u
		JOIN core.tenants t ON t.id = u.tenant_id
		WHERE u.id = $1 AND u.tenant_id = $2 AND u.status = 'active'
	`, userID, tenantID).Scan(&email, &role, &slug)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}
	access, refresh, err := h.issueAuthTokens(r, tenantID, slug, userID, email, role, sessionID, refreshTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"access_token":  access,
		"refresh_token": refresh,
	})
}

type forgotPasswordRequest struct {
	Email      string `json:"email"`
	TenantSlug string `json:"tenant_slug"`
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Email == "" || req.TenantSlug == "" {
		writeError(w, http.StatusBadRequest, "email and tenant_slug are required")
		return
	}
	ctx := r.Context()
	var userID string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT u.id::text FROM core.users u
		JOIN core.tenants t ON t.id = u.tenant_id
		WHERE t.slug = $1 AND u.email = $2
	`, req.TenantSlug, strings.ToLower(req.Email)).Scan(&userID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
		return
	}
	token := randomToken(32)
	hash := sha256Hex(token)
	expires := time.Now().Add(2 * time.Hour)
	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, hash, expires)

	resetURL := strings.TrimRight(envOr("FRONTEND_URL", "http://localhost:3000"), "/") +
		"/reset-password?token=" + url.QueryEscape(token)
	if h.mail.Enabled() {
		if err := h.mail.Send(req.Email, "Восстановление пароля", mail.PasswordResetBody(resetURL)); err != nil {
			log.Printf("forgot-password: письмо на %s не отправлено: %v", req.Email, err)
		}
	} else {
		log.Printf("forgot-password: SMTP не настроен, ссылка для %s: %s", req.Email, resetURL)
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
}

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Token == "" || len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "token and password (8+ chars) required")
		return
	}
	ctx := r.Context()
	hash := sha256Hex(req.Token)
	var userID string
	var expires time.Time
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT user_id::text, expires_at FROM core.password_reset_tokens
		WHERE token_hash = $1 AND used_at IS NULL
		ORDER BY created_at DESC LIMIT 1
	`, hash).Scan(&userID, &expires)
	if err != nil || time.Now().After(expires) {
		writeError(w, http.StatusBadRequest, "invalid or expired token")
		return
	}
	newHash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.users SET password_hash = $1 WHERE id = $2`, string(newHash), userID)
	_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.password_reset_tokens SET used_at = now() WHERE token_hash = $1`, hash)

	// Сброс пароля — это способ вернуть угнанный аккаунт, но старые сеансы
	// переживали его и оставались у того, кто увёл доступ. Здесь закрываем все
	// без исключения: своего сеанса у человека сейчас нет, он входит заново.
	if _, err := h.dbOf(ctx).Exec(ctx, `
		DELETE FROM core.user_sessions WHERE user_id = $1::uuid
	`, userID); err != nil {
		log.Printf("сброс пароля %s: сеансы не закрыты: %v", userID, err)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func randomToken(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
