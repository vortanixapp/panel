package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/internal/api/hosting"
)

func (h *Handler) adminHostingPanelRef(r *http.Request, accountID string) (string, hosting.ServerConfig, error) {
	var panelID string
	var cfg hosting.ServerConfig
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(ha.panel_account_id, ha.username), hs.panel_type, hs.api_url,
		       COALESCE(hs.api_username, ''), COALESCE(hs.api_token_enc, '')
		FROM core.hosting_accounts ha
		JOIN core.hosting_servers hs ON hs.id = ha.hosting_server_id
		WHERE ha.id = $1
	`, accountID).Scan(&panelID, &cfg.PanelType, &cfg.APIURL, &cfg.APIUsername, &cfg.APIToken)
	if err != nil {
		return "", cfg, err
	}
	cfg.APIToken = h.secrets.MustDecrypt(cfg.APIToken)
	return panelID, cfg, nil
}

func (h *Handler) AdminHostingAccountSuspend(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	accountID := chi.URLParam(r, "id")
	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Reason == "" {
		body.Reason = "suspended by admin"
	}
	panelID, cfg, err := h.adminHostingPanelRef(r, accountID)
	if err != nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}
	if err := hosting.NewAdapter(cfg).Suspend(r.Context(), panelID, body.Reason); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.hosting_accounts
		SET status = 'suspended', suspended_at = now(), updated_at = now()
		WHERE id = $1
	`, accountID)
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "hosting.suspend",
		"hosting_account:"+accountID, map[string]any{"reason": body.Reason})
	writeJSON(w, http.StatusOK, map[string]string{"status": "suspended"})
}

func (h *Handler) AdminHostingAccountUnsuspend(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	accountID := chi.URLParam(r, "id")
	panelID, cfg, err := h.adminHostingPanelRef(r, accountID)
	if err != nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}
	if err := hosting.NewAdapter(cfg).Unsuspend(r.Context(), panelID); err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.hosting_accounts
		SET status = 'active', suspended_at = NULL, updated_at = now()
		WHERE id = $1
	`, accountID)
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "hosting.unsuspend",
		"hosting_account:"+accountID, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "active"})
}
