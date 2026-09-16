package cache

import (
	"context"
	"time"
)

func (c *Cache) AllowWrite(ctx context.Context, scope string, limit int, window time.Duration) (bool, error) {
	return c.Allow(ctx, "rl:write:"+scope, limit, window)
}

func (c *Cache) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	if limit <= 0 {
		return true, nil
	}
	n, err := c.rdb.Incr(ctx, key).Result()
	if err != nil {
		return true, err
	}
	if n == 1 {
		_ = c.rdb.Expire(ctx, key, window).Err()
	}
	return n <= int64(limit), nil
}
