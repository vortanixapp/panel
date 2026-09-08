package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) ServerConsoleCommand(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	var body struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Command) == "" {
		writeError(w, http.StatusBadRequest, "command required")
		return
	}
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var gameID string
	if err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(game_id, '') FROM core.servers WHERE id = $1 AND tenant_id = $2
	`, serverID, claims.TenantID).Scan(&gameID); err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	result, ok := h.agentCommandForServer(w, r, serverID, "console_command", map[string]any{
		"command": body.Command,
		"game_id": gameID,
	})
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "result": result})
}

func (h *Handler) ServerMapsFolderList(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	// Относительно корня данных сервера: агент сам подставит /data. С полным
	// путём список уходил в /data/data/cstrike/maps и всегда был пуст.
	path := "/cstrike/maps"
	result, ok := h.agentCommandForServer(w, r, serverID, "files_list", map[string]any{"path": path})
	if !ok {
		return
	}
	maps := []string{}
	if files, exists := result["files"]; exists {
		if arr, ok := files.([]any); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]any); ok {
					name, _ := m["name"].(string)
					isDir, _ := m["is_dir"].(bool)
					if !isDir && strings.HasSuffix(strings.ToLower(name), ".bsp") {
						maps = append(maps, strings.TrimSuffix(strings.TrimSuffix(name, ".bsp"), ".BSP"))
					}
				}
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"maps": maps})
}

func (h *Handler) ServerMapChangeByName(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	var body struct {
		Map string `json:"map"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Map) == "" {
		writeError(w, http.StatusBadRequest, "map required")
		return
	}
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	mapName := strings.TrimSpace(body.Map)
	_, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.servers SET config = jsonb_set(
			COALESCE(config, '{}'::jsonb),
			'{startup_params}',
			to_jsonb('map ' || $3::text)
		)
		WHERE id = $1 AND tenant_id = $2
	`, serverID, claims.TenantID, mapName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	var runtime, gameID string
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(runtime_status, ''), COALESCE(game_id, '') FROM core.servers WHERE id = $1
	`, serverID).Scan(&runtime, &gameID)
	if runtime == "running" {
		if _, ok := h.agentCommandForServer(w, r, serverID, "console_command", map[string]any{
			"command": "changelevel " + mapName,
			"game_id": gameID,
		}); !ok {
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "map": mapName})
}
