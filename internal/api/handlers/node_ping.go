package handlers

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vortanixapp/panel/pkg/nodeping"
)

const (
	nodePingCacheOK   = 15 * time.Minute
	nodePingCacheFail = 3 * time.Minute
	nodePingCheckTTL  = 3 * time.Second
)

func nodePingHost(candidates ...string) string {
	for _, c := range candidates {
		if c = strings.TrimSpace(c); c != "" {
			return c
		}
	}
	return ""
}

func (h *Handler) nodePingReachable(ctx context.Context, host string, port int) bool {
	if host == "" || port <= 0 {
		return false
	}
	key := "nodeping:" + host + ":" + strconv.Itoa(port)
	var reachable bool
	if found, err := h.cache.GetJSON(ctx, key, &reachable); err == nil && found {
		return reachable
	}
	checkCtx, cancel := context.WithTimeout(ctx, nodePingCheckTTL)
	defer cancel()
	reachable = nodeping.Available(checkCtx, host, port)
	ttl := nodePingCacheOK
	if !reachable {
		ttl = nodePingCacheFail
	}
	_ = h.cache.SetJSON(context.WithoutCancel(ctx), key, reachable, ttl)
	return reachable
}

func (h *Handler) nodePingReachableMany(ctx context.Context, hosts []string, ports []int) []bool {
	out := make([]bool, len(hosts))
	var wg sync.WaitGroup
	for i := range hosts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			out[i] = h.nodePingReachable(ctx, hosts[i], ports[i])
		}(i)
	}
	wg.Wait()
	return out
}
