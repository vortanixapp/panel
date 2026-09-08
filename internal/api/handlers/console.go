package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/vortanix/vortanix/internal/api/paneljwt"
	"github.com/vortanix/vortanix/internal/api/relay"
	"github.com/vortanix/vortanix/pkg/protocol"
)

// consoleCommandAllowed говорит, вправе ли зритель вводить команды. Владелец и
// сотрудник — да; друг — только с явно выданным правом.
func (h *Handler) consoleCommandAllowed(ctx context.Context, claims *paneljwt.Claims, serverID string) bool {
	access, err := h.resolveServerAccess(ctx, claims.TenantID, claims.UserID, claims.Role, serverID)
	if err != nil {
		return false
	}
	if access.IsOwner || access.IsStaff {
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

	// Билет открывает живую консоль сервера — по ней видно всё происходящее и
	// вводятся команды. Проверялась только принадлежность арендатору, поэтому
	// консоль чужого сервера открывалась по одному его id.
	if !h.authorizeServerAction(w, r, claims, serverID, "console_attach") {
		return
	}
	// Право на ввод команд отдельное от права смотреть: друг с доступом только
	// на чтение не должен управлять сервером через консоль. Шлюз сам решить
	// это не может — он видит лишь билет, поэтому кладём признак в него.
	canCommand := h.consoleCommandAllowed(ctx, claims, serverID)

	var nodeID, status string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT node_id::text, status FROM core.servers WHERE id = $1 AND tenant_id = $2
	`, serverID, claims.TenantID).Scan(&nodeID, &status)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	ticket := uuid.NewString()
	sessionID := uuid.NewString()
	ticketData := protocol.ConsoleTicket{
		ServerID: serverID,
		NodeID:   nodeID,
		TenantID: claims.TenantID,
		UserID:   claims.UserID,
	}
	_ = h.cache.SetConsoleTicket(ctx, ticket, map[string]any{
		"server_id":   serverID,
		"node_id":     nodeID,
		"tenant_id":   claims.TenantID,
		"user_id":     claims.UserID,
		"session_id":  sessionID,
		"can_command": canCommand,
	}, 30*time.Second)

	_ = h.cache.SetJSON(ctx, "console:session:"+sessionID, ticketData, 2*time.Hour)

	err = h.relay.StartConsole(ctx, relay.ConsoleStartRequest{
		SessionID: sessionID,
		ServerID:  serverID,
		NodeID:    nodeID,
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, "failed to start console: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ticket":     ticket,
		"session_id": sessionID,
		"expires_in": 30,
	})
}

func (h *Handler) GetServerMetrics(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	ctx := r.Context()

	var exists bool
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM core.servers WHERE id = $1 AND tenant_id = $2)
	`, serverID, claims.TenantID).Scan(&exists)
	if !exists {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

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
