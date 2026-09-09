package handlers

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) deleteTenantRow(
	w http.ResponseWriter, r *http.Request, table, action, resourcePrefix string,
) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	tag, err := h.dbOf(r.Context()).Exec(r.Context(),
		fmt.Sprintf(`DELETE FROM %s WHERE id = $1 AND tenant_id = $2`, table),
		id, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, action, resourcePrefix+":"+id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) DeleteNews(w http.ResponseWriter, r *http.Request) {
	h.deleteTenantRow(w, r, "core.news", "news.delete", "news")
}

func (h *Handler) DeleteMailing(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	var status string
	if err := h.dbOf(r.Context()).QueryRow(r.Context(),
		`SELECT status FROM core.mailings WHERE id = $1 AND tenant_id = $2`,
		id, claims.TenantID).Scan(&status); err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if status == "sending" {
		writeError(w, http.StatusConflict, "рассылка отправляется — дождитесь завершения")
		return
	}
	h.deleteTenantRow(w, r, "core.mailings", "mailing.delete", "mailing")
}

func (h *Handler) DeletePlugin(w http.ResponseWriter, r *http.Request) {
	h.deleteCatalogItem(w, r, "core.plugins", "core.server_plugins", "plugin_id", "plugin.delete", "plugin")
}

func (h *Handler) DeleteMap(w http.ResponseWriter, r *http.Request) {
	h.deleteCatalogItem(w, r, "core.maps", "core.server_maps", "map_id", "map.delete", "map")
}

func (h *Handler) deleteCatalogItem(
	w http.ResponseWriter, r *http.Request,
	table, usageTable, usageColumn, action, resourcePrefix string,
) {
	if _, ok := tenantClaims(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	if r.URL.Query().Get("force") != "1" {
		var used int
		_ = h.dbOf(r.Context()).QueryRow(r.Context(),
			fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s = $1 AND installed = true`, usageTable, usageColumn),
			id).Scan(&used)
		if used > 0 {
			writeJSON(w, http.StatusConflict, map[string]any{
				"error":   fmt.Sprintf("установлен на %d сервер(ах) — сначала удалите оттуда или повторите с force=1", used),
				"servers": used,
			})
			return
		}
	}
	h.deleteTenantRow(w, r, table, action, resourcePrefix)
}
