package handlers

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

const defaultWriteRPM = 600

const rateWindow = time.Minute

func (h *Handler) tenantWriteRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			next.ServeHTTP(w, r)
			return
		}

		claims, ok := tenantClaims(r.Context())
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		limit := defaultWriteRPM

		allowed, err := h.cache.AllowWrite(r.Context(), claims.TenantID, limit, rateWindow)
		if err != nil {
			allowed = allowLocal(claims.TenantID, limit)
			rateLimitFallbacks.Add(1)
		}

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		if !allowed {
			writeCodedError(w, http.StatusTooManyRequests, "rate_limit_exceeded",
				"превышена частота запросов: "+strconv.Itoa(limit)+" в минуту по вашему тарифу")
			return
		}
		next.ServeHTTP(w, r)
	})
}

type localWindow struct {
	mu    sync.Mutex
	start time.Time
	count int
}

var (
	localLimiters      sync.Map
	rateLimitFallbacks atomicCounter
)

func allowLocal(tenantID string, limit int) bool {
	v, _ := localLimiters.LoadOrStore(tenantID, &localWindow{})
	win := v.(*localWindow)

	win.mu.Lock()
	defer win.mu.Unlock()

	now := time.Now()
	if now.Sub(win.start) >= rateWindow {
		win.start = now
		win.count = 0
	}
	win.count++
	return win.count <= limit
}

type atomicCounter struct {
	mu sync.Mutex
	n  int64
}

func (c *atomicCounter) Add(delta int64) {
	c.mu.Lock()
	c.n += delta
	c.mu.Unlock()
}

func (c *atomicCounter) Value() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}
