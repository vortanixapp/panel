package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
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
		scope := claims.UserID
		if key, isKey := apiKeyFromContext(r.Context()); isKey {
			scope = "key:" + key.ID
		}
		if scope == "" {
			scope = clientIP(r)
		}

		allowed, err := h.cache.AllowWrite(r.Context(), scope, limit, rateWindow)
		if err != nil {
			allowed = allowLocal(scope, limit)
			rateLimitFallbacks.Add(1)
		}

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		if !allowed {
			writeCodedError(w, http.StatusTooManyRequests, "rate_limit_exceeded",
				"Превышена частота запросов: "+strconv.Itoa(limit)+" в минуту по вашему тарифу")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) allowAttempt(ctx context.Context, key string, limit int, window time.Duration) bool {
	allowed, err := h.cache.Allow(ctx, key, limit, window)
	if err != nil {
		return attemptFallback.allow(key, limit, window)
	}
	return allowed
}

func (h *Handler) tooManyAttempts(w http.ResponseWriter, r *http.Request, scope string, limit int, window time.Duration, subjects ...string) bool {
	ip := clientIP(r)
	if ip != "" && !h.allowAttempt(r.Context(), "rl:"+scope+":ip:"+ip, limit, window) {
		writeError(w, http.StatusTooManyRequests, "Слишком много попыток, попробуйте позже")
		return true
	}
	for _, subject := range subjects {
		subject = strings.ToLower(strings.TrimSpace(subject))
		if subject == "" {
			continue
		}
		sum := sha256.Sum256([]byte(subject))
		if !h.allowAttempt(r.Context(), "rl:"+scope+":id:"+hex.EncodeToString(sum[:8]), limit, window) {
			writeError(w, http.StatusTooManyRequests, "Слишком много попыток, попробуйте позже")
			return true
		}
	}
	return false
}

type localWindow struct {
	mu    sync.Mutex
	start time.Time
	count int
}

var (
	localLimiters      sync.Map
	rateLimitFallbacks atomicCounter
	attemptFallback    = &attemptLimiter{attempt: make(map[string]*attemptWindow)}
)

func allowLocal(scope string, limit int) bool {
	v, _ := localLimiters.LoadOrStore(scope, &localWindow{})
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
