package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/vortanixapp/panel/pkg/protocol"
)

func (h *Handler) loadCronJobs(ctx context.Context, serverID string) ([]map[string]any, error) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT id::text, schedule, command, enabled FROM core.server_cron_jobs WHERE server_id = $1 ORDER BY created_at
	`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]map[string]any, 0)
	for rows.Next() {
		var id, schedule, command string
		var enabled bool
		if err := rows.Scan(&id, &schedule, &command, &enabled); err != nil {
			return nil, err
		}
		list = append(list, map[string]any{
			"id": id, "schedule": schedule, "command": command, "enabled": enabled,
		})
	}
	return list, rows.Err()
}

func (h *Handler) loadFirewallRules(ctx context.Context, serverID string) ([]map[string]any, error) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT id::text, protocol, port_from, port_to, enabled
		FROM core.server_firewall_rules WHERE server_id = $1 ORDER BY created_at
	`, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]map[string]any, 0)
	for rows.Next() {
		var id, protocol string
		var portFrom int
		var portTo *int
		var enabled bool
		if err := rows.Scan(&id, &protocol, &portFrom, &portTo, &enabled); err != nil {
			return nil, err
		}
		item := map[string]any{
			"id": id, "protocol": protocol, "port_from": portFrom, "enabled": enabled,
		}
		if portTo != nil {
			item["port_to"] = *portTo
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// loadExtraPorts — дополнительные порты сервера, заведённые во вкладке «Порты».
//
// Основной порт игры сюда не входит: он публикуется раскладкой каталога при
// старте контейнера и в этой таблице не хранится.
func (h *Handler) loadExtraPorts(ctx context.Context, tenantID, serverID string) ([]map[string]any, error) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT port, protocol, COALESCE(purpose, '')
		FROM core.server_ports
		WHERE server_id = $1 AND tenant_id = $2
		ORDER BY port
	`, serverID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]map[string]any, 0)
	for rows.Next() {
		var port int
		var proto, purpose string
		if err := rows.Scan(&port, &proto, &purpose); err != nil {
			return nil, err
		}
		list = append(list, map[string]any{
			"port": port, "protocol": proto, "purpose": purpose,
		})
	}
	return list, rows.Err()
}

// pushPortsState отдаёт агенту список портов вместе с тем, что нужно для
// пересоздания контейнера: публикация портов задаётся только при его создании.
func (h *Handler) pushPortsState(ctx context.Context, tenantID, nodeID, serverID string) error {
	ports, err := h.loadExtraPorts(ctx, tenantID, serverID)
	if err != nil {
		return err
	}

	// Статус запоминаем до отправки: агент прежних версий не знает команды
	// ports_sync и обработчика неизвестных команд у него нет — он отвечает
	// «выполнено» и заодно сообщает, что сервер остановлен. Работающий сервер
	// после этого числился бы остановленным, поэтому статус придётся вернуть.
	var name, gameID, prevStatus, prevRuntime string
	var limitsRaw []byte
	var primaryPort int
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT name, game_id, COALESCE(limits, '{}'::jsonb), COALESCE(primary_port, 0),
		       COALESCE(status, ''), COALESCE(runtime_status, '')
		FROM core.servers WHERE id = $1 AND tenant_id = $2
	`, serverID, tenantID).Scan(&name, &gameID, &limitsRaw, &primaryPort, &prevStatus, &prevRuntime); err != nil {
		return err
	}
	limits := map[string]any{}
	_ = json.Unmarshal(limitsRaw, &limits)

	payload := map[string]any{
		"ports":   ports,
		"name":    name,
		"game_id": gameID,
		"limits":  limits,
	}
	if primaryPort > 0 {
		payload["primary_port"] = primaryPort
	}
	if bindIP := h.serverBindIP(ctx, tenantID, serverID); bindIP != "" {
		payload["bind_ip"] = bindIP
	}
	if img := h.resolveDockerImage(ctx, tenantID, serverID); img != "" {
		payload["docker_image"] = img
	}

	res, err := h.agentCommand(ctx, nodeID, serverID, protocol.ActionPortsSync, payload)
	if err != nil {
		return err
	}
	// Новый агент отчитывается числом опубликованных портов. Пустой ответ
	// означает, что команду никто не разобрал, и порт на ноде не открыт:
	// сообщаем об этом прямо, а не выдаём чужое «ок» за успех.
	if _, ok := res["ports"]; !ok {
		h.restoreServerStatus(ctx, tenantID, serverID, prevStatus, prevRuntime)
		return errAgentTooOld
	}
	return nil
}

// errAgentTooOld отделяет устаревшего агента от обычного отказа: чинится он не
// повтором действия, а обновлением агента на странице ноды.
var errAgentTooOld = errors.New("агент на ноде устарел и не умеет открывать порты — обновите его на странице ноды")

func (h *Handler) restoreServerStatus(ctx context.Context, tenantID, serverID, status, runtime string) {
	if status == "" && runtime == "" {
		return
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.servers
		SET status = NULLIF($3, ''), runtime_status = NULLIF($4, '')
		WHERE id = $1 AND tenant_id = $2
	`, serverID, tenantID, status, runtime)
}

func (h *Handler) syncPortsForServerHTTP(w http.ResponseWriter, r *http.Request, serverID string) bool {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return false
	}
	nodeID, err := h.serverNodeID(r.Context(), claims.TenantID, serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return false
	}
	if err := h.pushPortsState(r.Context(), claims.TenantID, nodeID, serverID); err != nil {
		// Устаревший агент — не сбой связи: отвечаем 409, чтобы в панели не
		// предлагалось «повторить», когда повтор ничего не изменит.
		status := http.StatusBadGateway
		if errors.Is(err, errAgentTooOld) {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return false
	}
	return true
}

func (h *Handler) pushCronState(ctx context.Context, nodeID, serverID string) error {
	jobs, err := h.loadCronJobs(ctx, serverID)
	if err != nil {
		return err
	}
	_, err = h.agentCommand(ctx, nodeID, serverID, protocol.ActionCronSync, map[string]any{"jobs": jobs})
	return err
}

func (h *Handler) pushFirewallState(ctx context.Context, nodeID, serverID string) error {
	rules, err := h.loadFirewallRules(ctx, serverID)
	if err != nil {
		return err
	}
	_, err = h.agentCommand(ctx, nodeID, serverID, protocol.ActionFirewallSync, map[string]any{"rules": rules})
	return err
}

func (h *Handler) syncCronForServerHTTP(w http.ResponseWriter, r *http.Request, serverID string) bool {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return false
	}
	nodeID, err := h.serverNodeID(r.Context(), claims.TenantID, serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return false
	}
	if err := h.pushCronState(r.Context(), nodeID, serverID); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return false
	}
	return true
}

func (h *Handler) syncFirewallForServerHTTP(w http.ResponseWriter, r *http.Request, serverID string) bool {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return false
	}
	nodeID, err := h.serverNodeID(r.Context(), claims.TenantID, serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return false
	}
	if err := h.pushFirewallState(r.Context(), nodeID, serverID); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return false
	}
	return true
}
