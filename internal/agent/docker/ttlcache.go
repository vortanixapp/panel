package docker

import (
	"sync"
	"time"
)

const ttlCacheMaxKeys = 1024

type ttlEntry[T any] struct {
	value T
	at    time.Time
}

type ttlCache[T any] struct {
	mu  sync.Mutex
	ttl time.Duration
	m   map[string]ttlEntry[T]
}

func newTTLCache[T any](ttl time.Duration) *ttlCache[T] {
	return &ttlCache[T]{ttl: ttl, m: map[string]ttlEntry[T]{}}
}

func (c *ttlCache[T]) get(key string) (T, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.m[key]
	if !ok || time.Since(entry.at) > c.ttl {
		var zero T
		return zero, false
	}
	return entry.value, true
}

func (c *ttlCache[T]) set(key string, value T) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.m) >= ttlCacheMaxKeys {
		for k, e := range c.m {
			if time.Since(e.at) > c.ttl {
				delete(c.m, k)
			}
		}
	}
	c.m[key] = ttlEntry[T]{value: value, at: time.Now()}
}

func (c *ttlCache[T]) forget(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.m, key)
}
