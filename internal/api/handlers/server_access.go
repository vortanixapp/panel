package handlers

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

var viewerPermissionKeys = []string{
	"can_view_main",
	"can_view_console",
	"can_view_logs",
	"can_view_metrics",
	"can_view_ftp",
	"can_view_mysql",
	"can_view_cron",
	"can_view_firewall",
	"can_view_ports",
	"can_view_settings",
	"can_view_friends",
	"can_start",
	"can_stop",
	"can_restart",
	"can_reinstall",
	"can_console_command",
	"can_files",
	"can_cron_manage",
	"can_firewall_manage",
	"can_ports_manage",
	"can_settings_edit",
}

type serverAccess struct {
	OwnerID     string
	IsOwner     bool
	IsFriend    bool
	IsStaff     bool
	FriendPerms map[string]any
	// Блокировка администратором. Раньше признак только показывался в панели:
	// заблокированный сервер продолжал работать, и владелец управлял им как
	// обычно — останавливал, правил настройки, менял файлы.
	IsBlocked     bool
	BlockedReason string
}

func ownerViewerPermissions() map[string]any {
	out := map[string]any{}
	for _, key := range viewerPermissionKeys {
		out[key] = true
	}
	return out
}

func friendViewerPermissions(raw map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range viewerPermissionKeys {
		out[key] = false
	}
	if raw == nil {
		return out
	}
	for _, key := range viewerPermissionKeys {
		if v, ok := raw[key]; ok {
			out[key] = truthy(v)
		}
	}
	out["can_view_main"] = true
	return out
}

func (h *Handler) resolveServerAccess(ctx context.Context, tenantID, userID, role, serverID string) (*serverAccess, error) {
	access := &serverAccess{IsStaff: isStaffRole(role)}
	var ownerID string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(user_id::text, ''), COALESCE(is_blocked, false), COALESCE(blocked_reason, '')
		FROM core.servers WHERE id = $1 AND tenant_id = $2
	`, serverID, tenantID).Scan(&ownerID, &access.IsBlocked, &access.BlockedReason)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	access.OwnerID = ownerID
	if access.IsStaff || ownerID == userID {
		access.IsOwner = ownerID == userID
		return access, nil
	}

	var permsRaw []byte
	friendErr := h.dbOf(ctx).QueryRow(ctx, `
		SELECT permissions FROM core.server_friends
		WHERE server_id = $1 AND user_id = $2::uuid
	`, serverID, userID).Scan(&permsRaw)
	if friendErr != nil {
		if errors.Is(friendErr, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, friendErr
	}
	access.IsFriend = true
	perms := map[string]any{}
	_ = json.Unmarshal(permsRaw, &perms)
	access.FriendPerms = perms
	return access, nil
}

func viewerPermissionsForAccess(access *serverAccess) map[string]any {
	if access == nil {
		return friendViewerPermissions(nil)
	}
	if access.IsOwner || access.IsStaff {
		return ownerViewerPermissions()
	}
	if access.IsFriend {
		return friendViewerPermissions(access.FriendPerms)
	}
	return friendViewerPermissions(nil)
}

func viewerHasPermission(perms map[string]any, key string) bool {
	if perms == nil {
		return false
	}
	if v, ok := perms[key]; ok {
		return truthy(v)
	}
	return false
}
