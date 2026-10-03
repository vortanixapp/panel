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

	"github.com/vortanixapp/panel/internal/api/mail"
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/settingsreg"
)

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	h.clearAuthCookies(w, r)
	if ok {
		h.forgetLogin(w, r, claims.UserID)
	}
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
	token := strings.TrimSpace(req.RefreshToken)
	if token == "" {
		token = cookieValue(r, refreshCookie)
	}
	userID, sessionID, tokenID, expiresAt, err := h.tokens.ParseRefreshDetails(token)
	if err != nil {
		h.clearAuthCookies(w, r)
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		h.clearAuthCookies(w, r)
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	refreshTTL := refreshTTLFromRemaining(remaining, h.tokens.DefaultRefreshTTL())
	ctx := r.Context()

	if sessionID == "" {
		h.clearAuthCookies(w, r)
		writeError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}
	refreshJTI, outcome, rotateErr := h.rotateRefresh(ctx, userID, sessionID, tokenID)
	if rotateErr != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	switch outcome {
	case refreshOutcomeGone:
		h.clearAuthCookies(w, r)
		h.forgetLogin(w, r, userID)
		writeError(w, http.StatusUnauthorized, "Сессия закрыта")
		return
	case refreshOutcomeReused:
		h.onRefreshReuse(ctx, r, userID, sessionID)
		h.clearAuthCookies(w, r)
		h.forgetLogin(w, r, userID)
		writeError(w, http.StatusUnauthorized, "Сессия закрыта: токен обновления уже использовался. Войдите заново")
		return
	}
	var email, role string
	err = h.dbOf(ctx).QueryRow(ctx, `
		SELECT u.email, u.role
		FROM core.users u
		WHERE u.id = $1 AND u.status = 'active'
	`, userID).Scan(&email, &role)
	if err != nil {
		h.clearAuthCookies(w, r)
		h.forgetLogin(w, r, userID)
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}
	access, refresh, err := h.reissueAuthTokens(r, userID, email, role, sessionID, refreshJTI, refreshTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}
	h.startSession(w, r, userID, email, role, access, refresh, refreshTTL)
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
	started := time.Now()
	var req forgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.TenantSlug = singleTenantSlug
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "email and tenant_slug are required")
		return
	}
	ctx := r.Context()
	if h.tooManyAttempts(w, r, "forgot", intOf(settingsreg.AuthForgotAttempts), attemptsWindow(), req.Email) {
		return
	}
	var userID string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT u.id::text FROM core.users u
		WHERE u.email = $1
	`, strings.ToLower(req.Email)).Scan(&userID)
	if err != nil {
		holdUntil(started, uniformResponseFloor)
		writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})
		return
	}
	token := randomToken(32)
	hash := sha256Hex(token)
	expires := time.Now().Add(settingsreg.AuthResetLinkTTL.Duration())
	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.password_reset_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, hash, expires)

	resetURL := strings.TrimRight(envOr("FRONTEND_URL", "http://localhost:3000"), "/") +
		"/reset-password?token=" + url.QueryEscape(token)
	if h.mailConfigured(ctx) {
		msg := mail.PasswordResetEmail(i18n.ForUser(ctx, h.dbOf(ctx), userID), h.mailBrand(ctx, r), resetURL)
		msg.To = req.Email
		h.sendMailAsync("mail.reset", userID, req.Email, msg)
	} else if h.mail.DevExpose {
		log.Printf("восстановление пароля: почта не настроена, ссылка для %s: %s", req.Email, resetURL)
	} else {
		log.Printf("восстановление пароля: почта не настроена, письмо для %s не отправлено", req.Email)
	}

	holdUntil(started, uniformResponseFloor)
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
	if req.Token == "" || passwordTooShort(req.NewPassword) {
		writeError(w, http.StatusBadRequest, passwordTooShortMessage())
		return
	}
	ctx := r.Context()
	if h.tooManyAttempts(w, r, "reset", intOf(settingsreg.AuthResetAttempts), attemptsWindow()) {
		return
	}
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
	newHash, hashErr := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if hashErr != nil {
		writeError(w, http.StatusBadRequest, "Пароль слишком длинный — не больше 72 байт")
		return
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.users SET password_hash = $1 WHERE id = $2`, string(newHash), userID)
	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.password_reset_tokens SET used_at = now()
		WHERE user_id = $1::uuid AND used_at IS NULL
	`, userID)

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
