package handlers

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vortanixapp/panel/pkg/notify"
)

const liveSync = "sync"

type notifyLive struct {
	mu   sync.Mutex
	subs map[string]map[chan string]struct{}
}

func (h *Handler) StartNotificationsLive(ctx context.Context) {
	h.live = &notifyLive{subs: map[string]map[chan string]struct{}{}}
	go h.live.run(ctx, h.db)
}

func (l *notifyLive) subscribe(userID string) (<-chan string, func()) {
	ch := make(chan string, 16)
	l.mu.Lock()
	set := l.subs[userID]
	if set == nil {
		set = map[chan string]struct{}{}
		l.subs[userID] = set
	}
	set[ch] = struct{}{}
	l.mu.Unlock()

	return ch, func() {
		l.mu.Lock()
		if set := l.subs[userID]; set != nil {
			delete(set, ch)
			if len(set) == 0 {
				delete(l.subs, userID)
			}
		}
		l.mu.Unlock()
	}
}

func (l *notifyLive) deliver(userID, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for ch := range l.subs[userID] {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (l *notifyLive) broadcast(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, set := range l.subs {
		for ch := range set {
			select {
			case ch <- msg:
			default:
			}
		}
	}
}

func (l *notifyLive) run(ctx context.Context, pool *pgxpool.Pool) {
	wait := time.Second
	for ctx.Err() == nil {
		listened, err := l.listen(ctx, pool)
		if ctx.Err() != nil {
			return
		}
		if listened {
			wait = time.Second
		}
		if err != nil {
			log.Printf("оповещения: подписка на поток прервана: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		if wait < 30*time.Second {
			wait *= 2
		}
	}
}

func (l *notifyLive) listen(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	pooled, err := pool.Acquire(ctx)
	if err != nil {
		return false, err
	}
	conn := pooled.Hijack()
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = conn.Close(closeCtx)
	}()

	if _, err := conn.Exec(ctx, "LISTEN "+notify.LiveChannel); err != nil {
		return false, err
	}
	l.broadcast(liveSync)

	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return true, err
		}
		userID, msg, ok := strings.Cut(n.Payload, ":")
		if !ok || userID == "" || msg == "" {
			continue
		}
		l.deliver(userID, msg)
	}
}
