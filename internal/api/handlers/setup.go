package handlers

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

func (h *Handler) SetupStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"bootstrapped":     h.tenantExists(r),
		"suggested_domain": hostOnly(r.Host),
	})
}

func (h *Handler) tenantExists(r *http.Request) bool {
	var exists bool
	if err := h.db.QueryRow(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM core.users WHERE role = 'owner')`,
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
