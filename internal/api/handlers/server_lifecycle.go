package handlers

import (
	"encoding/json"
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

	spec := h.resolveInstallSpec(ctx, claims.TenantID, serverID)
	if spec == nil {
		writeError(w, http.StatusBadRequest,
			"для версии игры не указан источник файлов — обновлять нечего")
		return
	}

	var name, gameID, prevStatus, prevRuntime string
	var limitsRaw []byte
	var primaryPort int
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT name, game_id, COALESCE(limits, '{}'::jsonb), COALESCE(primary_port, 0),
		       COALESCE(status, ''), COALESCE(runtime_status, '')
		FROM core.servers WHERE id = $1 AND tenant_id = $2
	`, serverID, claims.TenantID).Scan(&name, &gameID, &limitsRaw, &primaryPort, &prevStatus, &prevRuntime); err != nil {
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
	if bindIP := h.serverBindIP(ctx, claims.TenantID, serverID); bindIP != "" {
		payload["bind_ip"] = bindIP
	}
	if img := h.resolveDockerImage(ctx, claims.TenantID, serverID); img != "" {
		payload["docker_image"] = img
	}

	result, ok := h.agentCommandForServer(w, r, serverID, "update", payload)
	if !ok {
		return
	}
	if _, started := result["status"]; !started {
		h.restoreServerStatus(ctx, claims.TenantID, serverID, prevStatus, prevRuntime)
		writeError(w, http.StatusConflict,
			"агент на ноде устарел и не умеет обновлять игру — обновите его на странице ноды")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.TenantID, claims.UserID, "server.update", serverID, nil)
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
	var nodeID string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT node_id::text FROM core.servers WHERE id = $1 AND tenant_id = $2
	`, serverID, claims.TenantID).Scan(&nodeID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	cmdID := uuid.NewString()
	if err := h.relay.SendCommand(r.Context(), nodeID, relay.CommandRequest{
		CommandID: cmdID,
		Action:    "destroy",
		ServerID:  serverID,
		Payload:   map[string]any{"wipe": true},
	}); err != nil {
		writeError(w, http.StatusBadGateway, "agent unreachable: "+err.Error())
		return
	}
	ctx := r.Context()
	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.servers
		SET status = 'reinstalling', provisioning_status = 'provisioning', provisioning_error = NULL
		WHERE id = $1
	`, serverID)

	_ = h.cache.SetServerStatus(ctx, serverID, "reinstalling")
	_ = h.cache.InvalidateTenantServers(ctx, claims.TenantID)

	_ = h.cache.Delete(ctx, "srv:"+serverID+":provisioning_progress", "install:log:"+serverID)

	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.jobs (tenant_id, type, status, payload)
		VALUES ($1, 'provision_server', 'pending', $2::jsonb)
	`, claims.TenantID, mustJSON(map[string]string{"server_id": serverID}))
	jobwake.Notify("provision_server")
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "server.reinstall", serverID, nil)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "reinstalling", "command_id": cmdID})
}
