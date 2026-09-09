package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
)

func (h *Handler) AdminDashboardSeries(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	hours := 24
	if v, err := strconv.Atoi(r.URL.Query().Get("hours")); err == nil && v > 0 && v <= 168 {
		hours = v
	}
	bucketMinutes := 15
	if hours > 48 {
		bucketMinutes = 60
	}
	nodeID := strings.TrimSpace(r.URL.Query().Get("node_id"))

	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT
			to_char(
				to_timestamp(floor(extract(epoch FROM measured_at) / ($2 * 60)) * ($2 * 60))
					AT TIME ZONE 'UTC',
				'YYYY-MM-DD"T"HH24:MI:SS"Z"'
			) AS bucket,
			AVG(value) FILTER (WHERE metric_type = 'cpu_usage') AS cpu,
			AVG(value) FILTER (WHERE metric_type = 'ram_usage') AS ram
		FROM core.node_metrics
		WHERE metric_type IN ('cpu_usage', 'ram_usage')
		  AND measured_at >= now() - make_interval(hours => $1)
		  AND ($3 = '' OR node_id = $3::uuid)
		GROUP BY bucket
		ORDER BY bucket
	`, hours, bucketMinutes, nodeID)
	if err != nil {
		log.Printf("dashboard series: %v", err)
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	points := []map[string]any{}
	for rows.Next() {
		var bucket string
		var cpu, ram *float64
		if rows.Scan(&bucket, &cpu, &ram) != nil {
			continue
		}
		points = append(points, map[string]any{"ts": bucket, "cpu": cpu, "ram": ram})
	}

	nodeRows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT id::text, name FROM core.nodes ORDER BY name
	`)
	nodes := []map[string]any{}
	if err == nil {
		defer nodeRows.Close()
		for nodeRows.Next() {
			var id, name string
			if nodeRows.Scan(&id, &name) == nil {
				nodes = append(nodes, map[string]any{"id": id, "name": name})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"points":         points,
		"nodes":          nodes,
		"hours":          hours,
		"bucket_minutes": bucketMinutes,
		"node_id":        nodeID,
	})
}
