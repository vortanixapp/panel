package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/vortanixapp/panel/internal/api/paneljwt"
)

func (h *Handler) consoleCommandAllowed(ctx context.Context, claims *paneljwt.Claims, serverID string) bool {
	access, err := h.resolveServerAccess(ctx, claims.UserID, claims.Role, serverID)
	if err != nil {
		return false
	}
	if access.IsOwner || access.StaffWrite {
		return true
	}
	if !access.IsFriend {
		return false
	}
	allowed, _ := access.FriendPerms["can_console_command"].(bool)
	return allowed
}

func (h *Handler) CreateConsoleTicket(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	ctx := r.Context()

	if !h.authorizeServerAction(w, r, claims, serverID, "console_attach") {
		return
	}
	canCommand := h.consoleCommandAllowed(ctx, claims, serverID)

	var nodeID, gameID string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(node_id::text, ''), COALESCE(game_id, '') FROM core.servers WHERE id = $1
	`, serverID).Scan(&nodeID, &gameID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	if nodeID == "" {
		writeError(w, http.StatusConflict, "сервер ещё не размещён на узле")
		return
	}

	ticket := uuid.NewString()
	sessionID := uuid.NewString()
	if err := h.cache.SetConsoleTicket(ctx, ticket, map[string]any{
		"server_id":   serverID,
		"node_id":     nodeID,
		"game_id":     gameID,
		"user_id":     claims.UserID,
		"session_id":  sessionID,
		"can_command": canCommand,
	}, 30*time.Second); err != nil {
		writeError(w, http.StatusServiceUnavailable, "console unavailable")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ticket":     ticket,
		"session_id": sessionID,
		"expires_in": 30,
	})
}

func (h *Handler) GetServerMetrics(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, serverID, "metrics"); !ok {
		return
	}
	ctx := r.Context()

	hours := 0
	if v := r.URL.Query().Get("hours"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			hours = n
		}
	}

	var points []map[string]any
	var err error

	if hours > 0 {
		points, err = getMetricsFromPG(ctx, h.dbOf(ctx), serverID, hours, 500)
	} else {
		points, err = h.cache.GetMetrics(ctx, serverID, 60)
		if err == nil && len(points) < 5 {
			pgPoints, pgErr := getMetricsFromPG(ctx, h.dbOf(ctx), serverID, 24, 500)
			if pgErr == nil && len(pgPoints) > 0 {
				points = mergeMetricPoints(points, pgPoints)
			}
		}
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "metrics unavailable")
		return
	}
	if points == nil {
		points = []map[string]any{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"points": points})
}
