package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vortanixapp/panel/pkg/gamecatalog"
)

func (h *Handler) serverRuntimeSelector(ctx context.Context, serverID string) (gamecatalog.RuntimeSelector, bool) {
	var gameID string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(game_id, '') FROM core.servers WHERE id = $1
	`, serverID).Scan(&gameID); err != nil {
		return gamecatalog.RuntimeSelector{}, false
	}
	return gamecatalog.RuntimeSelectorOf(gameID)
}

func runtimeVersionAllowed(sel gamecatalog.RuntimeSelector, version string) bool {
	for _, v := range sel.Versions {
		if v == version {
			return true
		}
	}
	return false
}

func (h *Handler) GetServerRuntime(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, serverID, "files_read"); !ok {
		return
	}
	ctx := r.Context()
	sel, ok := h.serverRuntimeSelector(ctx, serverID)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"supported": false})
		return
	}
	current := ""
	if nodeID, err := h.serverNodeID(ctx, serverID); err == nil {
		result, cmdErr := h.agentCommand(ctx, nodeID, serverID, "files_read",
			map[string]any{"path": "/" + sel.File})
		if cmdErr == nil {
			content, _ := result["content"].(string)
			current = strings.TrimSpace(content)
		}
	}
	if !runtimeVersionAllowed(sel, current) {
		current = ""
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"supported": true,
		"kind":      sel.Kind,
		"versions":  sel.Versions,
		"current":   current,
	})
}

func (h *Handler) SetServerRuntime(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	claims, ok := h.authorizeServerTab(w, r, serverID, "files_write")
	if !ok {
		return
	}
	ctx := r.Context()
	sel, ok := h.serverRuntimeSelector(ctx, serverID)
	if !ok {
		writeError(w, http.StatusBadRequest, "Для этой игры версия среды не меняется")
		return
	}
	var body struct {
		Version string `json:"version"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	version := strings.TrimSpace(body.Version)
	if version != "" && !runtimeVersionAllowed(sel, version) {
		writeError(w, http.StatusBadRequest, "Эта версия среды недоступна")
		return
	}

	if version == "" {
		if _, ok := h.agentCommandForServer(w, r, serverID, "files_delete",
			map[string]any{"path": "/" + sel.File}); !ok {
			return
		}
	} else {
		if _, ok := h.agentCommandForServer(w, r, serverID, "files_write",
			map[string]any{"path": "/" + sel.File, "content": version + "\n"}); !ok {
			return
		}
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "server.runtime_version", "server:"+serverID,
		map[string]any{"kind": sel.Kind, "version": version})
	writeJSON(w, http.StatusOK, map[string]any{"status": "saved", "kind": sel.Kind, "version": version})
}
