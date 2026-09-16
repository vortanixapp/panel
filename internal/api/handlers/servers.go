package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/internal/api/relay"
	"github.com/vortanixapp/panel/pkg/gamecatalog"
	"github.com/vortanixapp/panel/pkg/portalloc"
)

type serverResponse struct {
	ID        string         `json:"id"`
	NodeID    string         `json:"node_id"`
	GameID    string         `json:"game_id"`
	Name      string         `json:"name"`
	Status    string         `json:"status"`
	Config    map[string]any `json:"config"`
	Limits    map[string]any `json:"limits"`
	CreatedAt time.Time      `json:"created_at"`
}

func (h *Handler) ListServers(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	cacheKey := "panel:servers:" + claims.UserID
	var cached []serverResponse
	if hit, err := h.cache.GetJSON(ctx, cacheKey, &cached); err == nil && hit {
		writeJSON(w, http.StatusOK, map[string]any{"servers": cached})
		return
	}

	query := `
		SELECT id::text, node_id::text, game_id, name, status, config, limits, created_at
		FROM core.servers WHERE true`
	args := []any{claims.UserID}
	query += ` AND (user_id = $1 OR user_id IS NULL) ORDER BY created_at DESC`
	rows, err := h.readerOf(ctx).Query(ctx, query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	servers := make([]serverResponse, 0)
	for rows.Next() {
		var s serverResponse
		var cfg, lim []byte
		if err := rows.Scan(&s.ID, &s.NodeID, &s.GameID, &s.Name, &s.Status, &cfg, &lim, &s.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "scan error")
			return
		}
		_ = json.Unmarshal(cfg, &s.Config)
		_ = json.Unmarshal(lim, &s.Limits)
		if s.Config == nil {
			s.Config = map[string]any{}
		}
		if s.Limits == nil {
			s.Limits = map[string]any{}
		}
		if live, ok := h.cache.GetServerStatus(ctx, s.ID); ok {
			s.Status = resolveEffectiveStatus(s.Status, "", &live)
		}
		servers = append(servers, s)
	}
	_ = h.cache.SetJSON(ctx, cacheKey, servers, 5*time.Second)
	writeJSON(w, http.StatusOK, map[string]any{"servers": servers})
}

type createServerRequest struct {
	NodeID string `json:"node_id"`
	Name   string `json:"name"`
	GameID string `json:"game_id"`
}

func (h *Handler) CreateServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "Серверы создаются через аренду")
		return
	}
	var req createServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.NodeID == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "node_id and name are required")
		return
	}
	if req.GameID == "" {
		req.GameID = "test"
	}

	ctx := r.Context()

	var nodeExists bool
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM core.nodes WHERE id = $1)
	`, req.NodeID).Scan(&nodeExists); err != nil || !nodeExists {
		writeError(w, http.StatusBadRequest, "node not found")
		return
	}
	if h.nodeInMaintenance(ctx, req.NodeID) {
		writeError(w, http.StatusConflict, "Нода на техническом обслуживании")
		return
	}
	if reason := h.nodeCapacityReason(ctx, req.NodeID, req.GameID); reason != "" {
		writeError(w, http.StatusConflict, reason)
		return
	}

	limits := gamecatalog.DefaultLimits(req.GameID)
	limitsJSON, _ := json.Marshal(limits)
	var id string
	err := h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.servers ( node_id, game_id, name, limits, user_id, config)
		VALUES ( $1, $2, $3, $4::jsonb, $5, jsonb_build_object('startup_params', $6::text))
		RETURNING id::text
	`, req.NodeID, req.GameID, req.Name, limitsJSON, claims.UserID,
		h.defaultStartupParams(ctx, req.GameID)).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create server")
		return
	}

	if req.GameID != "test" {
		if _, portErr := portalloc.Assign(ctx, h.dbOf(ctx), req.NodeID, id, req.GameID); portErr != nil {
			log.Printf("servers: не удалось выдать порт серверу %s (%s): %v", id, req.GameID, portErr)
		}
	}

	_ = h.cache.InvalidateTenantServers(ctx)
	audit(ctx, h.dbOf(ctx), claims.UserID, "server.create", "server:"+id, map[string]any{"name": req.Name})

	writeJSON(w, http.StatusCreated, serverResponse{
		ID: id, NodeID: req.NodeID, GameID: req.GameID, Name: req.Name,
		Status: "stopped", Limits: limits,
		CreatedAt: time.Now(),
	})
}

