package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) ServerInstallLog(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims, ok := h.authorizeServerTab(w, r, id, "logs")
	if !ok {
		return
	}

	limit := int64(200)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 && n <= 1000 {
			limit = n
		}
	}

	lines, err := h.cache.InstallLog(r.Context(), id, limit)
	if err != nil {
		lines = nil
	}
	if lines == nil {
		lines = []string{}
	}

	var provStatus, status string
	_ = h.readerOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(provisioning_status, ''), COALESCE(status, '')
		FROM core.servers WHERE id = $1 AND tenant_id = $2
	`, id, claims.TenantID).Scan(&provStatus, &status)

	resp := map[string]any{"lines": lines}
	if progress := h.deriveProvisioningProgress(r.Context(), id, provStatus, status); progress != nil {
		resp["progress"] = progress
	}
	writeJSON(w, http.StatusOK, resp)
}
