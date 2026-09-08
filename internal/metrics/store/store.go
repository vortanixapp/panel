package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/vortanix/vortanix/pkg/metricsquery"
	"github.com/vortanix/vortanix/pkg/tenantpools"
)

const (
	metricsListMax = 120
	metricsListTTL = 24 * time.Hour
)

type Store struct {
	db    *pgxpool.Pool
	rdb   *redis.Client
	pools *tenantpools.Pools
}

func New(db *pgxpool.Pool, rdb *redis.Client, pools *tenantpools.Pools) *Store {
	return &Store{db: db, rdb: rdb, pools: pools}
}

// poolFor выбирает базу арендатора: сервер и его метрики лежат там же, где
// остальные его данные. Не нашли — это ошибка, а не повод писать в
// центральную: там нет ни сервера, ни его метрик.
func (s *Store) poolFor(ctx context.Context, tenantID string) (*pgxpool.Pool, error) {
	if s.pools == nil {
		return s.db, nil
	}
	return s.pools.ForTenant(ctx, tenantID)
}

// poolForServer ищет базу по серверу: на чтение метрик арендатор не приходит,
// в запросе есть только идентификатор сервера.
func (s *Store) poolForServer(ctx context.Context, serverID string) (*pgxpool.Pool, error) {
	if s.pools == nil {
		return s.db, nil
	}
	for _, pool := range s.pools.All(ctx) {
		var exists bool
		if pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM core.servers WHERE id = $1)`, serverID).Scan(&exists) == nil && exists {
			return pool, nil
		}
	}
	return nil, fmt.Errorf("сервер %s не найден ни в одной базе арендатора", serverID)
}

type MetricPoint struct {
	TenantID   string
	ServerID   string
	CPUPct     float64
	MemUsedMB  int
	MemLimitMB int
	TS         time.Time
}

func (s *Store) ValidateServerTenant(ctx context.Context, serverID, tenantID string) (bool, error) {
	pool, err := s.poolFor(ctx, tenantID)
	if err != nil {
		return false, err
	}
	var exists bool
	err = pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM core.servers WHERE id = $1 AND tenant_id = $2)
	`, serverID, tenantID).Scan(&exists)
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

	pool, err := s.poolFor(ctx, p.TenantID)
	if err != nil {
		return err
	}
	_, err = pool.Exec(ctx, `
		INSERT INTO core.server_metric_points (server_id, tenant_id, ts, cpu_pct, mem_used_mb, mem_limit_mb)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, p.ServerID, p.TenantID, p.TS, p.CPUPct, p.MemUsedMB, p.MemLimitMB)
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

// PurgeOlderThan чистит метрики во всех базах: у каждого арендатора своя.
func (s *Store) PurgeOlderThan(ctx context.Context, days int) (int64, error) {
	pools := []*pgxpool.Pool{s.db}
	if s.pools != nil {
		pools = s.pools.All(ctx)
	}

	var total int64
	var firstErr error
	for _, pool := range pools {
		tag, err := pool.Exec(ctx, `
			DELETE FROM core.server_metric_points
			WHERE ts < now() - ($1::text || ' days')::interval
		`, days)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		total += tag.RowsAffected()
	}
	return total, firstErr
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
