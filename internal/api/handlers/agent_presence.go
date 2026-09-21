package handlers

import (
	"context"
	"encoding/json"
	"strconv"
	"time"
)

const (
	agentStateOnline         = "online"
	agentStateOffline        = "offline"
	agentStateNeverConnected = "never_connected"

	nodeSnapshotFresh = 10 * time.Minute
)

func agentPresenceState(daemonStatus string, lastSeen *time.Time) string {
	if agentDaemonOnline(daemonStatus, lastSeen) {
		return agentStateOnline
	}
	if lastSeen == nil {
		return agentStateNeverConnected
	}
	return agentStateOffline
}

func agentUptimeSeconds(online bool, startedAt, connectedAt *time.Time) (float64, bool) {
	if !online {
		return 0, false
	}
	since := startedAt
	if since == nil {
		since = connectedAt
	}
	if since == nil {
		return 0, false
	}
	secs := time.Since(*since).Seconds()
	if secs < 0 {
		secs = 0
	}
	return secs, true
}

type nodeResources struct {
	Source      string
	MeasuredAt  *time.Time
	CPUPercent  *float64
	RAMPercent  *float64
	DiskPercent *float64
	RAMTotalMB  float64
	RAMUsedMB   float64
	DiskTotalMB float64
	DiskUsedMB  float64
	DiskFreeMB  float64
	DiskQuota   *bool
}

func (r *nodeResources) known() bool {
	return r != nil && (r.RAMTotalMB > 0 || r.DiskTotalMB > 0)
}

func (h *Handler) loadNodeResources(ctx context.Context, nodeIDs []string) map[string]*nodeResources {
	out := make(map[string]*nodeResources, len(nodeIDs))
	if len(nodeIDs) == 0 {
		return out
	}
	db := h.readerOf(ctx)
	rows, err := db.Query(ctx, `
		SELECT node_id::text, stats, stats_at
		FROM core.node_daemons
		WHERE node_id::text = ANY($1::text[])
		  AND stats_at IS NOT NULL
		  AND stats_at > now() - make_interval(secs => $2)
	`, nodeIDs, nodeSnapshotFresh.Seconds())
	if err == nil {
		for rows.Next() {
			var id string
			var raw []byte
			var at time.Time
			if rows.Scan(&id, &raw, &at) != nil {
				continue
			}
			if res := resourcesFromStats(raw, at); res != nil {
				out[id] = res
			}
		}
		rows.Close()
	}

	var missing []string
	for _, id := range nodeIDs {
		if _, ok := out[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return out
	}
	rows, err = db.Query(ctx, `
		SELECT n.id, t.metric_type, m.value, m.text_value, m.measured_at
		FROM unnest($1::text[]) AS n(id)
		CROSS JOIN unnest(ARRAY[
			'cpu_usage', 'ram_usage', 'disk_usage', 'ram_total',
			'disk_total', 'disk_used', 'disk_available', 'disk_quota'
		]) AS t(metric_type)
		CROSS JOIN LATERAL (
			SELECT value, text_value, measured_at
			FROM core.node_metrics
			WHERE node_id = n.id::uuid AND metric_type = t.metric_type
			ORDER BY measured_at DESC
			LIMIT 1
		) m
	`, missing)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, metricType string
		var value float64
		var text *string
		var at time.Time
		if rows.Scan(&id, &metricType, &value, &text, &at) != nil {
			continue
		}
		res := out[id]
		if res == nil {
			res = &nodeResources{Source: "ssh"}
			out[id] = res
		}
		if res.MeasuredAt == nil || at.After(*res.MeasuredAt) {
			t := at
			res.MeasuredAt = &t
		}
		v := value
		switch metricType {
		case "cpu_usage":
			res.CPUPercent = &v
		case "ram_usage":
			res.RAMPercent = &v
		case "disk_usage":
			res.DiskPercent = &v
		case "ram_total":
			res.RAMTotalMB = parseHumanSizeMB(strPtr(text))
		case "disk_total":
			res.DiskTotalMB = parseHumanSizeMB(strPtr(text))
		case "disk_used":
			res.DiskUsedMB = parseHumanSizeMB(strPtr(text))
		case "disk_available":
			res.DiskFreeMB = parseHumanSizeMB(strPtr(text))
		case "disk_quota":
			q := value > 0
			res.DiskQuota = &q
		}
	}
	for _, res := range out {
		if res.Source != "ssh" {
			continue
		}
		if res.RAMUsedMB == 0 && res.RAMTotalMB > 0 && res.RAMPercent != nil {
			res.RAMUsedMB = res.RAMTotalMB * *res.RAMPercent / 100
		}
		if res.DiskFreeMB == 0 && res.DiskTotalMB > 0 {
			res.DiskFreeMB = res.DiskTotalMB - res.DiskUsedMB
		}
		if res.DiskPercent == nil && res.DiskTotalMB > 0 && res.DiskUsedMB > 0 {
			p := res.DiskUsedMB / res.DiskTotalMB * 100
			res.DiskPercent = &p
		}
	}
	return out
}

func resourcesFromStats(raw []byte, at time.Time) *nodeResources {
	var stats struct {
		Host map[string]any `json:"host"`
	}
	if json.Unmarshal(raw, &stats) != nil || len(stats.Host) == 0 {
		return nil
	}
	host := stats.Host
	num := func(key string) (float64, bool) {
		v, ok := host[key].(float64)
		return v, ok
	}
	t := at
	res := &nodeResources{Source: "relay", MeasuredAt: &t}
	if v, ok := num("cpu_percent"); ok {
		res.CPUPercent = &v
	}
	if v, ok := num("ram_percent"); ok {
		res.RAMPercent = &v
	}
	if v, ok := num("disk_percent"); ok {
		res.DiskPercent = &v
	}
	res.RAMTotalMB, _ = num("ram_total_mb")
	res.RAMUsedMB, _ = num("ram_used_mb")
	res.DiskTotalMB, _ = num("disk_total_mb")
	res.DiskUsedMB, _ = num("disk_used_mb")
	if v, ok := num("disk_free_mb"); ok {
		res.DiskFreeMB = v
	} else if res.DiskTotalMB > 0 {
		res.DiskFreeMB = res.DiskTotalMB - res.DiskUsedMB
	}
	if q, ok := host["disk_quota"].(bool); ok {
		res.DiskQuota = &q
	}
	return res
}

func mbToBytesString(mb float64) string {
	if mb <= 0 {
		return ""
	}
	return strconv.FormatInt(int64(mb*1024*1024), 10)
}
