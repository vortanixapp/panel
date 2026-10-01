package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/pkg/rbac"
)

const dashboardRecheckEvery = time.Minute

type viewerState int

const (
	viewerAllowed viewerState = iota
	viewerDenied
	viewerUnknown
)

func (h *Handler) dashboardViewer(ctx context.Context, userID, sessionID string) (map[string]bool, viewerState) {
	if h.db == nil || strings.TrimSpace(userID) == "" {
		return nil, viewerUnknown
	}
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var role, status string
	var sessionAlive bool
	err := h.db.QueryRow(checkCtx, `
		SELECT u.role, u.status,
		       ($2 = '' OR EXISTS(
		           SELECT 1 FROM core.user_sessions s
		           WHERE s.id::text = $2 AND s.user_id = u.id
		       ))
		FROM core.users u WHERE u.id::text = $1
	`, userID, sessionID).Scan(&role, &status, &sessionAlive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, viewerDenied
		}
		return nil, viewerUnknown
	}
	if status != "active" || !sessionAlive {
		return nil, viewerDenied
	}
	switch role {
	case rbac.RoleOwner, rbac.RoleAdmin, rbac.RoleSupport:
	default:
		return nil, viewerDenied
	}
	return rbac.ForUser(checkCtx, h.db, userID, role), viewerAllowed
}

func dashboardEventPermissions(eventType string) []string {
	switch {
	case strings.HasPrefix(eventType, "node."):
		return []string{"admin.daemons.read", "admin.locations.read"}
	case strings.HasPrefix(eventType, "server."):
		return []string{"admin.servers.read"}
	}
	return nil
}

func dashboardEventAllowed(permissions map[string]bool, payload []byte) bool {
	var head struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(payload, &head) != nil {
		return false
	}
	for _, permission := range dashboardEventPermissions(head.Type) {
		if permissions[permission] {
			return true
		}
	}
	return false
}
