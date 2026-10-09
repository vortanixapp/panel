package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/pkg/protocol"
)

func (h *Handler) PostAdminAgentSandboxInstall(w http.ResponseWriter, r *http.Request) {
	h.startAgentTaskHTTP(w, r, protocol.ActionSandboxInstall, map[string]any{}, "agent.sandbox_install")
}

func (h *Handler) GetAdminAgentSandbox(w http.ResponseWriter, r *http.Request) {
	nodeID, link, ok := h.agentLinkOr404(w, r)
	if !ok {
		return
	}
	out := h.latestNodeTasks(r.Context(), chi.URLParam(r, "id"), protocol.ActionSandboxInstall)
	out["supported"] = link.can(protocol.CapSandbox)
	out["online"] = link.Online
	out["node_id"] = nodeID
	writeJSON(w, http.StatusOK, out)
}
