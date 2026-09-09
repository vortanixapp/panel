package handlers

import (
	"context"
	"errors"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type onlineBucket struct {
	online     int
	maxPlayers int
}

func (h *Handler) onlinePointsByBucket(
	ctx context.Context, serverID string, hours, bucketSec int,
) map[int64]onlineBucket {
	out := map[int64]onlineBucket{}
	if bucketSec <= 0 {
		return out
	}
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT (floor(EXTRACT(EPOCH FROM ts) / $3) * $3)::bigint AS bucket,
		       MAX(online), MAX(max_players)
		FROM core.server_online_points
		WHERE server_id = $1::uuid AND ts >= now() - ($2::text || ' hours')::interval
		GROUP BY 1
		ORDER BY 1 DESC
		LIMIT 1000
	`, serverID, strconv.Itoa(hours), bucketSec)
	if err != nil {
		log.Printf("онлайн сервера %s: %v", serverID, err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var bucket int64
		var online, maxPlayers int
		if err := rows.Scan(&bucket, &online, &maxPlayers); err != nil {
			return out
		}
		out[bucket] = onlineBucket{online: online, maxPlayers: maxPlayers}
	}
	return out
}

func (h *Handler) MonitoringServerStats(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	access, err := h.resolveServerAccess(r.Context(), claims.TenantID, claims.UserID, claims.Role, serverID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	perms := viewerPermissionsForAccess(access)
	if !access.IsOwner && !access.IsStaff && !viewerHasPermission(perms, "can_view_metrics") {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	days := parseMonitoringDays(r.URL.Query().Get("days"))
	series := h.buildMonitoringStatsSeries(r.Context(), serverID, days)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"days":   days,
		"series": series,
	})
}

func parseMonitoringDays(raw string) int {
	allowed := map[int]bool{1: true, 7: true, 14: true, 30: true, 60: true, 90: true, 180: true}
	if raw == "" {
		return 1
	}
	n, err := strconv.Atoi(raw)
	if err != nil || !allowed[n] {
		return 1
	}
	return n
}

func (h *Handler) buildMonitoringStatsSeries(ctx context.Context, serverID string, days int) map[string]any {
	hours := days * 24
	limit := 500
	if days > 30 {
		limit = 2000
	}
	points, err := getMetricsFromPG(ctx, h.dbOf(ctx), serverID, hours, limit)
	if err != nil {
		log.Printf("история метрик сервера %s: %v", serverID, err)
	}
	if len(points) == 0 {
		points, _ = h.cache.GetMetrics(ctx, serverID, 500)
	}

	bucketSec, fmtLayout := monitoringBucketConfig(days)
	buckets := map[int64]map[string]any{}
	online := h.onlinePointsByBucket(ctx, serverID, hours, bucketSec)

	for _, p := range points {
		ts, ok := metricTS(p)
		if !ok {
			continue
		}
		bucket := (ts / int64(bucketSec)) * int64(bucketSec)
		cpu := floatFromAny(p["cpu_pct"])
		ram := 0.0
		if memUsed, ok := p["mem_used_mb"]; ok {
			if memLimit, ok2 := p["mem_limit_mb"]; ok2 {
				limitMB := floatFromAny(memLimit)
				if limitMB > 0 {
					ram = floatFromAny(memUsed) / limitMB * 100
				}
			}
		}
		disk := floatFromAny(p["disk_percent"])

		cur, exists := buckets[bucket]
		if !exists {
			buckets[bucket] = map[string]any{
				"ts": bucket, "cpu": cpu, "ram": ram, "disk": disk,
			}
			continue
		}
		if cpu > floatFromAny(cur["cpu"]) {
			cur["cpu"] = cpu
		}
		if ram > floatFromAny(cur["ram"]) {
			cur["ram"] = ram
		}
		if disk > floatFromAny(cur["disk"]) {
			cur["disk"] = disk
		}
	}

	for k := range online {
		if _, ok := buckets[k]; !ok {
			buckets[k] = map[string]any{"ts": k}
		}
	}

	keys := make([]int64, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	labels := []string{}
	onlineArr := []any{}
	maxArr := []any{}
	cpuArr := []any{}
	ramArr := []any{}
	diskArr := []any{}
	var maxCap *int

	for _, k := range keys {
		b := buckets[k]
		labels = append(labels, time.Unix(k, 0).UTC().Format(fmtLayout))
		if o, ok := online[k]; ok {
			b["online"] = o.online
			b["max"] = o.maxPlayers
			onlineArr = append(onlineArr, o.online)
			maxArr = append(maxArr, nullableInt(o.maxPlayers))
		} else {
			onlineArr = append(onlineArr, nil)
			maxArr = append(maxArr, nil)
		}
		cpuArr = append(cpuArr, roundNullableFloat(b["cpu"]))
		ramArr = append(ramArr, roundNullableFloat(b["ram"]))
		diskArr = append(diskArr, roundNullableFloat(b["disk"]))
		if mx := intFromAny(b["max"]); mx > 0 {
			if maxCap == nil || mx > *maxCap {
				v := mx
				maxCap = &v
			}
		}
	}

	var maxCapAny any
	if maxCap != nil {
		maxCapAny = *maxCap
	}
	return map[string]any{
		"labels": labels, "online": onlineArr, "max": maxArr,
		"cpu": cpuArr, "ram": ramArr, "disk": diskArr, "max_cap": maxCapAny,
	}
}

func monitoringBucketConfig(days int) (bucketSec int, fmtLayout string) {
	switch {
	case days == 1:
		return 600, "15:04"
	case days <= 7:
		return 3600, "02.01 15:04"
	case days <= 30:
		return 14400, "02.01 15:04"
	default:
		return 86400, "02.01.2006"
	}
}

func nullableInt(v any) any {
	if v == nil {
		return nil
	}
	n := intFromAny(v)
	return n
}

func roundNullableFloat(v any) any {
	if v == nil {
		return nil
	}
	f := floatFromAny(v)
	if f <= 0 {
		return nil
	}
	return mathRound(f, 2)
}

func mathRound(v float64, places int) float64 {
	p := math.Pow(10, float64(places))
	return math.Round(v*p) / p
}
