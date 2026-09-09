package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/vortanixapp/panel/pkg/metricsquery"
)

const (
	metricsListMax = 120
	metricsListTTL = 24 * time.Hour
)

type Store struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

func New(db *pgxpool.Pool, rdb *redis.Client) *Store {
	return &Store{db: db, rdb: rdb}
}

func (s *Store) poolFor(context.Context) (*pgxpool.Pool, error) {
	return s.db, nil
}

func (s *Store) poolForServer(context.Context, string) (*pgxpool.Pool, error) {
	return s.db, nil
}

type MetricPoint struct {
	ServerID   string
	CPUPct     float64
	MemUsedMB  int
	MemLimitMB int
	TS         time.Time
}

func (s *Store) ValidateServer(ctx context.Context, serverID string) (bool, error) {
	pool, err := s.poolFor(ctx)
	if err != nil {
		return false, err
	}
	var exists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM core.servers WHERE id = $1)
	`, serverID).Scan(&exists)
	return exists, err
}

func (s *Store) StoreMetricPoint(ctx context.Context, p MetricPoint) error {
	if p.TS.IsZero() {
		p.TS = time.Now().UTC()
	}

	key := "metrics:" + p.ServerID
	point, err := json.Marshal(map[string]any{
		"ts":           p.TS.Unix(),
		"cpu_pct":      p.CPUPct,
		"mem_used_mb":  p.MemUsedMB,
		"mem_limit_mb": p.MemLimitMB,
	})
	if err != nil {
		return err
	}

	pipe := s.rdb.Pipeline()
	pipe.LPush(ctx, key, point)
	pipe.LTrim(ctx, key, 0, metricsListMax-1)
	pipe.Expire(ctx, key, metricsListTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	pool, err := s.poolFor(ctx)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO core.server_metric_points (server_id, ts, cpu_pct, mem_used_mb, mem_limit_mb)
		VALUES ($1, $2, $3, $4, $5)
	`, p.ServerID, p.TS, p.CPUPct, p.MemUsedMB, p.MemLimitMB)
	return err
}

func (s *Store) GetMetrics(ctx context.Context, serverID string, limit int64, hours int) ([]map[string]any, error) {
	if hours > 0 {
		return s.getMetricsFromPG(ctx, serverID, hours, limit)
	}
	return s.getMetricsFromRedis(ctx, serverID, limit)
}

func (s *Store) getMetricsFromRedis(ctx context.Context, serverID string, limit int64) ([]map[string]any, error) {
	key := "metrics:" + serverID
	vals, err := s.rdb.LRange(ctx, key, 0, limit-1).Result()
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(vals))
	for i := len(vals) - 1; i >= 0; i-- {
		var m map[string]any
		if json.Unmarshal([]byte(vals[i]), &m) == nil {
			out = append(out, m)
		}
	}
	return out, nil
}

func (s *Store) getMetricsFromPG(ctx context.Context, serverID string, hours int, limit int64) ([]map[string]any, error) {
	pool, err := s.poolForServer(ctx, serverID)
	if err != nil {
		return nil, err
	}
	return metricsquery.QueryServerMetrics(ctx, pool, serverID, hours, int(limit))
}

func (s *Store) PurgeOlderThan(ctx context.Context, days int) (int64, error) {
	tag, err := s.db.Exec(ctx, `
		DELETE FROM core.server_metric_points
		WHERE ts < now() - ($1::text || ' days')::interval
	`, days)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *Store) ServerExists(ctx context.Context, serverID string) (bool, error) {
	pool, err := s.poolForServer(ctx, serverID)
	if err != nil {
		return false, err
	}
	var exists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM core.servers WHERE id = $1)
	`, serverID).Scan(&exists)
	return exists, err
}
