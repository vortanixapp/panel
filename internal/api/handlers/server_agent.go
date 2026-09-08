package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/vortanix/vortanix/internal/api/paneljwt"
	"github.com/vortanix/vortanix/internal/api/relay"
)

func (h *Handler) serverNodeID(ctx context.Context, tenantID, serverID string) (string, error) {
	var nodeID string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT node_id::text FROM core.servers WHERE id = $1 AND tenant_id = $2
	`, serverID, tenantID).Scan(&nodeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
		return "", err
	}
	return nodeID, nil
}

func (h *Handler) agentCommand(
	ctx context.Context,
	nodeID, serverID, action string,
	payload map[string]any,
) (map[string]any, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	resp, err := h.relay.CommandSync(ctx, nodeID, relay.CommandRequest{
		CommandID: uuid.NewString(),
		Action:    action,
		ServerID:  serverID,
		Payload:   payload,
	})
	if err != nil {
		return nil, err
	}
	if resp.Result == nil {
		return map[string]any{}, nil
	}
	return resp.Result, nil
}

func (h *Handler) ensureServerForTenant(w http.ResponseWriter, r *http.Request, serverID string) (*paneljwt.Claims, bool) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return nil, false
	}
	var exists bool
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT EXISTS(SELECT 1 FROM core.servers WHERE id = $1 AND tenant_id = $2)
	`, serverID, claims.TenantID).Scan(&exists)
	if !exists {
		writeError(w, http.StatusNotFound, "server not found")
		return nil, false
	}
	return claims, true
}

