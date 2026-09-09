package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vortanixapp/panel/pkg/protocol"
)

func TenantChannel(tenantID string) string {
	return protocol.TenantEventsChannel(tenantID)
}

func ConsoleChannel(sessionID string) string {
	return protocol.ConsoleChannel(sessionID)
}

func PublishTenantEvent(ctx context.Context, rdb *redis.Client, tenantID string, ev protocol.TenantEvent) {
	b, _ := json.Marshal(ev)
	_ = rdb.Publish(ctx, TenantChannel(tenantID), b).Err()
}

func PublishConsoleOutput(ctx context.Context, rdb *redis.Client, sessionID, data string) {
	_ = rdb.Publish(ctx, ConsoleChannel(sessionID), data).Err()
}

func StoreMetricPoint(ctx context.Context, rdb *redis.Client, serverID string, cpu float64, memUsed, memLimit int) error {
	key := "metrics:" + serverID
	point, _ := json.Marshal(map[string]any{
		"ts":           time.Now().Unix(),
		"cpu_pct":      cpu,
		"mem_used_mb":  memUsed,
		"mem_limit_mb": memLimit,
	})
	pipe := rdb.Pipeline()
	pipe.LPush(ctx, key, point)
	pipe.LTrim(ctx, key, 0, 119)
	pipe.Expire(ctx, key, 24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

func GetMetrics(ctx context.Context, rdb *redis.Client, serverID string, limit int64) ([]map[string]any, error) {
	key := "metrics:" + serverID
	vals, err := rdb.LRange(ctx, key, 0, limit-1).Result()
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

func MetricsKey(serverID string) string {
	return fmt.Sprintf("metrics:%s", serverID)
}
