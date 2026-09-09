package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"
)

const (
	sessionRefreshTTL  = 24 * time.Hour
	rememberRefreshTTL = 30 * 24 * time.Hour
)

func refreshTTLForRemember(remember bool) time.Duration {
	if remember {
		return rememberRefreshTTL
	}
	return sessionRefreshTTL
}

func refreshTTLFromRemaining(remaining time.Duration, defaultTTL time.Duration) time.Duration {
	if remaining <= 0 {
		return 0
	}
	if remaining > 8*24*time.Hour {
		return rememberRefreshTTL
	}
	if defaultTTL > 0 {
		return defaultTTL
	}
	return sessionRefreshTTL
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.TrimSpace(strings.Split(ip, ",")[0])
	}
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		return host[:i]
	}
	return host
}

func (h *Handler) createUserSession(ctx context.Context, tenantID, userID, ip, ua string) (string, error) {
	var sessionID string
	err := h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.user_sessions (tenant_id, user_id, ip_address, user_agent)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text
	`, tenantID, userID, nullString(ip), nullString(ua)).Scan(&sessionID)
	return sessionID, err
}

func (h *Handler) touchUserSession(ctx context.Context, sessionID string) {
	if sessionID == "" {
		return
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.user_sessions SET last_active = now() WHERE id = $1`, sessionID)
}

func (h *Handler) issueAuthTokens(r *http.Request, tenantID, tenantSlug, userID, email, role, sessionID string, refreshTTL time.Duration) (access, refresh string, err error) {
	ctx := r.Context()
	if sessionID == "" {
		sessionID, err = h.createUserSession(ctx, tenantID, userID, clientIP(r), r.UserAgent())
		if err != nil {
			return "", "", err
		}
	} else {
		h.touchUserSession(ctx, sessionID)
	}
	access, err = h.tokens.AccessToken(tenantID, tenantSlug, userID, email, role, sessionID)
	if err != nil {
		return "", "", err
	}
	if refreshTTL <= 0 {
		refreshTTL = h.tokens.DefaultRefreshTTL()
	}
	refresh, err = h.tokens.RefreshTokenWithTTL(tenantID, userID, sessionID, refreshTTL)
	return access, refresh, err
}
