package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"
)

const freezeCacheTTL = 5 * time.Second

var freezeState struct {
	mu    sync.Mutex
	on    bool
	until time.Time
}

func (h *Handler) freezeActive(ctx context.Context) bool {
	freezeState.mu.Lock()
	if time.Now().Before(freezeState.until) {
		on := freezeState.on
		freezeState.mu.Unlock()
		return on
	}
	freezeState.mu.Unlock()

	var raw []byte
	on := false
	if h.dbOf(ctx).QueryRow(ctx, `SELECT value FROM core.tenant_settings WHERE key = $1`, settingPanelFreeze).Scan(&raw) == nil {
		_ = json.Unmarshal(raw, &on)
	}
	freezeState.mu.Lock()
	freezeState.on = on
	freezeState.until = time.Now().Add(freezeCacheTTL)
	freezeState.mu.Unlock()
	return on
}

func resetFreezeCache() {
	freezeState.mu.Lock()
	freezeState.until = time.Time{}
	freezeState.mu.Unlock()
}

func freezeExempt(path string) bool {
	switch {
	case strings.Contains(path, "/admin/panel-transfer"):
		return true
	case strings.HasSuffix(path, "/auth/logout"):
		return true
	case strings.Contains(path, "/auth/"):
		return true
	}
	return false
}

func (h *Handler) freezeGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		if freezeExempt(r.URL.Path) || !h.freezeActive(r.Context()) {
			next.ServeHTTP(w, r)
			return
		}
		writeCodedError(w, http.StatusLocked, "panel_frozen",
			"Панель переведена в режим только чтение на время переноса на другой сервер")
	})
}
