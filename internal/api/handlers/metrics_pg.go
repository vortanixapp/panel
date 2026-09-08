package handlers

import (
	"context"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vortanixapp/panel/pkg/metricsquery"
)

func getMetricsFromPG(ctx context.Context, db *pgxpool.Pool, serverID string, hours, limit int) ([]map[string]any, error) {
	return metricsquery.QueryServerMetrics(ctx, db, serverID, hours, limit)
}

func mergeMetricPoints(redisPoints, pgPoints []map[string]any) []map[string]any {
	byTS := make(map[int64]map[string]any)

	for _, p := range pgPoints {
		if ts, ok := metricTS(p); ok {
			byTS[ts] = p
		}
	}
	for _, p := range redisPoints {
		if ts, ok := metricTS(p); ok {
			byTS[ts] = p
		}
	}

	keys := make([]int64, 0, len(byTS))
	for ts := range byTS {
		keys = append(keys, ts)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	out := make([]map[string]any, 0, len(keys))
	for _, ts := range keys {
		out = append(out, byTS[ts])
	}
	return out
}

func metricTS(p map[string]any) (int64, bool) {
	switch v := p["ts"].(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	default:
		return 0, false
	}
}
