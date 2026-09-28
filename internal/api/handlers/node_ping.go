package handlers

import (
	"context"
	"strconv"
	"time"

	"github.com/vortanixapp/panel/pkg/nodeping"
)

const (
	nodePingCacheOK   = 15 * time.Minute
	nodePingCacheFail = 3 * time.Minute
	nodePingCheckTTL  = 5 * time.Second
)

func (h *Handler) nodePingReachable(ctx context.Context, host string, port int) bool {
	if host == "" || port <= 0 {
		return false
	}
	key := "nodeping:" + host + ":" + strconv.Itoa(port)
	var reachable bool
	if found, err := h.cache.GetJSON(ctx, key, &reachable); err == nil && found {
		return reachable
	}
	go h.checkNodePing(key, host, port)
	return true
}

func (h *Handler) checkNodePing(key, host string, port int) {
	ctx, cancel := context.WithTimeout(context.Background(), nodePingCheckTTL)
	defer cancel()
	reachable := nodeping.Available(ctx, host, port)
	ttl := nodePingCacheOK
	if !reachable {
		ttl = nodePingCacheFail
	}
	_ = h.cache.SetJSON(ctx, key, reachable, ttl)
}
