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
			(SELECT COUNT(*)::int FROM core.servers s WHERE s.node_id = n.id)
		FROM core.nodes n
		LEFT JOIN core.node_daemons d ON d.node_id = n.id
		ORDER BY COALESCE(n.sort_order, 0), n.name
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	list := []map[string]any{}
	var ids []string
	for rows.Next() {
		var id, name, country, code, fqdn, ipAddress, daemonStatus string
		var meta []byte
		var daemonLastSeen *time.Time
		var serversCount int
		if rows.Scan(&id, &name, &country, &code, &fqdn, &ipAddress, &meta,
			&daemonStatus, &daemonLastSeen, &serversCount) != nil {
			continue
		}
		metaMap := parseMetaMap(meta)
		containerRunning, containerKnown := agentContainerRunning(metaMap)
		isOnline := agentDaemonOnline(daemonStatus, daemonLastSeen)
		item := map[string]any{
			"id": id, "name": name, "country": country, "code": code,
			"fqdn": fqdn, "ip_address": ipAddress,
			"status":        resolveAgentStatus(isOnline, containerKnown, containerRunning, daemonStatus),
			"state":         agentPresenceState(daemonStatus, daemonLastSeen),
			"daemon_status": daemonStatus,
			"is_online":     isOnline,
			"servers_count": serversCount,
		}
		if daemonLastSeen != nil {
			item["daemon_last_seen_at"] = daemonLastSeen.Format(time.RFC3339)
		}
		list = append(list, item)
		ids = append(ids, id)
	}
	rows.Close()

	resources := h.loadNodeResources(r.Context(), ids)
	for _, item := range list {
		res := resources[item["id"].(string)]
		if res == nil {
			continue
		}
		if res.CPUPercent != nil {
			item["cpu_percent"] = *res.CPUPercent
		}
		if res.RAMPercent != nil {
			item["ram_percent"] = *res.RAMPercent
		}
		if res.DiskPercent != nil {
			item["disk_percent"] = *res.DiskPercent
		}
		if v := mbToBytesString(res.RAMTotalMB); v != "" {
			item["ram_total"] = v
			item["ram_total_mb"] = res.RAMTotalMB
		}
		if v := mbToBytesString(res.DiskTotalMB); v != "" {
			item["disk_total"] = v
			item["disk_total_mb"] = res.DiskTotalMB
		}
		if v := mbToBytesString(res.DiskUsedMB); v != "" {
			item["disk_used"] = v
			item["disk_used_mb"] = res.DiskUsedMB
		}
		if res.MeasuredAt != nil {
			item["measured_at"] = res.MeasuredAt.Format(time.RFC3339)
		}
		item["metrics_source"] = res.Source
	}

	writeJSON(w, http.StatusOK, map[string]any{"nodes": list})
}
