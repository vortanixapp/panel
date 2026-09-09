package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateHostingServer(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Hostname    string `json:"hostname"`
		IPAddress   string `json:"ip_address"`
		Port        int    `json:"port"`
		PanelType   string `json:"panel_type"`
		APIURL      string `json:"api_url"`
		APIUsername string `json:"api_username"`
		APIToken    string `json:"api_token"`
		UseSSL      bool   `json:"use_ssl"`
		MaxAccounts int    `json:"max_accounts"`
		Active      bool   `json:"active"`
		Description string `json:"description"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Name == "" || body.Hostname == "" || body.PanelType == "" {
		writeError(w, http.StatusBadRequest, "name, hostname and panel_type are required")
		return
	}
	if body.Port == 0 {
		body.Port = 443
	}
	apiToken, err := h.secrets.Encrypt(body.APIToken)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encrypt api token")
		return
	}

	if body.ID != "" {
		if body.APIToken != "" {
			_, err := h.dbOf(r.Context()).Exec(r.Context(), `
				UPDATE core.hosting_servers SET
					name=$2, hostname=$3, ip_address=$4, port=$5, panel_type=$6, api_url=$7,
					api_username=$8, api_token_enc=$9, use_ssl=$10, max_accounts=$11, active=$12,
					description=$13, updated_at=now()
				WHERE id=$1::uuid
			`, body.ID, body.Name, body.Hostname, body.IPAddress, body.Port, body.PanelType,
				body.APIURL, body.APIUsername, apiToken, body.UseSSL, body.MaxAccounts, body.Active, body.Description)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to update hosting server")
				return
			}
		} else {
			_, err := h.dbOf(r.Context()).Exec(r.Context(), `
				UPDATE core.hosting_servers SET
					name=$2, hostname=$3, ip_address=$4, port=$5, panel_type=$6, api_url=$7,
					api_username=$8, use_ssl=$9, max_accounts=$10, active=$11, description=$12, updated_at=now()
				WHERE id=$1::uuid
			`, body.ID, body.Name, body.Hostname, body.IPAddress, body.Port, body.PanelType,
				body.APIURL, body.APIUsername, body.UseSSL, body.MaxAccounts, body.Active, body.Description)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to update hosting server")
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]string{"id": body.ID, "status": "updated"})
		return
	}

	var id string
	err = h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.hosting_servers ( name, hostname, ip_address, port, panel_type, api_url,
			api_username, api_token_enc, use_ssl, max_accounts, active, description
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING id::text
	`, body.Name, body.Hostname, body.IPAddress, body.Port, body.PanelType, body.APIURL,
		body.APIUsername, apiToken, body.UseSSL, body.MaxAccounts, body.Active, body.Description).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create hosting server")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) GetHostingServer(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	var name, hostname, ipAddress, panelType, apiURL, apiUsername, description string
	var port, maxAccounts, currentAccounts int
	var useSSL, active, hasToken bool
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT name, hostname, COALESCE(ip_address,''), port, panel_type, api_url,
		       COALESCE(api_username,''), use_ssl, max_accounts, current_accounts, active,
		       COALESCE(description,''), (COALESCE(api_token_enc,'') <> '')
		FROM core.hosting_servers WHERE id = $1::uuid
	`, id).Scan(&name, &hostname, &ipAddress, &port, &panelType, &apiURL,
		&apiUsername, &useSSL, &maxAccounts, &currentAccounts, &active, &description, &hasToken)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": id, "name": name, "hostname": hostname, "ip_address": ipAddress, "port": port,
		"panel_type": panelType, "api_url": apiURL, "api_username": apiUsername, "use_ssl": useSSL,
		"max_accounts": maxAccounts, "current_accounts": currentAccounts, "active": active,
		"description": description, "has_api_token": hasToken,
	})
}

func (h *Handler) DeleteHostingServer(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	var accounts int
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT count(*) FROM core.hosting_accounts WHERE hosting_server_id = $1::uuid
	`, id).Scan(&accounts); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось проверить аккаунты сервера")
		return
	}
	if accounts > 0 {
		writeError(w, http.StatusConflict,
			fmt.Sprintf("на сервере %d аккаунт(ов): перенесите или удалите их, прежде чем удалять сервер", accounts))
		return
	}

	_, err := h.dbOf(ctx).Exec(ctx, `DELETE FROM core.hosting_servers WHERE id = $1::uuid`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось удалить сервер")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) CreateHostingPlan(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		ID               string  `json:"id"`
		HostingServerID  string  `json:"hosting_server_id"`
		Name             string  `json:"name"`
		PanelPackageName string  `json:"panel_package_name"`
		DiskMB           int     `json:"disk_mb"`
		BandwidthMB      int     `json:"bandwidth_mb"`
		MaxDomains       int     `json:"max_domains"`
		MaxDatabases     int     `json:"max_databases"`
		MaxEmailAccounts int     `json:"max_email_accounts"`
		HasSSL           bool    `json:"has_ssl"`
		HasSSH           bool    `json:"has_ssh"`
		HasCron          bool    `json:"has_cron"`
		HasBackup        bool    `json:"has_backup"`
		PHPVersion       string  `json:"php_version"`
		PriceMonthly     float64 `json:"price_monthly"`
		Position         int     `json:"position"`
		Active           bool    `json:"active"`
		Description      string  `json:"description"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Name == "" || body.PanelPackageName == "" {
		writeError(w, http.StatusBadRequest, "name and panel_package_name are required")
		return
	}

	if body.ID != "" {
		_, err := h.dbOf(r.Context()).Exec(r.Context(), `
			UPDATE core.hosting_plans SET
				name=$2, panel_package_name=$3, disk_mb=$4, bandwidth_mb=$5, max_domains=$6,
				max_databases=$7, max_email_accounts=$8, has_ssl=$9, has_ssh=$10, has_cron=$11,
				has_backup=$12, php_version=NULLIF($13,''), price_monthly=$14, position=$15,
				active=$16, description=$17, updated_at=now()
			WHERE id=$1::uuid
		`, body.ID, body.Name, body.PanelPackageName, body.DiskMB, body.BandwidthMB,
			body.MaxDomains, body.MaxDatabases, body.MaxEmailAccounts, body.HasSSL, body.HasSSH,
			body.HasCron, body.HasBackup, body.PHPVersion, body.PriceMonthly, body.Position,
			body.Active, body.Description)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update hosting plan")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"id": body.ID, "status": "updated"})
		return
	}

	if body.HostingServerID == "" {
		writeError(w, http.StatusBadRequest, "hosting_server_id is required")
		return
	}
	var id string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.hosting_plans ( hosting_server_id, name, panel_package_name, disk_mb, bandwidth_mb,
			max_domains, max_databases, max_email_accounts, has_ssl, has_ssh, has_cron, has_backup,
			php_version, price_monthly, position, active, description
		) VALUES ($1::uuid,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NULLIF($13,''),$14,$15,$16,$17)
		RETURNING id::text
	`, body.HostingServerID, body.Name, body.PanelPackageName, body.DiskMB, body.BandwidthMB,
		body.MaxDomains, body.MaxDatabases, body.MaxEmailAccounts, body.HasSSL, body.HasSSH, body.HasCron,
		body.HasBackup, body.PHPVersion, body.PriceMonthly, body.Position, body.Active, body.Description).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create hosting plan (check hosting_server_id)")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) DeleteHostingPlan(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	_, err := h.dbOf(r.Context()).Exec(r.Context(), `DELETE FROM core.hosting_plans WHERE id = $1::uuid`, id)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to delete (plan may still be in use)")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
