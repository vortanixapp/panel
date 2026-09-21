package agent

import (
	"sort"
	"sync"
	"time"
)

type opEntry struct {
	ServerID string    `json:"server_id,omitempty"`
	Kind     string    `json:"kind"`
	Since    time.Time `json:"since"`
}

type opsRegistry struct {
	mu  sync.Mutex
	ops map[string]opEntry
}

func (o *opsRegistry) begin(serverID, kind string) func() {
	key := serverID + "|" + kind
	o.mu.Lock()
	if o.ops == nil {
		o.ops = map[string]opEntry{}
	}
	o.ops[key] = opEntry{ServerID: serverID, Kind: kind, Since: time.Now()}
	o.mu.Unlock()
	return func() {
		o.mu.Lock()
		delete(o.ops, key)
		o.mu.Unlock()
	}
}

func (o *opsRegistry) list() []opEntry {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]opEntry, 0, len(o.ops))
	for _, op := range o.ops {
		out = append(out, op)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Since.Before(out[j].Since) })
	return out
}

func (o *opsRegistry) serverOp(serverID string) (string, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, op := range o.ops {
		if op.ServerID == serverID {
			return op.Kind, true
		}
	}
	return "", false
}
