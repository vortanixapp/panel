package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vortanixapp/panel/internal/api/hosting"
)

func (h *Handler) ListHostingDomains(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	if !h.ownsHostingAccount(r, id, claims.UserID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"domains": h.listHostingDomains(r, id)})
}

func (h *Handler) CreateHostingDomain(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	accountID := chi.URLParam(r, "id")
	panelID, panelCfg, err := h.hostingPanelRef(r, accountID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var body struct {
		Domain string `json:"domain"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Domain == "" {
		writeError(w, http.StatusBadRequest, "domain required")
		return
	}
	adapter := hosting.NewAdapter(panelCfg)
	if err := adapter.AddDomain(r.Context(), panelID, body.Domain); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var id string
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.hosting_domains (tenant_id, hosting_account_id, domain, status)
		SELECT tenant_id, $1::uuid, $2, 'active' FROM core.hosting_accounts WHERE id = $1::uuid
		RETURNING id::text
	`, accountID, body.Domain).Scan(&id)
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) ListHostingDatabases(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	if !h.ownsHostingAccount(r, id, claims.UserID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"databases": h.listHostingDatabases(r, id)})
}

func (h *Handler) CreateHostingDatabase(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	accountID := chi.URLParam(r, "id")
	panelID, panelCfg, err := h.hostingPanelRef(r, accountID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var body struct {
		Name string `json:"name"`
		User string `json:"user"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	if body.User == "" {
		body.User = body.Name
	}
	adapter := hosting.NewAdapter(panelCfg)
	if err := adapter.AddDatabase(r.Context(), panelID, body.Name, body.User); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var id string
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.hosting_databases (tenant_id, hosting_account_id, name, db_user, status)
		SELECT tenant_id, $1::uuid, $2, $3, 'active' FROM core.hosting_accounts WHERE id = $1::uuid
		RETURNING id::text
	`, accountID, body.Name, body.User).Scan(&id)
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) ListHostingEmails(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	if !h.ownsHostingAccount(r, id, claims.UserID) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"emails": h.listHostingEmails(r, id)})
}

func (h *Handler) CreateHostingEmail(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	accountID := chi.URLParam(r, "id")
	panelID, panelCfg, err := h.hostingPanelRef(r, accountID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var body struct {
		Address  string `json:"address"`
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Address == "" {
		writeError(w, http.StatusBadRequest, "address required")
		return
	}
	adapter := hosting.NewAdapter(panelCfg)
	if err := adapter.AddEmail(r.Context(), panelID, body.Address, body.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var id string
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.hosting_emails (tenant_id, hosting_account_id, address, mailbox_name, status)
		SELECT tenant_id, $1::uuid, $2, split_part($2, '@', 1), 'active' FROM core.hosting_accounts WHERE id = $1::uuid
		RETURNING id::text
	`, accountID, body.Address).Scan(&id)
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) RenewHostingAccount(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	accountID := chi.URLParam(r, "id")
	panelID, panelCfg, err := h.hostingPanelRef(r, accountID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var body struct {
		Period int `json:"period"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Period <= 0 {
		body.Period = 30
	}
	adapter := hosting.NewAdapter(panelCfg)
	if err := adapter.Renew(r.Context(), panelID, body.Period); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var wasSuspended bool
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT suspended_at IS NOT NULL FROM core.hosting_accounts WHERE id = $1
	`, accountID).Scan(&wasSuspended)
	if wasSuspended {
		if err := adapter.Unsuspend(r.Context(), panelID); err != nil {
			writeError(w, http.StatusBadGateway, "не удалось снять приостановку на панели: "+err.Error())
			return
		}
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.hosting_accounts
		SET expires_at = GREATEST(COALESCE(expires_at, now()), now()) + make_interval(days => $2),
		    status = 'active',
		    suspended_at = NULL
		WHERE id = $1
	`, accountID, body.Period)
	writeJSON(w, http.StatusOK, map[string]string{"status": "renewed"})
}

func (h *Handler) ChangeHostingPassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	accountID := chi.URLParam(r, "id")
	panelID, panelCfg, err := h.hostingPanelRef(r, accountID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if len(body.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be 8+ chars")
		return
	}
	adapter := hosting.NewAdapter(panelCfg)
	if err := adapter.ChangePassword(r.Context(), panelID, body.Password); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) ownsHostingAccount(r *http.Request, accountID, userID string) bool {
	var uid string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `SELECT user_id::text FROM core.hosting_accounts WHERE id = $1`, accountID).Scan(&uid)
	return err == nil && uid == userID
}

func (h *Handler) hostingPanelRef(r *http.Request, accountID, userID string) (panelAccountID string, cfg hosting.ServerConfig, err error) {
	err = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(ha.panel_account_id, ha.username), hs.panel_type, hs.api_url,
		       COALESCE(hs.api_username, ''), COALESCE(hs.api_token_enc, '')
		FROM core.hosting_accounts ha
		JOIN core.hosting_servers hs ON hs.id = ha.hosting_server_id
		WHERE ha.id = $1 AND ha.user_id = $2
	`, accountID, userID).Scan(&panelAccountID, &cfg.PanelType, &cfg.APIURL, &cfg.APIUsername, &cfg.APIToken)
	cfg.APIToken = h.secrets.MustDecrypt(cfg.APIToken)
	return
}

func (h *Handler) provisionHostingAccount(r *http.Request, tenantID, accountID, username, domain, hostingServerID, planPackage string) {
	var cfg hosting.ServerConfig
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT panel_type, api_url, COALESCE(api_username, ''), COALESCE(api_token_enc, '')
		FROM core.hosting_servers WHERE id = $1
	`, hostingServerID).Scan(&cfg.PanelType, &cfg.APIURL, &cfg.APIUsername, &cfg.APIToken)
	if err != nil {
		return
	}
	cfg.APIToken = h.secrets.MustDecrypt(cfg.APIToken)
	adapter := hosting.NewAdapter(cfg)
	panelID, loginURL, err := adapter.CreateAccount(r.Context(), username, domain, planPackage)
	if err != nil {
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
			UPDATE core.hosting_accounts SET status = 'error', updated_at = now() WHERE id = $1
		`, accountID)
		return
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.hosting_accounts SET panel_account_id = $2, panel_login_url = $3, status = 'active', updated_at = now()
		WHERE id = $1
	`, accountID, panelID, loginURL)
	_ = tenantID
}
