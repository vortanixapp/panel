package handlers

import (
	"strconv"
	"strings"
	"time"
)

const (
	liveRingCap        = 512
	liveReplayMaxAge   = 30 * time.Minute
	liveReplayMaxItems = 20
)

type liveEvent struct {
	ms    int64
	topic string
}

type liveRing struct {
	events []liveEvent
	lastMS int64
	from   int64
}

func newLiveRing() liveRing {
	now := time.Now().UnixMilli()
	return liveRing{lastMS: now, from: now}
}

func (l *notifyLive) markGap() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now := time.Now().UnixMilli(); now > l.ring.from {
		l.ring.from = now
	}
}

func (l *notifyLive) record(topic string) int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	ms := time.Now().UnixMilli()
	if ms <= l.ring.lastMS {
		ms = l.ring.lastMS + 1
	}
	l.ring.lastMS = ms
	l.ring.events = append(l.ring.events, liveEvent{ms: ms, topic: topic})
	if over := len(l.ring.events) - liveRingCap; over > 0 {
		if dropped := l.ring.events[over-1].ms; dropped > l.ring.from {
			l.ring.from = dropped
		}
		l.ring.events = append([]liveEvent(nil), l.ring.events[over:]...)
	}
	return ms
}

func (l *notifyLive) now() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	if ms := time.Now().UnixMilli(); ms > l.ring.lastMS {
		return ms
	}
	return l.ring.lastMS
}

func (l *notifyLive) since(ms int64) ([]liveEvent, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if ms <= 0 || ms < l.ring.from {
		return nil, false
	}
	var out []liveEvent
	for _, ev := range l.ring.events {
		if ev.ms > ms {
			out = append(out, ev)
		}
	}
	return out, true
}

func stampInvalidate(topic string, ms int64) string {
	return liveInvalidate + topic + "@" + strconv.FormatInt(ms, 10)
}

func splitStamp(rest string) (string, int64) {
	i := strings.LastIndexByte(rest, '@')
	if i < 0 {
		return rest, 0
	}
	ms, err := strconv.ParseInt(rest[i+1:], 10, 64)
	if err != nil {
		return rest, 0
	}
	return rest[:i], ms
}

func parseLiveCursor(raw string) (int64, string) {
	raw = strings.TrimSpace(raw)
	ms, cursor, _ := strings.Cut(raw, "|")
	n, err := strconv.ParseInt(ms, 10, 64)
	if err != nil || n <= 0 {
		return 0, cursor
	}
	return n, cursor
}
