package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vortanixapp/panel/internal/api/jobwake"
	"github.com/vortanixapp/panel/internal/api/relay"
)

func (h *Handler) UpdateServerGame(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	ctx := r.Context()

	spec := h.resolveInstallSpec(ctx, serverID)
	if spec == nil {
		writeError(w, http.StatusBadRequest,
			"Для версии игры не указан источник файлов — обновлять нечего")
		return
	}

	var name, gameID, prevStatus, prevRuntime string
	var limitsRaw []byte
	var primaryPort int
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT name, game_id, COALESCE(limits, '{}'::jsonb), COALESCE(primary_port, 0),
		       COALESCE(status, ''), COALESCE(runtime_status, '')
		FROM core.servers WHERE id = $1
	`, serverID).Scan(&name, &gameID, &limitsRaw, &primaryPort, &prevStatus, &prevRuntime); err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	limits := map[string]any{}
	_ = json.Unmarshal(limitsRaw, &limits)

	payload := map[string]any{
		"install": spec,
		"name":    name,
		"game_id": gameID,
		"limits":  limits,
	}
	if primaryPort > 0 {
		payload["primary_port"] = primaryPort
	}
	if bindIP := h.serverBindIP(ctx, serverID); bindIP != "" {
		payload["bind_ip"] = bindIP
	}
	if img := h.resolveDockerImage(ctx, serverID); img != "" {
		payload["docker_image"] = img
	}

	result, ok := h.agentCommandForServer(w, r, serverID, "update", payload)
	if !ok {
		return
	}
	if _, started := result["status"]; !started {
		h.restoreServerStatus(ctx, serverID, prevStatus, prevRuntime)
		writeError(w, http.StatusConflict,
			"Агент на ноде устарел и не умеет обновлять игру — обновите его на странице ноды")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "server.update", "server:"+serverID, nil)
	writeJSON(w, http.StatusOK, map[string]any{"status": "updating", "result": result})
}

func (h *Handler) ReinstallServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	if !h.authorizeServerAction(w, r, claims, serverID, "reinstall") {
		return
	}
	cmdID, status, err := h.startReinstall(r.Context(), serverID)
	if err != nil {
		writeError(w, status, err.Error())
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "server.reinstall", "server:"+serverID, nil)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "reinstalling", "command_id": cmdID})
}

func (h *Handler) startReinstall(ctx context.Context, serverID string) (string, int, error) {
	var nodeID string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT node_id::text FROM core.servers WHERE id = $1
	`, serverID).Scan(&nodeID); err != nil {
		return "", http.StatusNotFound, fmt.Errorf("not found")
	}
	cmdID := uuid.NewString()
	if err := h.relay.SendCommand(ctx, nodeID, relay.CommandRequest{
		CommandID: cmdID,
		Action:    "destroy",
		ServerID:  serverID,
		Payload:   map[string]any{"wipe": true},
	}); err != nil {
		return "", http.StatusBadGateway, fmt.Errorf("agent unreachable: %w", err)
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.servers
		SET status = 'reinstalling', provisioning_status = 'provisioning', provisioning_error = NULL
		WHERE id = $1
	`, serverID)

	_ = h.cache.SetServerStatus(ctx, serverID, "reinstalling")
	_ = h.cache.InvalidateTenantServers(ctx)

	_ = h.cache.Delete(ctx, "srv:"+serverID+":provisioning_progress", "install:log:"+serverID)

	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.jobs ( type, status, payload)
		VALUES ( 'provision_server', 'pending', $1::jsonb)
	`, mustJSON(map[string]string{"server_id": serverID}))
	jobwake.Notify("provision_server")
	return cmdID, 0, nil
}
