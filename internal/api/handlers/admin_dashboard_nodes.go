package handlers

import (
	"net/http"
	"time"
)

func (h *Handler) AdminDashboardNodes(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT n.id::text, n.name, COALESCE(n.country, ''),
			COALESCE(n.meta->>'code', n.fqdn, ''), COALESCE(n.fqdn, ''),
			COALESCE(n.ip_address, ''), n.meta,
			COALESCE(d.status, 'unknown'), d.last_seen_at,
			(SELECT COUNT(*)::int FROM core.servers s WHERE s.node_id = n.id),
			m.cpu_usage, m.ram_usage, m.ram_total, m.disk_total, m.disk_used, m.measured_at
		FROM core.nodes n
		LEFT JOIN core.node_daemons d ON d.node_id = n.id
		LEFT JOIN LATERAL (
			SELECT
				MAX(value) FILTER (WHERE metric_type = 'cpu_usage')       AS cpu_usage,
				MAX(value) FILTER (WHERE metric_type = 'ram_usage')       AS ram_usage,
				MAX(text_value) FILTER (WHERE metric_type = 'ram_total')  AS ram_total,
				MAX(text_value) FILTER (WHERE metric_type = 'disk_total') AS disk_total,
				MAX(text_value) FILTER (WHERE metric_type = 'disk_used')  AS disk_used,
				MAX(measured_at)                                          AS measured_at
			FROM (
				SELECT DISTINCT ON (metric_type) metric_type, value, text_value, measured_at
				FROM core.node_metrics
				WHERE node_id = n.id
				  AND metric_type IN ('cpu_usage', 'ram_usage', 'ram_total', 'disk_total', 'disk_used')
				ORDER BY metric_type, measured_at DESC
			) latest
		) m ON TRUE
		ORDER BY COALESCE(n.sort_order, 0), n.name
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var id, name, country, code, fqdn, ipAddress, daemonStatus string
		var meta []byte
		var daemonLastSeen, measuredAt *time.Time
		var serversCount int
		var cpuUsage, ramUsage *float64
		var ramTotal, diskTotal, diskUsed *string
		if rows.Scan(&id, &name, &country, &code, &fqdn, &ipAddress, &meta,
			&daemonStatus, &daemonLastSeen, &serversCount,
			&cpuUsage, &ramUsage, &ramTotal, &diskTotal, &diskUsed, &measuredAt) != nil {
			continue
		}
		metaMap := parseMetaMap(meta)
		containerRunning, containerKnown := agentContainerRunning(metaMap)
		isOnline := agentDaemonOnline(daemonStatus, daemonLastSeen)
		item := map[string]any{
			"id": id, "name": name, "country": country, "code": code,
			"fqdn": fqdn, "ip_address": ipAddress,
			"status":        resolveAgentStatus(isOnline, containerKnown, containerRunning, daemonStatus),
			"daemon_status": daemonStatus,
			"is_online":     isOnline,
			"servers_count": serversCount,
		}
		if cpuUsage != nil {
			item["cpu_percent"] = *cpuUsage
		}
		if ramUsage != nil {
			item["ram_percent"] = *ramUsage
		}
		if ramTotal != nil && *ramTotal != "" {
			item["ram_total"] = *ramTotal
		}
		if diskTotal != nil && *diskTotal != "" {
			item["disk_total"] = *diskTotal
		}
		if diskUsed != nil && *diskUsed != "" {
			item["disk_used"] = *diskUsed
		}
		if measuredAt != nil {
			item["measured_at"] = measuredAt.Format(time.RFC3339)
		}
		if daemonLastSeen != nil {
			item["daemon_last_seen_at"] = daemonLastSeen.Format(time.RFC3339)
		}
		list = append(list, item)
	}

	writeJSON(w, http.StatusOK, map[string]any{"nodes": list})
}
