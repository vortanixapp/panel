package metricsquery

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	RollupBucket = 5 * time.Minute
	rollupLag    = time.Hour
	rollupChunk  = 6 * time.Hour
)

type MaintainResult struct {
	RolledUp      int64
	RawDeleted    int64
	RollupDeleted int64
}

func Rollup(ctx context.Context, db *pgxpool.Pool) (int64, error) {
	var start *time.Time
	if err := db.QueryRow(ctx, `
		SELECT COALESCE(
			(SELECT max(ts) + interval '5 minutes' FROM core.server_metric_rollups),
			(SELECT min(ts) FROM core.server_metric_points))
	`).Scan(&start); err != nil {
		return 0, err
	}
	if start == nil {
		return 0, nil
	}
	cur := start.UTC().Truncate(RollupBucket)
	end := time.Now().UTC().Add(-rollupLag).Truncate(RollupBucket)

	var total int64
	for cur.Before(end) {
		next := cur.Add(rollupChunk)
		if next.After(end) {
			next = end
		}
		tag, err := db.Exec(ctx, `
			INSERT INTO core.server_metric_rollups
				(server_id, ts, cpu_avg, cpu_max, mem_used_avg, mem_used_max, mem_limit_mb, samples)
			SELECT server_id,
			       to_timestamp(floor(extract(epoch FROM ts) / 300) * 300) AS bucket,
			       avg(cpu_pct), max(cpu_pct),
			       avg(mem_used_mb)::int, max(mem_used_mb),
			       max(mem_limit_mb), count(*)::int
			FROM core.server_metric_points
			WHERE ts >= $1 AND ts < $2
			GROUP BY server_id, bucket
			ON CONFLICT (server_id, ts) DO NOTHING
		`, cur, next)
		if err != nil {
			return total, err
		}
		total += tag.RowsAffected()
		cur = next
	}
	return total, nil
}

func Maintain(ctx context.Context, db *pgxpool.Pool, rawDays, rollupDays int) (MaintainResult, error) {
	var res MaintainResult
	rolled, err := Rollup(ctx, db)
	res.RolledUp = rolled
	if err != nil {
		return res, err
	}
	tag, err := db.Exec(ctx, `
		DELETE FROM core.server_metric_points
		WHERE ts < now() - make_interval(days => $1)
		  AND ts < now() - interval '1 hour'
	`, rawDays)
	if err != nil {
		return res, err
	}
	res.RawDeleted = tag.RowsAffected()
	tag, err = db.Exec(ctx, `
		DELETE FROM core.server_metric_rollups
		WHERE ts < now() - make_interval(days => $1)
	`, rollupDays)
	if err != nil {
		return res, err
	}
	res.RollupDeleted = tag.RowsAffected()
	return res, nil
}
