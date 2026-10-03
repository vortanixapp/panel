package handlers

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	agentStatusTTL     = 2 * time.Second
	agentStatusTimeout = 20 * time.Second
	agentStatusMaxKeys = 2048
)

type agentStatusSnap struct {
	query    map[string]any
	queryErr error
	stats    map[string]any
	statsErr error
	at       time.Time
}

type agentStatusShare struct {
	mu    sync.Mutex
	snaps map[string]agentStatusSnap
	group singleflight.Group
}

var sharedAgentStatuses = &agentStatusShare{snaps: map[string]agentStatusSnap{}}

func (s *agentStatusShare) cached(key string) (agentStatusSnap, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, ok := s.snaps[key]
	if !ok || time.Since(snap.at) > agentStatusTTL {
		return agentStatusSnap{}, false
	}
	return snap, true
}

func (s *agentStatusShare) store(key string, snap agentStatusSnap) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.snaps) >= agentStatusMaxKeys {
		for k, v := range s.snaps {
			if time.Since(v.at) > agentStatusTTL {
				delete(s.snaps, k)
			}
		}
	}
	s.snaps[key] = snap
}

func (h *Handler) sharedAgentStatus(
	ctx context.Context,
	nodeID, serverID, gameID string,
	limits map[string]any,
	port int,
	withStats bool,
) agentStatusSnap {
	key := serverID
	if withStats {
		key += ":stats"
	}
	if snap, ok := sharedAgentStatuses.cached(key); ok {
		return snap
	}
	value, _, _ := sharedAgentStatuses.group.Do(key, func() (any, error) {
		if snap, ok := sharedAgentStatuses.cached(key); ok {
			return snap, nil
		}
		flight, cancel := context.WithTimeout(context.WithoutCancel(ctx), agentStatusTimeout)
		defer cancel()
		var snap agentStatusSnap
		snap.query, snap.queryErr = h.agentCommand(flight, nodeID, serverID, "game_query", map[string]any{
			"game_id": gameID,
			"limits":  limits,
			"port":    port,
		})
		if withStats {
			snap.stats, snap.statsErr = h.agentCommand(flight, nodeID, serverID, "stats", map[string]any{"limits": limits})
		}
		snap.at = time.Now()
		sharedAgentStatuses.store(key, snap)
		return snap, nil
	})
	return value.(agentStatusSnap)
}
