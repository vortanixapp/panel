package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vortanix/vortanix/internal/metrics/store"
)

type Handler struct {
	store  *store.Store
	secret string
}

func New(st *store.Store, secret string) *Handler {
	return &Handler{store: st, secret: secret}
}

func secretEqual(got, want string) bool {
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/health", h.Health)
	r.Post("/internal/v1/ingest", h.Ingest)
	r.Get("/internal/v1/servers/{serverID}/metrics", h.GetServerMetrics)
	return r
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type ingestRequest struct {
	TenantID   string  `json:"tenant_id"`
	ServerID   string  `json:"server_id"`
	CPUPct     float64 `json:"cpu_pct"`
	MemUsedMB  int     `json:"mem_used_mb"`
	MemLimitMB int     `json:"mem_limit_mb"`
	TS         *int64  `json:"ts,omitempty"`
}

func (h *Handler) Ingest(w http.ResponseWriter, r *http.Request) {
	if !secretEqual(r.Header.Get("X-Internal-Secret"), h.secret) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.TenantID == "" || req.ServerID == "" {
		writeError(w, http.StatusBadRequest, "tenant_id and server_id required")
		return
	}

	ctx := r.Context()
	ok, err := h.store.ValidateServerTenant(ctx, req.ServerID, req.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "validation failed")
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	point := store.MetricPoint{
		TenantID:   req.TenantID,
		ServerID:   req.ServerID,
		CPUPct:     req.CPUPct,
		MemUsedMB:  req.MemUsedMB,
		MemLimitMB: req.MemLimitMB,
	}
	if req.TS != nil {
		point.TS = time.Unix(*req.TS, 0).UTC()
	}

	if err := h.store.StoreMetricPoint(ctx, point); err != nil {
		writeError(w, http.StatusInternalServerError, "store failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GetServerMetrics(w http.ResponseWriter, r *http.Request) {
	if !secretEqual(r.Header.Get("X-Internal-Secret"), h.secret) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	serverID := chi.URLParam(r, "serverID")
	limit := int64(60)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			limit = n
		}
	}
	hours := 24
	if v := r.URL.Query().Get("hours"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			hours = n
		}
	}

	ctx := r.Context()
	exists, err := h.store.ServerExists(ctx, serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	if !exists {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	points, err := h.store.GetMetrics(ctx, serverID, limit, hours)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "metrics unavailable")
		return
	}
	if points == nil {
		points = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"points": points})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
