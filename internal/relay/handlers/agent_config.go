package handlers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/vortanixapp/panel/internal/relay/hub"
	"github.com/vortanixapp/panel/pkg/protocol"
	"github.com/vortanixapp/panel/pkg/settingsreg"
)

const agentConfigInterval = 15 * time.Second

func agentConfigFor(c *hub.AgentConn) map[string]any {
	if !c.HasCap(protocol.CapAgentConfig) {
		return nil
	}
	return settingsreg.AgentValues()
}

func (h *Handler) RunAgentConfigPusher(ctx context.Context) {
	ticker := time.NewTicker(agentConfigInterval)
	defer ticker.Stop()
	raw, _ := json.Marshal(settingsreg.AgentValues())
	last := string(raw)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		values := settingsreg.AgentValues()
		raw, err := json.Marshal(values)
		if err != nil || string(raw) == last {
			continue
		}
		last = string(raw)
		payload, err := json.Marshal(protocol.AgentConfigMessage{Type: protocol.MsgAgentConfig, Values: values})
		if err != nil {
			continue
		}
		sent := 0
		for _, nodeID := range h.hub.Connected() {
			conn := h.hub.Get(nodeID)
			if conn == nil || !conn.HasCap(protocol.CapAgentConfig) {
				continue
			}
			if err := h.hub.SendCommand(nodeID, payload); err == nil {
				sent++
			}
		}
		if sent > 0 {
			log.Printf("relay: настройки агента разосланы узлам: %d", sent)
		}
	}
}
