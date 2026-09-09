package hub

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentConn struct {
	NodeID   string
	TenantID string
	DB       *pgxpool.Pool
	Conn     *websocket.Conn
	Send     chan OutboundMessage
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
	h.mu.RLock()
	c, ok := h.agents[nodeID]
	h.mu.RUnlock()
	if !ok {
		return ErrNodeOffline
	}
	select {
	case c.Send <- OutboundMessage{MessageType: websocket.TextMessage, Data: payload}:
		return nil
	default:
		return ErrNodeOffline
	}
}

func (h *Hub) SendBinary(nodeID string, payload []byte) error {
	h.mu.RLock()
	c, ok := h.agents[nodeID]
	h.mu.RUnlock()
	if !ok {
		return ErrNodeOffline
	}
	select {
	case c.Send <- OutboundMessage{MessageType: websocket.BinaryMessage, Data: payload}:
		return nil
	default:
		return ErrNodeOffline
	}
}

func (h *Hub) IsOnline(nodeID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.agents[nodeID]
	return ok
}

const (
	PingInterval = 25 * time.Second
	PongWait     = 70 * time.Second
	writeWait    = 10 * time.Second
)

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

func DecodeEnvelope(data []byte) (map[string]any, error) {
	var m map[string]any
	err := json.Unmarshal(data, &m)
	return m, err
}
