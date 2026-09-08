package hub

import (
	"sync"
	"time"

	"github.com/vortanix/vortanix/pkg/protocol"
)

type CommandWaiter struct {
	mu    sync.Mutex
	waits map[string]chan protocol.AckMessage
}

func NewCommandWaiter() *CommandWaiter {
	return &CommandWaiter{waits: make(map[string]chan protocol.AckMessage)}
}

func (w *CommandWaiter) Register(commandID string) <-chan protocol.AckMessage {
	ch := make(chan protocol.AckMessage, 1)
	w.mu.Lock()
	w.waits[commandID] = ch
	w.mu.Unlock()
	return ch
}

func (w *CommandWaiter) Cancel(commandID string) {
	w.mu.Lock()
	if ch, ok := w.waits[commandID]; ok {
		delete(w.waits, commandID)
		close(ch)
	}
	w.mu.Unlock()
}

func (w *CommandWaiter) Complete(ack protocol.AckMessage) bool {
	w.mu.Lock()
	ch, ok := w.waits[ack.CommandID]
	if ok {
		delete(w.waits, ack.CommandID)
	}
	w.mu.Unlock()
	if !ok {
		return false
	}
	select {
	case ch <- ack:
	default:
	}
	close(ch)
	return true
}

func (w *CommandWaiter) Wait(commandID string, ch <-chan protocol.AckMessage, timeout time.Duration) (protocol.AckMessage, bool) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case ack, ok := <-ch:
		if !ok {
			return protocol.AckMessage{}, false
		}
		return ack, true
	case <-timer.C:
		w.Cancel(commandID)
		return protocol.AckMessage{}, false
	}
}
