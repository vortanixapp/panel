package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vortanixapp/panel/pkg/gamecatalog"
)

func (h *Handler) serverRuntimeSelector(ctx context.Context, serverID string) (gamecatalog.RuntimeSelector, []runtimeVersionEntry, bool) {
	var gameID string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(game_id, '') FROM core.servers WHERE id = $1
	`, serverID).Scan(&gameID); err != nil {
		return gamecatalog.RuntimeSelector{}, nil, false
	}
	sel, ok := gamecatalog.RuntimeSelectorOf(gameID)
	if !ok {
		return gamecatalog.RuntimeSelector{}, nil, false
	}
	var raw []byte
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(meta, '{}'::jsonb) FROM core.games WHERE slug = $1
	`, gameID).Scan(&raw); err != nil {
		return sel, defaultRuntimeVersions(sel), true
	}
	return sel, runtimeVersionsOf(parseMetaMap(raw), sel), true
}

func (h *Handler) GetServerRuntime(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, serverID, "files_read"); !ok {
		return
	}
	ctx := r.Context()
	sel, list, ok := h.serverRuntimeSelector(ctx, serverID)
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
	if _, ok := findRuntimeVersion(list, current); !ok {
		current = ""
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"supported": true,
		"kind":      sel.Kind,
		"versions":  enabledRuntimeVersions(list),
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
	sel, list, ok := h.serverRuntimeSelector(ctx, serverID)
	if !ok {
		writeError(w, http.StatusBadRequest, "Для этой игры версия среды не меняется")
		return
	}
	var body struct {
		Version string `json:"version"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	version := strings.TrimSpace(body.Version)
	entry := runtimeVersionEntry{}
	if version != "" {
		found, ok := findRuntimeVersion(list, version)
		if !ok {
			writeError(w, http.StatusBadRequest, "Эта версия среды недоступна")
			return
		}
		entry = found
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
	if sel.URLFile != "" {
		if entry.URL == "" {
			if _, ok := h.agentCommandForServer(w, r, serverID, "files_delete",
				map[string]any{"path": "/" + sel.URLFile}); !ok {
				return
			}
		} else if _, ok := h.agentCommandForServer(w, r, serverID, "files_write",
			map[string]any{"path": "/" + sel.URLFile, "content": entry.URL + "\n"}); !ok {
			return
		}
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "server.runtime_version", "server:"+serverID,
		map[string]any{"kind": sel.Kind, "version": version})
	writeJSON(w, http.StatusOK, map[string]any{"status": "saved", "kind": sel.Kind, "version": version})
}