func (h *Handler) verifyServerTenant(ctx context.Context, tenantID, serverID string) error {
	var ok bool
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM core.servers WHERE id = $1 AND tenant_id = $2)
	`, serverID, tenantID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return pgx.ErrNoRows
	}
	return nil
}

func (h *Handler) requireServerTenant(w http.ResponseWriter, r *http.Request, tenantID, serverID string) bool {
	err := h.verifyServerTenant(r.Context(), tenantID, serverID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "server not found")
		} else {
			writeError(w, http.StatusInternalServerError, "database error")
		}
		return false
	}
	return true
}

func (h *Handler) agentCommandForServer(
	w http.ResponseWriter,
	r *http.Request,
	serverID, action string,
	payload map[string]any,
) (map[string]any, bool) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return nil, false
	}
	ctx := r.Context()
	if !h.authorizeServerAction(w, r, claims, serverID, action) {
		return nil, false
	}
	nodeID, err := h.serverNodeID(ctx, claims.TenantID, serverID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "server not found")
			return nil, false
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return nil, false
	}
	result, err := h.agentCommand(ctx, nodeID, serverID, action, payload)
	if err != nil {
		code, msg := explainAgentError(action, err)
		writeCodedError(w, http.StatusBadGateway, code, msg)
		return nil, false
	}
	return result, true
}

func explainAgentError(action string, err error) (code, message string) {
	text := err.Error()
	low := strings.ToLower(text)

	switch {
	case strings.Contains(low, "access denied") || strings.Contains(low, "error 1045"):
		return "mysql_root_denied",
			"Сервер MySQL на ноде не принял пароль root. Так бывает, когда контейнер " +
				"создан вне установки панели: MySQL применяет заданный пароль только при " +
				"первичной инициализации пустого тома. Укажите действующий пароль в " +
				"настройках ноды — поле root_password у нужного инстанса MySQL."

	case strings.Contains(low, "no such container"):
		return "mysql_container_missing",
			"На ноде нет контейнера MySQL. Запустите установку компонента «MySQL» " +
				"в настройках ноды."

	case strings.Contains(low, "mysql instance not configured") ||
		strings.Contains(low, "unknown mysql_instance_key"):
		return "mysql_not_configured",
			"Для ноды не настроен ни один инстанс MySQL. Добавьте его в настройках ноды " +
				"или запустите установку компонента «MySQL»."

	case strings.Contains(low, "node offline") || strings.Contains(low, "agent not connected"):
		return "node_offline",
			"Нода не на связи: агент не подключён к relay."

	case strings.Contains(low, "timeout") || strings.Contains(low, "deadline exceeded"):
		return "agent_timeout",
			"Агент не ответил вовремя. Проверьте состояние ноды и повторите."
	}

	return "agent_error", text
}

// blockedForbids — запрещено ли действие на заблокированном сервере.
//
// Запрещаем всё, что запускает сервер или меняет его состояние; чтение
// оставляем доступным. Скачать свои файлы и посмотреть логи владелец вправе и
// под блокировкой — она про то, чтобы сервер не работал, а не про то, чтобы
// отобрать данные.
func blockedForbids(action string) bool {
	switch action {
	case "files_list", "files_read", "files_download", "logs",
		"settings_read", "cron_list", "firewall_list", "ports_list", "friends_list",
		"backup_schedule_read", "power_stop", "power_kill":
		return false
	default:
		return true
	}
}

func agentActionPermission(action string) string {
	switch action {
	case "files_list", "files_read", "files_download":
		return "can_view_ftp"
	case "files_write", "files_mkdir", "files_delete", "files_upload":
		return "can_files"
	case "console_command":
		return "can_console_command"
	case "cron_list":
		return "can_view_cron"
	case "cron_create", "cron_delete", "cron_toggle":
		return "can_cron_manage"
	case "firewall_list":
		return "can_view_firewall"
	case "firewall_create", "firewall_delete", "firewall_toggle":
		return "can_firewall_manage"
	case "ports_list":
		return "can_view_ports"
	case "ports_create", "ports_delete":
		return "can_ports_manage"
	case "friends_list":
		return "can_view_friends"
	case "backup_schedule_read":
		return "can_view_settings"
	case "backup_schedule_write":
		return "can_settings_edit"
	case "logs":
		return "can_view_logs"
	case "update", "reinstall":
		return "can_reinstall"
	case "console_attach":
		// Смотреть консоль и вводить в неё команды — разные права: второе
		// проверяется отдельно, уже при вводе, по признаку в билете.
		return "can_view_console"
	case "plugins_apply", "maps_apply":
		// Плагины и карты меняют настройки сервера, а не читают их. Без этой
		// строки право сводилось к пустому: друзья владельца с can_settings_edit
		// не могли ничего, а проверка сводилась к «владелец или сотрудник».
		return "can_settings_edit"
	case "power_start":
		return "can_start"
	case "power_stop", "power_kill":
		return "can_stop"
	case "power_restart":
		return "can_restart"
	case "settings_read":
		return "can_view_settings"
	case "settings_write", "settings_raw_write":
		// Права на просмотр и на правку настроек в системе разные, но ручки
		// настроек не проверяли ни одно: они звали agentCommand напрямую, минуя
		// authorizeServerAction. Достаточно было принадлежать тому же тенанту.
		return "can_settings_edit"
	default:
		return ""
	}
}

// Вкладки планировщика, firewall, портов и друзей ходили в базу через
// ensureServerForTenant: проверялась только принадлежность сервера арендатору,
// поэтому любой вошедший клиент правил чужой сервер, зная его id. Здесь та же
// проверка владельца и прав друга, что и у остальных вкладок.
func (h *Handler) authorizeServerTab(
	w http.ResponseWriter,
	r *http.Request,
	serverID, action string,
) (*paneljwt.Claims, bool) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return nil, false
	}
	if !h.authorizeServerAction(w, r, claims, serverID, action) {
		return nil, false
	}
	return claims, true
}

func (h *Handler) authorizeServerAction(
	w http.ResponseWriter,
	r *http.Request,
	claims *paneljwt.Claims,
	serverID, action string,
) bool {
	access, err := h.resolveServerAccess(r.Context(), claims.TenantID, claims.UserID, claims.Role, serverID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "server not found")
			return false
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return false
	}
	// Блокировка бьёт по всем, кроме сотрудников: им нужно и разобраться, и
	// снять её. Чтение оставляем — клиент должен видеть свой сервер и причину,
	// по которой он к нему не допущен, а не упираться в отказ на каждом экране.
	if access.IsBlocked && !access.IsStaff && blockedForbids(action) {
		reason := access.BlockedReason
		if reason == "" {
			reason = "обратитесь в поддержку"
		}
		writeError(w, http.StatusForbidden, "сервер заблокирован: "+reason)
		return false
	}
	if access.IsOwner || access.IsStaff {
		return true
	}
	perm := agentActionPermission(action)
	if perm == "" || !access.IsFriend {
		writeError(w, http.StatusForbidden, "нет доступа к этому серверу")
		return false
	}
	if allowed, _ := access.FriendPerms[perm].(bool); !allowed {
		writeError(w, http.StatusForbidden, "нет права "+perm)
		return false
	}
	return true
}
