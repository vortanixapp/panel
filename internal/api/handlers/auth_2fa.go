package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pquerna/otp/totp"
)

type challenge2FARequest struct {
	Token string `json:"two_factor_token"`
	Code  string `json:"code"`
}

func (h *Handler) Challenge2FA(w http.ResponseWriter, r *http.Request) {
	var req challenge2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Token = strings.TrimSpace(req.Token)
	req.Code = strings.TrimSpace(req.Code)
	if req.Token == "" || req.Code == "" {
		writeError(w, http.StatusBadRequest, "two_factor_token and code required")
		return
	}

	ctx := r.Context()
	var pending map[string]string
	ok, err := h.cache.GetJSON(ctx, "2fa:"+req.Token, &pending)
	if err != nil || !ok || pending["user_id"] == "" {
		writeError(w, http.StatusUnauthorized, "invalid or expired challenge")
		return
	}

	var secret string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT secret FROM core.two_factor_secrets WHERE user_id = $1
	`, pending["user_id"]).Scan(&secret); err != nil || secret == "" {
		writeError(w, http.StatusUnauthorized, "2fa not configured")
		return
	}
	secret = h.secrets.MustDecrypt(secret)
	if !totp.Validate(req.Code, secret) {
		writeError(w, http.StatusUnauthorized, "invalid code")
		return
	}

	_ = h.cache.Delete(ctx, "2fa:"+req.Token)
	access, refresh, err := h.issueAuthTokens(r, pending["tenant_id"], pending["tenant_slug"], pending["user_id"], pending["email"], pending["role"], "", rememberRefreshTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}
	audit(ctx, h.dbOf(ctx), pending["tenant_id"], pending["user_id"], "auth.2fa", "login", nil)
	h.recordLoginAttempt(ctx, r, pending["tenant_id"], pending["user_id"], pending["email"], "", true)
	h.notifyNewLogin(ctx, r, pending["tenant_id"], pending["user_id"], pending["email"])
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  access,
		"refresh_token": refresh,
		"user": map[string]string{
			"id":    pending["user_id"],
			"email": pending["email"],
			"role":  pending["role"],
		},
	})
}
