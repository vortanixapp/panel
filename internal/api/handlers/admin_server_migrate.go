package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/internal/api/jobwake"
)

type migrateServerBody struct {
	ToNodeID     string `json:"to_node_id"`
	RemoveSource *bool  `json:"remove_source"`
}

func (h *Handler) AdminServerMigrationTargets(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	ctx := r.Context()

	var currentNode, gameID string
	var limits []byte
	if h.readerOf(ctx).QueryRow(ctx, `
		SELECT node_id::text, game_id, limits FROM core.servers WHERE id = $1
	`, serverID).Scan(&currentNode, &gameID, &limits) != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	var lim map[string]any
	_ = json.Unmarshal(limits, &lim)
	needRAM := intFromLimits(lim, "memory_mb", "ram_mb", "memory")
	needDisk := intFromLimits(lim, "disk_mb", "disk")

	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT n.id::text, n.name, COALESCE(n.fqdn, ''), n.status,
		       COALESCE(n.maintenance_mode, false),
		       (SELECT COUNT(*)::int FROM core.servers s WHERE s.node_id = n.id),
		       m.cpu_usage, m.ram_usage, m.ram_total, m.disk_total, m.disk_used, m.disk_available
		FROM core.nodes n
		LEFT JOIN LATERAL (
			SELECT
				MAX(value) FILTER (WHERE metric_type = 'cpu_usage')           AS cpu_usage,
				MAX(value) FILTER (WHERE metric_type = 'ram_usage')           AS ram_usage,
				MAX(text_value) FILTER (WHERE metric_type = 'ram_total')      AS ram_total,
				MAX(text_value) FILTER (WHERE metric_type = 'disk_total')     AS disk_total,
				MAX(text_value) FILTER (WHERE metric_type = 'disk_used')      AS disk_used,
				MAX(text_value) FILTER (WHERE metric_type = 'disk_available') AS disk_available
			FROM (
				SELECT DISTINCT ON (metric_type) metric_type, value, text_value, measured_at
				FROM core.node_metrics
				WHERE node_id = n.id
				  AND metric_type IN ('cpu_usage', 'ram_usage', 'ram_total',
				                      'disk_total', 'disk_used', 'disk_available')
				ORDER BY metric_type, measured_at DESC
			) latest
		) m ON TRUE
		WHERE n.id::text <> $1
		ORDER BY n.name
	`, currentNode)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	nodes := []map[string]any{}
	for rows.Next() {
		var id, name, fqdn, status string
		var maintenance bool
		var servers int
		var cpu, ramPercent *float64
		var ramTotal, diskTotal, diskUsed, diskAvail *string
		if rows.Scan(&id, &name, &fqdn, &status, &maintenance, &servers,
			&cpu, &ramPercent, &ramTotal, &diskTotal, &diskUsed, &diskAvail) != nil {
			continue
		}

		ramTotalMB := parseHumanSizeMB(strPtr(ramTotal))
		freeRAM := 0.0
		if ramTotalMB > 0 && ramPercent != nil {
			freeRAM = ramTotalMB * (1 - *ramPercent/100)
		}
		freeDisk := parseHumanSizeMB(strPtr(diskAvail))
		diskTotalMB := parseHumanSizeMB(strPtr(diskTotal))
		if freeDisk == 0 && diskTotalMB > 0 {
			freeDisk = diskTotalMB - parseHumanSizeMB(strPtr(diskUsed))
		}

		known := ramTotalMB > 0 || diskTotalMB > 0
		fits := true
		reason := ""
		switch {
		case maintenance:
			fits, reason = false, "нода на обслуживании"
		case status != "online":
			fits, reason = false, "нода не в сети"
		case known && needRAM > 0 && freeRAM > 0 && freeRAM < float64(needRAM):
			fits, reason = false, "не хватает памяти"
		case known && needDisk > 0 && freeDisk > 0 && freeDisk < float64(needDisk):
			fits, reason = false, "не хватает места на диске"
		}
		nodes = append(nodes, map[string]any{
			"id": id, "name": name, "fqdn": fqdn, "status": status,
			"maintenance_mode": maintenance,
			"servers":          servers, "metrics_known": known,
			"ram_total_mb": ramTotalMB, "ram_free_mb": freeRAM,
			"disk_total_mb": diskTotalMB, "disk_free_mb": freeDisk,
			"cpu_percent": floatOrZero(cpu), "ram_percent": floatOrZero(ramPercent),
			"fits": fits, "reason": reason,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"nodes":           nodes,
		"current_node_id": currentNode,
		"game_id":         gameID,
		"need_ram_mb":     needRAM,
		"need_disk_mb":    needDisk,
	})
}

func intFromLimits(lim map[string]any, keys ...string) int {
	for _, k := range keys {
		switch v := lim[k].(type) {
		case float64:
			if v > 0 {
				return int(v)
			}
		case int:
			if v > 0 {
				return v
			}
		case string:
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}

func parseHumanSizeMB(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	s = strings.TrimSuffix(s, "i")
	if s == "" {
		return 0
	}
	unit := s[len(s)-1]
	number := s
	multiplier := 1.0 / (1024 * 1024)
	switch unit {
	case 'K', 'k':
		number, multiplier = s[:len(s)-1], 1.0/1024
	case 'M', 'm':
		number, multiplier = s[:len(s)-1], 1
	case 'G', 'g':
		number, multiplier = s[:len(s)-1], 1024
	case 'T', 't':
		number, multiplier = s[:len(s)-1], 1024*1024
	case 'B', 'b':
		number = s[:len(s)-1]
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(number), 64)
	if err != nil {
		return 0
	}
	return value * multiplier
}

func floatOrZero(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func (h *Handler) AdminServerMigrate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	var body migrateServerBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	toNode := strings.TrimSpace(body.ToNodeID)
	if toNode == "" {
		writeError(w, http.StatusBadRequest, "to_node_id required")
		return
	}
	removeSource := true
	if body.RemoveSource != nil {
		removeSource = *body.RemoveSource
	}
	ctx := r.Context()

	var fromNode, serverName, provStatus, serverStatus string
	if h.dbOf(ctx).QueryRow(ctx, `
		SELECT node_id::text, name, COALESCE(provisioning_status, ''), COALESCE(status, '')
		FROM core.servers WHERE id = $1
	`, serverID).Scan(&fromNode, &serverName, &provStatus, &serverStatus) != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	if fromNode == toNode {
		writeError(w, http.StatusBadRequest, "server is already on this node")
		return
	}
	if provStatus == "provisioning" || provStatus == "pending" || provStatus == "migrating" {
		writeError(w, http.StatusConflict, "server is not ready: "+provStatus)
		return
	}
	if serverStatus == "installing" || serverStatus == "reinstalling" {
		writeError(w, http.StatusConflict, "server is installing")
		return
	}

	var fromName, toName, toStatus string
	if h.dbOf(ctx).QueryRow(ctx, `SELECT name, status FROM core.nodes WHERE id = $1`,
		toNode).Scan(&toName, &toStatus) != nil {
		writeError(w, http.StatusNotFound, "target node not found")
		return
	}
	if toStatus != "online" {
		writeError(w, http.StatusConflict, "target node is "+toStatus)
		return
	}
	if h.nodeInMaintenance(ctx, toNode) {
		writeError(w, http.StatusConflict, "нода назначения на обслуживании")
		return
	}
	_ = h.dbOf(ctx).QueryRow(ctx, `SELECT name FROM core.nodes WHERE id = $1`, fromNode).Scan(&fromName)

	var migrationID string
	err := h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.server_migrations
			( server_id, from_node_id, to_node_id, from_node_name, to_node_name,
			 status, stage, remove_source, created_by)
		VALUES ( $1, $2, $3, $4, $5, 'pending', 'queued', $6, $7)
		RETURNING id::text
	`, serverID, fromNode, toNode, fromName, toName,
		removeSource, nullableUUID(claims.UserID)).Scan(&migrationID)
	if err != nil {
		writeError(w, http.StatusConflict, "migration already in progress")
		return
	}

	payload, _ := json.Marshal(map[string]any{
		"server_id":     serverID,
		"to_node_id":    toNode,
		"migration_id":  migrationID,
		"remove_source": removeSource,
	})
	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.jobs ( type, status, payload)
		VALUES ( 'migrate_server', 'pending', $1::jsonb)
	`, payload); err != nil {
		_, _ = h.dbOf(ctx).Exec(ctx, `
			UPDATE core.server_migrations SET status = 'failed', error = 'не удалось поставить задачу', finished_at = now()
			WHERE id = $1
		`, migrationID)
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	jobwake.Notify("migrate_server")
	audit(ctx, h.dbOf(ctx), claims.UserID, "server.migrate", "server:"+serverID,
		map[string]any{"from": fromName, "to": toName, "remove_source": removeSource})
	h.auditAlert(ctx, claims.UserID, claims.Email, "server.migrate",
		serverName+": "+fromName+" → "+toName)

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":       "queued",
		"migration_id": migrationID,
		"to_node":      toName,
	})
}

func (h *Handler) AdminServerMigrations(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT id::text, status, stage, COALESCE(error, ''), bytes, remove_source,
		       from_node_name, to_node_name, created_at, started_at, finished_at
		FROM core.server_migrations
		WHERE server_id = $1
		ORDER BY created_at DESC
		LIMIT 20
	`, serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var id, status, stage, errText, fromName, toName string
		var bytes int64
		var removeSource bool
		var createdAt time.Time
		var startedAt, finishedAt *time.Time
		if rows.Scan(&id, &status, &stage, &errText, &bytes, &removeSource,
			&fromName, &toName, &createdAt, &startedAt, &finishedAt) != nil {
			continue
		}
		list = append(list, map[string]any{
			"id": id, "status": status, "stage": stage, "error": errText,
			"bytes": bytes, "remove_source": removeSource,
			"from_node": fromName, "to_node": toName,
			"created_at":  createdAt.Format(time.RFC3339),
			"started_at":  timeOrNil(startedAt),
			"finished_at": timeOrNil(finishedAt),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"migrations": list})
}

func timeOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}
