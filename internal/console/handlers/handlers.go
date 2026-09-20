package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	"github.com/vortanixapp/panel/internal/console/relayclient"
	"github.com/vortanixapp/panel/pkg/paneljwt"
	"github.com/vortanixapp/panel/pkg/protocol"
)

const (
	consolePongWait   = 70 * time.Second
	consolePingPeriod = 25 * time.Second
	consoleWriteWait  = 10 * time.Second
	consoleReadLimit  = 64 << 10
)

type Handler struct {
	redis *redis.Client
	relay *relayclient.Client
	jwt   *paneljwt.Verifier
	upgr  websocket.Upgrader
}

func New(rdb *redis.Client, relay *relayclient.Client, jwt *paneljwt.Verifier) *Handler {
	return &Handler{
		redis: rdb, relay: relay, jwt: jwt,
		upgr: websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }},
	}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/health", h.Health)
	r.Get("/v1/console", h.ConsoleWS)
	r.Get("/v1/dashboard/stream", h.DashboardWS)
	return r
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ConsoleWS(w http.ResponseWriter, r *http.Request) {
	ticket := r.URL.Query().Get("ticket")
	if ticket == "" {
		writeError(w, http.StatusBadRequest, "ticket required")
		return
	}
	ctx := r.Context()
	var data map[string]any
	ok, err := h.getJSON(ctx, "console:ticket:"+ticket, &data)
	if err != nil || !ok {
		writeError(w, http.StatusUnauthorized, "invalid or expired ticket")
		return
	}
	sessionID, _ := data["session_id"].(string)
	serverID, _ := data["server_id"].(string)
	nodeID, _ := data["node_id"].(string)
	canCommand, _ := data["can_command"].(bool)
	_ = h.redis.Del(ctx, "console:ticket:"+ticket)

	conn, err := h.upgr.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	pubsub := h.redis.Subscribe(ctx, protocol.ConsoleChannel(sessionID))
	defer pubsub.Close()

	conn.SetReadLimit(consoleReadLimit)
	_ = conn.SetReadDeadline(time.Now().Add(consolePongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(consolePongWait))
	})

	var writeMu sync.Mutex
	write := func(kind int, payload []byte) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(consoleWriteWait))
		return conn.WriteMessage(kind, payload)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer conn.Close()
		ticker := time.NewTicker(consolePingPeriod)
		defer ticker.Stop()
		messages := pubsub.Channel()
		for {
			select {
			case msg, ok := <-messages:
				if !ok {
					return
				}
				if write(websocket.TextMessage, []byte(msg.Payload)) != nil {
					return
				}
			case <-ticker.C:
				if write(websocket.PingMessage, nil) != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var input struct {
			Type string `json:"type"`
			Data string `json:"data"`
		}
		if json.Unmarshal(raw, &input) != nil || input.Type != "input" {
			continue
		}
		if !canCommand {
			_ = write(websocket.TextMessage,
				[]byte(`{"type":"error","data":"нет права вводить команды на этом сервере"}`))
			continue
		}
		_ = h.relay.ConsoleInput(ctx, relayclient.ConsoleInputRequest{
			SessionID: sessionID, ServerID: serverID, NodeID: nodeID, Data: input.Data,
		})
	}
	<-done
}

func (h *Handler) DashboardWS(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if token == "" {
		if c, err := r.Cookie("vtx_access"); err == nil && c != nil {
			token = strings.TrimSpace(c.Value)
		}
	}
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing token")
		return
	}
	claims, err := h.jwt.ParseAccess(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	switch claims.Role {
	case "owner", "admin", "support":
	default:
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	conn, err := h.upgr.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	ctx := r.Context()
	pubsub := h.redis.Subscribe(ctx, protocol.TenantEventsChannel())
	defer pubsub.Close()

	conn.SetReadLimit(consoleReadLimit)
	_ = conn.SetReadDeadline(time.Now().Add(consolePongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(consolePongWait))
	})
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				_ = conn.Close()
				return
			}
		}
	}()

	ticker := time.NewTicker(consolePingPeriod)
	defer ticker.Stop()
	messages := pubsub.Channel()
	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(consoleWriteWait))
			if conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)) != nil {
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(consoleWriteWait))
			if conn.WriteMessage(websocket.PingMessage, nil) != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *Handler) getJSON(ctx context.Context, key string, dest any) (bool, error) {
	val, err := h.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal([]byte(val), dest)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
