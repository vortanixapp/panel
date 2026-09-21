package agent

import "time"

const (
	outboxMaxItems = 512
	outboxMaxBytes = 2 << 20
	outboxAckTTL   = 30 * time.Minute
)

type outItem struct {
	key  string
	ack  bool
	data []byte
	at   time.Time
}

type outbox struct {
	items []outItem
	bytes int
}

func (o *outbox) push(key string, ack bool, data []byte) {
	if key != "" {
		for i, it := range o.items {
			if it.key == key {
				o.bytes -= len(it.data)
				o.items = append(o.items[:i], o.items[i+1:]...)
				break
			}
		}
	}
	o.items = append(o.items, outItem{key: key, ack: ack, data: data, at: time.Now()})
	o.bytes += len(data)
	for len(o.items) > outboxMaxItems || o.bytes > outboxMaxBytes {
		drop := 0
		for i, it := range o.items {
			if !it.ack {
				drop = i
				break
			}
		}
		o.bytes -= len(o.items[drop].data)
		o.items = append(o.items[:drop], o.items[drop+1:]...)
	}
}

func (o *outbox) drain() []outItem {
	items := o.items
	o.items = nil
	o.bytes = 0
	now := time.Now()
	kept := items[:0]
	for _, it := range items {
		if it.ack && now.Sub(it.at) > outboxAckTTL {
			continue
		}
		kept = append(kept, it)
	}
	return kept
}

func (o *outbox) restore(items []outItem) {
	if len(items) == 0 {
		return
	}
	rest := o.items
	o.items = append(append([]outItem{}, items...), rest...)
	o.bytes = 0
	for _, it := range o.items {
		o.bytes += len(it.data)
	}
}

func (o *outbox) size() int {
	return len(o.items)
}
