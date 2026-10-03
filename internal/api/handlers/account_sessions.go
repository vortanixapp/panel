package handlers

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/settingsreg"
)

func sessionRefreshTTL() time.Duration {
	return settingsreg.AuthRefreshTTLSession.Duration()
}

func rememberRefreshTTL() time.Duration {
	return settingsreg.AuthRefreshTTLRemember.Duration()
}

func refreshTTLForRemember(remember bool) time.Duration {
	if remember {
		return rememberRefreshTTL()
	}
	return sessionRefreshTTL()
}

func refreshTTLFromRemaining(remaining time.Duration, defaultTTL time.Duration) time.Duration {
	if remaining <= 0 {
		return 0
	}
	if remaining > defaultTTL+24*time.Hour && remaining > sessionRefreshTTL() {
		return rememberRefreshTTL()
	}
	if defaultTTL > 0 {
		return defaultTTL
	}
	return sessionRefreshTTL()
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

func (h *Handler) createUserSession(ctx context.Context, userID, ip, ua string) (sessionID, refreshJTI string, err error) {
	refreshJTI = randomToken(16)
	err = h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.user_sessions ( user_id, ip_address, user_agent, refresh_jti)
		VALUES ( $1, $2, $3, $4)
		RETURNING id::text
	`, userID, nullString(ip), nullString(ua), refreshJTI).Scan(&sessionID)
	return sessionID, refreshJTI, err
}

func (h *Handler) touchUserSession(ctx context.Context, sessionID string) {
	if sessionID == "" {
		return
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.user_sessions SET last_active = now() WHERE id = $1`, sessionID)
}

func (h *Handler) issueAuthTokens(r *http.Request, userID, email, role string, refreshTTL time.Duration) (access, refresh string, err error) {
	sessionID, refreshJTI, err := h.createUserSession(r.Context(), userID, clientIP(r), r.UserAgent())
	if err != nil {
		return "", "", err
	}
	return h.signAuthTokens(userID, email, role, sessionID, refreshJTI, refreshTTL)
}

func (h *Handler) reissueAuthTokens(r *http.Request, userID, email, role, sessionID, refreshJTI string, refreshTTL time.Duration) (access, refresh string, err error) {
	h.touchUserSession(r.Context(), sessionID)
	return h.signAuthTokens(userID, email, role, sessionID, refreshJTI, refreshTTL)
}

func (h *Handler) signAuthTokens(userID, email, role, sessionID, refreshJTI string, refreshTTL time.Duration) (access, refresh string, err error) {
	access, err = h.tokens.AccessToken(userID, email, role, sessionID)
	if err != nil {
		return "", "", err
	}
	if refreshTTL <= 0 {
		refreshTTL = h.tokens.DefaultRefreshTTL()
	}
	refresh, err = h.tokens.RefreshTokenWithTTL(userID, sessionID, refreshJTI, refreshTTL)
	return access, refresh, err
}
