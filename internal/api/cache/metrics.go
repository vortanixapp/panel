package cache

import (
	"context"
	"encoding/json"
	"time"
)

func (c *Cache) SetConsoleTicket(ctx context.Context, ticket string, data any, ttl time.Duration) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, "console:ticket:"+ticket, b, ttl).Err()
}

func (c *Cache) GetConsoleTicket(ctx context.Context, ticket string, dest any) (bool, error) {
	return c.GetJSON(ctx, "console:ticket:"+ticket, dest)
}

func (c *Cache) GetMetrics(ctx context.Context, serverID string, limit int64) ([]map[string]any, error) {
	key := "metrics:" + serverID
	vals, err := c.rdb.LRange(ctx, key, 0, limit-1).Result()
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
