package hub

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentConn struct {
	NodeID      string
	DB          *pgxpool.Pool
	Conn        *websocket.Conn
	Send        chan OutboundMessage
	ConnectedAt time.Time
	RemoteAddr  string

	pingSentAt atomic.Int64
	rttMs      atomic.Int64
}

type OutboundMessage struct {
	MessageType int
	Data        []byte
}

type Hub struct {
	mu     sync.RWMutex
	agents map[string]*AgentConn
}

func New() *Hub {
	return &Hub{agents: make(map[string]*AgentConn)}
}

func (h *Hub) Register(c *AgentConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if old, ok := h.agents[c.NodeID]; ok && old != c {
		log.Printf("replacing agent connection node=%s", c.NodeID)
		close(old.Send)
		_ = old.Conn.Close()
	}
	h.agents[c.NodeID] = c
}

func (h *Hub) Unregister(nodeID string, conn *AgentConn) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	c, ok := h.agents[nodeID]
	if !ok || c != conn {
		return false
	}
	close(c.Send)
	_ = c.Conn.Close()
	delete(h.agents, nodeID)
	return true
}

func (h *Hub) SendCommand(nodeID string, payload []byte) error {
	return h.send(nodeID, OutboundMessage{MessageType: websocket.TextMessage, Data: payload})
}

func (h *Hub) SendBinary(nodeID string, payload []byte) error {
	return h.send(nodeID, OutboundMessage{MessageType: websocket.BinaryMessage, Data: payload})
}

func (h *Hub) send(nodeID string, msg OutboundMessage) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c, ok := h.agents[nodeID]
	if !ok {
		return ErrNodeOffline
	}
	select {
	case c.Send <- msg:
		return nil
	default:
		return ErrNodeOffline
	}
}

func (h *Hub) Get(nodeID string) *AgentConn {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.agents[nodeID]
}

func (h *Hub) Connected() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.agents))
	for id := range h.agents {
		ids = append(ids, id)
	}
	return ids
}

const (
	PingInterval = 25 * time.Second
	PongWait     = 70 * time.Second
	writeWait    = 10 * time.Second
)

func (c *AgentConn) MarkPong() {
	sent := c.pingSentAt.Load()
	if sent == 0 {
		return
	}
	if rtt := time.Since(time.Unix(0, sent)); rtt >= 0 && rtt < PongWait {
		c.rttMs.Store(rtt.Milliseconds())
	}
}

func (c *AgentConn) RTT() int {
	return int(c.rttMs.Load())
}

func (c *AgentConn) WritePump() {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				return
			}
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(msg.MessageType, msg.Data); err != nil {
				log.Printf("write error node=%s: %v", c.NodeID, err)
				return
			}
		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			c.pingSentAt.Store(time.Now().UnixNano())
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("ping error node=%s: %v", c.NodeID, err)
				return
			}
		}
	}
}

var ErrNodeOffline = errOffline{}

type errOffline struct{}

func (errOffline) Error() string { return "node offline" }
