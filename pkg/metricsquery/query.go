package metricsquery

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TimescaleEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("TIMESCALE_ENABLED")))
	return v == "1" || v == "true" || v == "yes"
}

func BucketInterval(hours int) string {
	if hours > 72 {
		return "15 minutes"
	}
	if hours > 24 {
		return "5 minutes"
	}
	return ""
}

func BucketSeconds(hours int) int {
	switch {
	case hours <= 1:
		return 60
	case hours <= 6:
		return 120
	case hours <= 24:
		return 300
	case hours <= 24*7:
		return 1800
	case hours <= 24*30:
		return 7200
	case hours <= 24*90:
		return 21600
	default:
		return 43200
	}
}

func QueryServerMetrics(ctx context.Context, db *pgxpool.Pool, serverID string, hours, limit int) ([]map[string]any, error) {
	if limit <= 0 {
		limit = 500
	}
	if limit > 500 {
		limit = 500
	}
	if hours <= 0 {
		hours = 24
	}

	bucket := ""
	if TimescaleEnabled() {
		bucket = BucketInterval(hours)
	}

	if bucket == "" && !TimescaleEnabled() && hours > 24 {
		if out, err := queryRolledUp(ctx, db, serverID, hours, limit); err == nil {
			return out, nil
		}
	}

	var err error
	var rows interface {
		Next() bool
		Scan(dest ...any) error
		Close()
		Err() error
	}
	if bucket != "" {
		rows, err = db.Query(ctx, fmt.Sprintf(`
			SELECT EXTRACT(EPOCH FROM time_bucket('%s', ts))::bigint AS ts,
			       AVG(cpu_pct) AS cpu_pct,
			       AVG(mem_used_mb)::int AS mem_used_mb,
			       MAX(mem_limit_mb) AS mem_limit_mb
			FROM core.server_metric_points
			WHERE server_id = $1 AND ts >= now() - make_interval(hours => $2)
			GROUP BY 1
			ORDER BY 1 DESC
			LIMIT $3
		`, bucket), serverID, hours, limit)
	} else {
		rows, err = db.Query(ctx, `
			SELECT (floor(EXTRACT(EPOCH FROM ts) / $4) * $4)::bigint AS ts,
			       AVG(cpu_pct) AS cpu_pct,
			       AVG(mem_used_mb)::int AS mem_used_mb,
			       MAX(mem_limit_mb) AS mem_limit_mb
			FROM core.server_metric_points
			WHERE server_id = $1 AND ts >= now() - make_interval(hours => $2)
			GROUP BY 1
			ORDER BY 1 DESC
			LIMIT $3
		`, serverID, hours, limit, BucketSeconds(hours))
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]map[string]any, 0)
	for rows.Next() {
		var ts int64
		var cpu float64
		var memUsed, memLimit int
		if err := rows.Scan(&ts, &cpu, &memUsed, &memLimit); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"ts":           ts,
			"cpu_pct":      cpu,
			"mem_used_mb":  memUsed,
			"mem_limit_mb": memLimit,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func queryRolledUp(ctx context.Context, db *pgxpool.Pool, serverID string, hours, limit int) ([]map[string]any, error) {
	bucketSec := BucketSeconds(hours)
	rows, err := db.Query(ctx, `
		WITH wm AS (
			SELECT COALESCE(max(ts) + interval '5 minutes', 'epoch'::timestamptz) AS t
			FROM core.server_metric_rollups
			WHERE server_id = $1 AND ts >= now() - make_interval(hours => $2)
		), src AS (
			SELECT ts, cpu_avg AS cpu, mem_used_avg AS mem, mem_limit_mb AS lim, samples AS n
			FROM core.server_metric_rollups
			WHERE server_id = $1 AND ts >= now() - make_interval(hours => $2)
			UNION ALL
			SELECT to_timestamp(floor(extract(epoch FROM p.ts) / 300) * 300),
			       avg(p.cpu_pct), avg(p.mem_used_mb)::int, max(p.mem_limit_mb), count(*)::int
			FROM core.server_metric_points p, wm
			WHERE p.server_id = $1
			  AND p.ts >= GREATEST(now() - make_interval(hours => $2), wm.t)
			GROUP BY 1
		)
		SELECT (floor(extract(epoch FROM ts) / $4) * $4)::bigint AS ts,
		       sum(cpu * n) / sum(n) AS cpu_pct,
		       (sum(mem::bigint * n) / sum(n))::int AS mem_used_mb,
		       max(lim) AS mem_limit_mb
		FROM src
		GROUP BY 1
		ORDER BY 1 DESC
		LIMIT $3
	`, serverID, hours, limit, bucketSec)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]map[string]any, 0)
	for rows.Next() {
		var ts int64
		var cpu float64
		var memUsed, memLimit int
		if err := rows.Scan(&ts, &cpu, &memUsed, &memLimit); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"ts":           ts,
			"cpu_pct":      cpu,
			"mem_used_mb":  memUsed,
			"mem_limit_mb": memLimit,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}
