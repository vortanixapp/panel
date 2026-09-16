package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

type nodeMaintenanceBody struct {
	Enabled      bool   `json:"enabled"`
	Reason       string `json:"reason"`
	Until        string `json:"until"`
	NotifyOwners bool   `json:"notify_owners"`
}

func (h *Handler) SetNodeMaintenance(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	nodeID := chi.URLParam(r, "id")
	var body nodeMaintenanceBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx := r.Context()

	var until any
	if s := strings.TrimSpace(body.Until); s != "" {
		parsed, ok := parseFlexibleTime(s)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid until")
			return
		}
		until = parsed
	}

	var nodeName string
	var affected int
	var startedAt *time.Time
	err := h.dbOf(ctx).QueryRow(ctx, `
		UPDATE core.nodes SET
			maintenance_mode       = $2,
			maintenance_reason     = CASE WHEN $2 THEN $3 ELSE '' END,
			maintenance_until      = CASE WHEN $2 THEN $4::timestamptz ELSE NULL END,
			maintenance_started_at = CASE
				WHEN $2 AND maintenance_started_at IS NULL THEN now()
				WHEN $2 THEN maintenance_started_at
				ELSE NULL
			END
		WHERE id = $1
		RETURNING name, (SELECT COUNT(*)::int FROM core.servers s WHERE s.node_id = core.nodes.id),
		          maintenance_started_at
	`, nodeID, body.Enabled, strings.TrimSpace(body.Reason), until).Scan(&nodeName, &affected, &startedAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "location not found")
		return
	}

	notified := 0
	if body.Enabled && body.NotifyOwners {
		episode := time.Now()
		if startedAt != nil {
			episode = *startedAt
		}
		notified = h.notifyNodeOwners(ctx, nodeID, nodeName, body.Reason, until, episode)
	}

	audit(ctx, h.dbOf(ctx), claims.UserID, "node.maintenance", "node:"+nodeID,
		map[string]any{"enabled": body.Enabled, "reason": body.Reason, "notified": notified})
	h.emitWebhook(ctx, "node.maintenance", map[string]any{
		"node_id": nodeID, "node_name": nodeName,
		"enabled": body.Enabled, "reason": body.Reason,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"maintenance_mode": body.Enabled,
		"servers":          affected,
		"notified":         notified,
	})
}

func (h *Handler) notifyNodeOwners(ctx context.Context, nodeID, nodeName, reason string, until any, episode time.Time) int {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT DISTINCT user_id::text FROM core.servers
		WHERE node_id = $1 AND user_id IS NOT NULL
	`, nodeID)
	if err != nil {
		return 0
	}
	defer rows.Close()

	owners := []string{}
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil && id != "" {
			owners = append(owners, id)
		}
	}

	var extra []i18n.Msg
	if reason = strings.TrimSpace(reason); reason != "" {
		extra = append(extra, i18n.Key("notify.node_maintenance.reason", i18n.Params{"reason": reason}))
	}
	if t, ok := until.(time.Time); ok {
		extra = append(extra, i18n.Key("notify.node_maintenance.until", i18n.Params{"until": t.Format("02.01.2006 15:04")}))
	}
	for _, userID := range owners {
		h.notifyUser(ctx, userID, notify.Event{
			Kind:      notify.KindNodeMaintenance,
			Title:     i18n.Key("notify.node_maintenance.title"),
			Body:      i18n.Key("notify.node_maintenance.body", i18n.Params{"node": nodeName}),
			Extra:     extra,
			Meta:      map[string]any{"node_id": nodeID, "node_name": nodeName},
			DedupeKey: "node.maintenance:" + nodeID + ":" + strconv.FormatInt(episode.Unix(), 10),
		})
	}
	return len(owners)
}

func parseFlexibleTime(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func (h *Handler) nodeMaintenanceInfo(ctx context.Context, nodeID string) map[string]any {
	var enabled bool
	var reason string
	var until, startedAt *time.Time
	err := h.readerOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(maintenance_mode, false), COALESCE(maintenance_reason, ''),
		       maintenance_until, maintenance_started_at
		FROM core.nodes WHERE id = $1
	`, nodeID).Scan(&enabled, &reason, &until, &startedAt)
	if err != nil || !enabled {
		return nil
	}
	return map[string]any{
		"enabled":    true,
		"reason":     reason,
		"until":      timeOrNil(until),
		"started_at": timeOrNil(startedAt),
	}
}

func (h *Handler) nodeInMaintenance(ctx context.Context, nodeID string) bool {
	var enabled bool
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(maintenance_mode, false) FROM core.nodes WHERE id = $1
	`, nodeID).Scan(&enabled)
	return err == nil && enabled
}
