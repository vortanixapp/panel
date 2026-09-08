package cache

import (
	"context"
	"fmt"
	"time"
)

func (c *Cache) AllowWrite(ctx context.Context, tenantID string, limit int, window time.Duration) (bool, error) {
	if tenantID == "" || limit <= 0 {
		return true, nil
	}
	key := fmt.Sprintf("rl:write:%s", tenantID)
	n, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return true, err
	}
	if n == 1 {
		_ = c.rdb.Expire(ctx, key, window).Err()
	}
	return n <= int64(limit), nil
}