func (h *Handler) GetServer(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, id, "settings_read"); !ok {
		return
	}
	ctx := r.Context()
	var s serverResponse
	var cfg, lim []byte
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT id::text, node_id::text, game_id, name, status, config, limits, created_at
		FROM core.servers WHERE id = $1
	`, id).Scan(&s.ID, &s.NodeID, &s.GameID, &s.Name, &s.Status, &cfg, &lim, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	_ = json.Unmarshal(cfg, &s.Config)
	_ = json.Unmarshal(lim, &s.Limits)
	if live, ok := h.cache.GetServerStatus(ctx, s.ID); ok {
		s.Status = resolveEffectiveStatus(s.Status, "", &live)
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *Handler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	if !h.authorizeServerAction(w, r, claims, id, "delete") {
		return
	}
	if !isStaffRole(claims.Role) && h.refuseWHMCSBilled(ctx, w, id) {
		return
	}

	var nodeID string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT node_id::text FROM core.servers WHERE id = $1
	`, id).Scan(&nodeID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	deferred, err := h.destroyServer(r, id, nodeID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "server.delete", "server:"+id, map[string]any{
		"node_cleaned":     !deferred,
		"cleanup_deferred": deferred,
	})
	res := map[string]string{"status": "deleted"}
	if deferred {
		res["cleanup"] = "deferred"
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) destroyServer(r *http.Request, id, nodeID string) (bool, error) {
	ctx := r.Context()
	agentErr := h.relay.SendCommand(ctx, nodeID, relay.CommandRequest{
		CommandID: uuid.NewString(),
		Action:    "destroy",
		ServerID:  id,
		Payload:   map[string]any{"wipe": true},
	})
	deferred := false
	if agentErr != nil {
		payload, _ := json.Marshal(map[string]string{"server_id": id, "node_id": nodeID})
		if _, err := h.dbOf(ctx).Exec(ctx, `
			INSERT INTO core.jobs (type, status, payload) VALUES ('server_destroy', 'pending', $1::jsonb)
		`, payload); err != nil {
			log.Printf("удаление сервера %s: отложенная очистка ноды не поставлена: %v", id, err)
			return false, errors.New("нода недоступна, а очистку на потом поставить не удалось")
		}
		deferred = true
	}

	h.enqueueFtpCleanup(r, id, nodeID)

	if _, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.ip_pools SET status = 'free', server_id = NULL
		WHERE server_id = $1::uuid
	`, id); err != nil {
		log.Printf("удаление сервера %s: адрес не возвращён в пул: %v", id, err)
	}

	tag, err := h.dbOf(ctx).Exec(ctx, `DELETE FROM core.servers WHERE id = $1`, id)
	if err != nil || tag.RowsAffected() == 0 {
		return deferred, pgx.ErrNoRows
	}
	_ = h.cache.InvalidateTenantServers(ctx)
	return deferred, nil
}

type powerRequest struct {
	Action string `json:"action"`
}

var (
	errServerNotFound   = errors.New("server not found")
	errServerExpired    = errors.New("rent expired")
	errAgentUnreachable = errors.New("agent unreachable")
)

func (h *Handler) sendServerPower(ctx context.Context, actorID, id, action string) (string, string, error) {
	var nodeID, name, gameID, status string
	var limits []byte
	var primaryPort int
	var expired bool
	var startupParams string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT node_id::text, name, game_id, status, limits, COALESCE(primary_port, 0),
		       (expires_at IS NOT NULL AND expires_at < now()) AS expired,
		       COALESCE(config->>'startup_params', '')
		FROM core.servers WHERE id = $1
	`, id).Scan(&nodeID, &name, &gameID, &status, &limits, &primaryPort, &expired, &startupParams)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", errServerNotFound
		}
		return "", "", err
	}

	if expired && (action == "start" || action == "restart") {
		return "", "", errServerExpired
	}

	transient := map[string]string{"start": "starting", "stop": "stopping", "restart": "starting", "kill": "stopping"}[action]
	_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.servers SET status = $1 WHERE id = $2`, transient, id)
	_ = h.cache.SetServerStatus(ctx, id, transient)
	_ = h.cache.InvalidateTenantServers(ctx)

	var lim map[string]any
	_ = json.Unmarshal(limits, &lim)
	dockerImage := h.resolveDockerImage(ctx, id)
	payload := map[string]any{
		"power_action": action,
		"name":         name,
		"game_id":      gameID,
		"limits":       lim,
	}
	if primaryPort > 0 && (action == "start" || action == "restart") {
		payload["primary_port"] = primaryPort
	}
	if action == "start" || action == "restart" {
		payload["startup_params"] = startupParams
	}
	if bindIP := h.serverBindIP(ctx, id); bindIP != "" {
		payload["bind_ip"] = bindIP
	}
	if dockerImage != "" {
		payload["docker_image"] = dockerImage
	}
	if action == "start" || action == "restart" {
		if spec := h.resolveInstallSpec(ctx, id); spec != nil {
			payload["install"] = spec
		}
	}
	cmdID := uuid.NewString()
	if err := h.relay.SendCommand(ctx, nodeID, relay.CommandRequest{
		CommandID: cmdID,
		Action:    "power",
		ServerID:  id,
		Payload:   payload,
	}); err != nil {
		return "", "", fmt.Errorf("%w: %v", errAgentUnreachable, err)
	}

	audit(ctx, h.dbOf(ctx), actorID, "server.power", "server:"+id, map[string]any{"action": action})
	return transient, cmdID, nil
}

func (h *Handler) PowerServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	var req powerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	action := req.Action
	switch action {
	case "start", "stop", "restart", "kill":
	default:
		writeError(w, http.StatusBadRequest, "action must be start, stop, restart or kill")
		return
	}

	if !h.authorizeServerAction(w, r, claims, id, "power_"+action) {
		return
	}

	status, cmdID, err := h.sendServerPower(r.Context(), claims.UserID, id, action)
	if err != nil {
		switch {
		case errors.Is(err, errServerNotFound):
			writeError(w, http.StatusNotFound, "server not found")
		case errors.Is(err, errServerExpired):
			writeError(w, http.StatusPaymentRequired, "Оплаченный период закончился — продлите аренду, чтобы запустить сервер")
		case errors.Is(err, errAgentUnreachable):
			writeError(w, http.StatusBadGateway, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "database error")
		}
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"server_id":  id,
		"action":     action,
		"status":     status,
		"command_id": cmdID,
	})
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
