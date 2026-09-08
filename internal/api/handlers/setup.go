package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/vortanix/vortanix/internal/api/licensejwt"
	"github.com/vortanix/vortanix/internal/api/licensestate"
)

type setupActivateRequest struct {
	LicenseKey string `json:"license_key"`
	Domain     string `json:"domain"`
}

func (h *Handler) SetupActivate(w http.ResponseWriter, r *http.Request) {
	if !setupRateLimiter.allow(clientIPOf(r)) {
		writeError(w, http.StatusTooManyRequests, "слишком много попыток, подождите минуту")
		return
	}
	if h.licenseOps == nil {
		writeError(w, http.StatusServiceUnavailable, "подсистема лицензий не готова")
		return
	}

	if h.tenantExists(r) {
		writeError(w, http.StatusConflict, "панель уже настроена")
		return
	}

	var req setupActivateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.LicenseKey = strings.TrimSpace(req.LicenseKey)
	if req.LicenseKey == "" {
		writeError(w, http.StatusBadRequest, "license_key обязателен")
		return
	}

	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	if domain == "" {
		domain = hostOnly(r.Host)
	}

	if err := h.licenseOps.ActivateIn(r.Context(), h.dbOf(r.Context()), req.LicenseKey, domain); err != nil {
		writeCodedError(w, http.StatusForbidden, "license_invalid", err.Error())
		return
	}

	st := h.licenseFor(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{
		"plan":               st.Plan,
		"tenant_slug":        st.TenantSlug,
		"key_hint":           st.KeyHint,
		"limits":             st.Limits,
		"license_expires_at": st.LicenseExpiresAt,
		"domain":             domain,
	})
}

func (h *Handler) SetupStatus(w http.ResponseWriter, r *http.Request) {
	st := h.licenseFor(r.Context())

	writeJSON(w, http.StatusOK, map[string]any{
		"bootstrapped":     h.tenantExists(r),
		"activated":        st.Activated || st.Legacy,
		"plan":             st.Plan,
		"key_hint":         st.KeyHint,
		"limits":           st.Limits,
		"domain":           st.Domain,
		"suggested_domain": hostOnly(r.Host),
	})
}

// core-api обслуживает панели всех клиентов сразу, поэтому «настроена ли
// панель» — вопрос про конкретного арендатора. Прежняя проверка спрашивала,
// есть ли в базе хоть один арендатор, и после первой же активации отвечала
// «да» всем остальным: их панели запирались навсегда.
func (h *Handler) tenantExists(r *http.Request) bool {
	return h.tenantExistsBySlug(r, strings.TrimSpace(r.Header.Get("X-Tenant-Slug")))
}

func (h *Handler) tenantExistsBySlug(r *http.Request, slug string) bool {
	if slug == "" {
		return false
	}
	var exists bool
	if err := h.dbOf(r.Context()).QueryRow(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM core.tenants WHERE slug = $1)`, slug,
	).Scan(&exists); err != nil {
		return true
	}
	return exists
}

func hostOnly(host string) string {
	if idx := strings.LastIndexByte(host, ':'); idx > 0 && !strings.Contains(host[idx:], "]") {
		host = host[:idx]
	}
	return strings.ToLower(strings.Trim(host, "[]"))
}

func clientIPOf(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		if idx := strings.IndexByte(fwd, ','); idx > 0 {
			return strings.TrimSpace(fwd[:idx])
		}
		return strings.TrimSpace(fwd)
	}
	return hostOnly(r.RemoteAddr)
}

var setupRateLimiter = &attemptLimiter{
	limit:   5,
	window:  time.Minute,
	attempt: make(map[string]*attemptWindow),
}

type attemptWindow struct {
	start time.Time
	count int
}

type attemptLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	attempt map[string]*attemptWindow
}

func (l *attemptLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	win, ok := l.attempt[key]
	if !ok || now.Sub(win.start) >= l.window {
		l.attempt[key] = &attemptWindow{start: now, count: 1}
		l.sweep(now)
		return true
	}

	win.count++
	return win.count <= l.limit
}

func (l *attemptLimiter) sweep(now time.Time) {
	if len(l.attempt) < 1000 {
		return
	}
	for key, win := range l.attempt {
		if now.Sub(win.start) >= l.window {
			delete(l.attempt, key)
		}
	}
}

func (h *Handler) bootstrapClaims(w http.ResponseWriter, r *http.Request, req bootstrapRequest) (*licensejwt.Claims, error) {
	if key := strings.TrimSpace(req.LicenseKey); key != "" {
		if h.licenseOps == nil {
			writeError(w, http.StatusServiceUnavailable, "подсистема лицензий не готова")
			return nil, errLicenseUnavailable
		}
		domain := strings.ToLower(strings.TrimSpace(req.Domain))
		if domain == "" {
			domain = hostOnly(r.Host)
		}
		if err := h.licenseOps.ActivateIn(r.Context(), h.dbOf(r.Context()), key, domain); err != nil {
			writeCodedError(w, http.StatusForbidden, "license_invalid", err.Error())
			return nil, errLicenseUnavailable
		}
	}

	if req.LicenseToken != "" {
		claims, err := h.license.Parse(req.LicenseToken)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid license token")
			return nil, errLicenseUnavailable
		}
		return claims, nil
	}

	row, err := licensestate.LoadRow(r.Context(), h.dbOf(r.Context()))
	if err != nil || row.LicenseToken == "" {
		writeError(w, http.StatusBadRequest, "сначала активируйте лицензию")
		return nil, errLicenseUnavailable
	}
	claims, err := h.license.Parse(row.LicenseToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "сохранённый лицензионный токен недействителен")
		return nil, errLicenseUnavailable
	}
	if claims.Status != "" && claims.Status != "active" {
		writeCodedError(w, http.StatusForbidden, "license_"+claims.Status,
			"лицензия не активна: "+claims.Status)
		return nil, errLicenseUnavailable
	}
	return claims, nil
}

var errLicenseUnavailable = errors.New("лицензия недоступна")
