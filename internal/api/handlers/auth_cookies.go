package handlers

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

const (
	accessCookie   = "vtx_access"
	refreshCookie  = "vtx_refresh"
	csrfCookie     = "vtx_csrf"
	userCookie     = "vtx_user"
	sessionsCookie = "vtx_sessions"
	csrfHeader     = "X-CSRF-Token"
	authPath       = "/v1/auth"
	maxSavedLogins = 6
)

type savedLogin struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	Refresh string `json:"refresh"`
	Used    int64  `json:"used"`
}

type publicClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Exp    int64  `json:"exp"`
}

func requestIsSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(firstListValue(r.Header.Get("X-Forwarded-Proto")), "https")
}

func crossSiteRequest(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" || origin == "null" {
		return false
	}
	host := firstListValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	return !strings.EqualFold(strings.TrimPrefix(strings.TrimPrefix(origin, "https://"), "http://"), host)
}

func (h *Handler) authCookie(r *http.Request, name, value, path string, ttl time.Duration) *http.Cookie {
	secure := requestIsSecure(r)
	sameSite := http.SameSiteLaxMode
	if secure && crossSiteRequest(r) {
		sameSite = http.SameSiteNoneMode
	}
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Secure:   secure,
		SameSite: sameSite,
	}
	switch {
	case value == "":
		c.MaxAge = -1
		c.Expires = time.Unix(0, 0)
	case ttl > 0:
		c.MaxAge = int(ttl.Seconds())
		c.Expires = time.Now().Add(ttl)
	}
	return c
}

func (h *Handler) setAuthCookies(w http.ResponseWriter, r *http.Request, access, refresh string, claims publicClaims, refreshTTL time.Duration) {
	accessTTL := time.Duration(h.accessTTLMinutes()) * time.Minute
	if refreshTTL <= 0 {
		refreshTTL = h.tokens.DefaultRefreshTTL()
	}

	accessJar := h.authCookie(r, accessCookie, access, "/", accessTTL)
	accessJar.HttpOnly = true
	http.SetCookie(w, accessJar)

	refreshJar := h.authCookie(r, refreshCookie, refresh, authPath, refreshTTL)
	refreshJar.HttpOnly = true
	http.SetCookie(w, refreshJar)

	token := randomToken(16)
	http.SetCookie(w, h.authCookie(r, csrfCookie, token, "/", refreshTTL))

	payload, _ := json.Marshal(claims)
	http.SetCookie(w, h.authCookie(r, userCookie,
		base64.RawURLEncoding.EncodeToString(payload), "/", refreshTTL))

	h.rememberLogin(w, r, savedLogin{
		ID: claims.UserID, Email: claims.Email, Role: claims.Role,
		Refresh: refresh, Used: time.Now().Unix(),
	}, refreshTTL)
}

func (h *Handler) startSession(w http.ResponseWriter, r *http.Request, userID, email, role, access, refresh string, refreshTTL time.Duration) {
	h.setAuthCookies(w, r, access, refresh, publicClaims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Exp:    time.Now().Add(time.Duration(h.accessTTLMinutes()) * time.Minute).Unix(),
	}, refreshTTL)
}

func (h *Handler) clearAuthCookies(w http.ResponseWriter, r *http.Request) {
	for _, name := range []string{accessCookie, csrfCookie, userCookie} {
		http.SetCookie(w, h.authCookie(r, name, "", "/", 0))
	}
	http.SetCookie(w, h.authCookie(r, refreshCookie, "", authPath, 0))
}

func (h *Handler) accessTTLMinutes() int {
	if h.accessTTLMin > 0 {
		return h.accessTTLMin
	}
	return 60
}

func cookieValue(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil || c == nil {
		return ""
	}
	return strings.TrimSpace(c.Value)
}

func csrfValid(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	}
	want := cookieValue(r, csrfCookie)
	got := strings.TrimSpace(r.Header.Get(csrfHeader))
	if want == "" || got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1
}

func (h *Handler) savedLogins(r *http.Request) []savedLogin {
	raw := cookieValue(r, sessionsCookie)
	if raw == "" {
		return nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil
	}
	var list []savedLogin
	if json.Unmarshal(decoded, &list) != nil {
		return nil
	}
	return list
}

func (h *Handler) writeSavedLogins(w http.ResponseWriter, r *http.Request, list []savedLogin, ttl time.Duration) {
	if len(list) > maxSavedLogins {
		list = list[:maxSavedLogins]
	}
	value := ""
	if len(list) > 0 {
		payload, err := json.Marshal(list)
		if err != nil {
			return
		}
		value = base64.RawURLEncoding.EncodeToString(payload)
	}
	jar := h.authCookie(r, sessionsCookie, value, authPath, ttl)
	jar.HttpOnly = true
	http.SetCookie(w, jar)
}

func (h *Handler) rememberLogin(w http.ResponseWriter, r *http.Request, login savedLogin, ttl time.Duration) {
	if login.ID == "" || login.Refresh == "" {
		return
	}
	list := []savedLogin{login}
	for _, item := range h.savedLogins(r) {
		if item.ID == login.ID || item.ID == "" || item.Refresh == "" {
			continue
		}
		list = append(list, item)
	}
	h.writeSavedLogins(w, r, list, ttl)
}

func (h *Handler) forgetLogin(w http.ResponseWriter, r *http.Request, userID string) {
	list := []savedLogin{}
	for _, item := range h.savedLogins(r) {
		if item.ID == userID || item.ID == "" {
			continue
		}
		list = append(list, item)
	}
	h.writeSavedLogins(w, r, list, h.tokens.DefaultRefreshTTL())
}
