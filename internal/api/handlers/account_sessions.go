package handlers

import (
	"context"
	"net/http"
	"os"
	"strconv"
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

func trustedProxyHops() int {
	if raw := strings.TrimSpace(os.Getenv("TRUSTED_PROXY_HOPS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 0 {
			return n
		}
	}
	return 1
}

func clientIP(r *http.Request) string {
	hops := trustedProxyHops()
	if hops > 0 {
		if list := r.Header.Get("X-Forwarded-For"); list != "" {
			parts := strings.Split(list, ",")
			idx := len(parts) - hops
			if idx < 0 {
				idx = 0
			}
			if ip := strings.TrimSpace(parts[idx]); ip != "" {
				return ip
			}
		}
		if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); ip != "" {
			return ip
		}
	}
	return hostOnly(r.RemoteAddr)
}

func (h *Handler) createUserSession(ctx context.Context, userID, ip, ua string) (string, error) {
	var sessionID string
	err := h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.user_sessions ( user_id, ip_address, user_agent)
		VALUES ( $1, $2, $3)
		RETURNING id::text
	`, userID, nullString(ip), nullString(ua)).Scan(&sessionID)
	return sessionID, err
}

func (h *Handler) touchUserSession(ctx context.Context, sessionID string) {
	if sessionID == "" {
		return
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.user_sessions SET last_active = now() WHERE id = $1`, sessionID)
}

func (h *Handler) issueAuthTokens(r *http.Request, userID, email, role, sessionID string, refreshTTL time.Duration) (access, refresh string, err error) {
	ctx := r.Context()
	if sessionID == "" {
		sessionID, err = h.createUserSession(ctx, userID, clientIP(r), r.UserAgent())
		if err != nil {
			return "", "", err
		}
	} else {
		h.touchUserSession(ctx, sessionID)
	}
	access, err = h.tokens.AccessToken(userID, email, role, sessionID)
	if err != nil {
		return "", "", err
	}
	if refreshTTL <= 0 {
		refreshTTL = h.tokens.DefaultRefreshTTL()
	}
	refresh, err = h.tokens.RefreshTokenWithTTL(userID, sessionID, refreshTTL)
	return access, refresh, err
}
