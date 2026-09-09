package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	rdb *redis.Client
}

func New(redisURL string) (*Cache, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}
	return &Cache{rdb: redis.NewClient(opts)}, nil
}

func (c *Cache) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Cache) Close() error {
	return c.rdb.Close()
}

func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *Cache) SetServerStatus(ctx context.Context, serverID, status string) error {
	key := "srv:" + serverID + ":status"
	return c.rdb.Set(ctx, key, status, 30*time.Second).Err()
}

func (c *Cache) GetServerStatus(ctx context.Context, serverID string) (string, bool) {
	val, err := c.rdb.Get(ctx, "srv:"+serverID+":status").Result()
	if err != nil {
		return "", false
	}
	return val, true
}

func (c *Cache) InvalidateTenantServers(ctx context.Context) error {
	var cursor uint64
	for {
		keys, next, err := c.rdb.Scan(ctx, cursor, "panel:servers:*", 200).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		if next == 0 {
			return nil
		}
		cursor = next
	}
}

func (c *Cache) InvalidateTenantNodes(ctx context.Context) error {
	return c.rdb.Del(ctx, "panel:nodes").Err()
}

func (c *Cache) InstallLog(ctx context.Context, serverID string, limit int64) ([]string, error) {
	if limit <= 0 {
		limit = 200
	}
	return c.rdb.LRange(ctx, "install:log:"+serverID, -limit, -1).Result()
}

func (c *Cache) GetJSON(ctx context.Context, key string, dest any) (bool, error) {
	val, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal([]byte(val), dest)
}

func (c *Cache) SetJSON(ctx context.Context, key string, v any, ttl time.Duration) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, b, ttl).Err()
}

func (c *Cache) Claim(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return c.rdb.SetNX(ctx, key, "1", ttl).Result()
}
