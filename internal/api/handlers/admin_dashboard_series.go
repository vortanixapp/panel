package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
)

func (h *Handler) AdminDashboardSeries(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
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
				to_timestamp(floor(extract(epoch FROM measured_at) / ($3 * 60)) * ($3 * 60))
					AT TIME ZONE 'UTC',
				'YYYY-MM-DD"T"HH24:MI:SS"Z"'
			) AS bucket,
			AVG(value) FILTER (WHERE metric_type = 'cpu_usage') AS cpu,
			AVG(value) FILTER (WHERE metric_type = 'ram_usage') AS ram
		FROM core.node_metrics
		WHERE tenant_id = $1
		  AND metric_type IN ('cpu_usage', 'ram_usage')
		  -- make_interval, а не склейка строк: интервал из числового
		  -- параметра нельзя собрать через «$2 || ' hours'» — Postgres не
		  -- выведет тип, а приведение к тексту ломает кодирование на
		  -- стороне pgx. Из-за этого запрос падал, обработчик отвечал
		  -- «database error», и график молча показывал «Данных пока нет»,
		  -- хотя метрики в базе были.
		  AND measured_at >= now() - make_interval(hours => $2)
		  AND ($4 = '' OR node_id = $4::uuid)
		GROUP BY bucket
		ORDER BY bucket
	`, claims.TenantID, hours, bucketMinutes, nodeID)
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
		SELECT id::text, name FROM core.nodes WHERE tenant_id = $1 ORDER BY name
	`, claims.TenantID)
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
