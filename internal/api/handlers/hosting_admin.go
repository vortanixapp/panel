package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) CreateHostingServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
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
					name=$3, hostname=$4, ip_address=$5, port=$6, panel_type=$7, api_url=$8,
					api_username=$9, api_token_enc=$10, use_ssl=$11, max_accounts=$12, active=$13,
					description=$14, updated_at=now()
				WHERE id=$1::uuid AND tenant_id=$2
			`, body.ID, claims.TenantID, body.Name, body.Hostname, body.IPAddress, body.Port, body.PanelType,
				body.APIURL, body.APIUsername, apiToken, body.UseSSL, body.MaxAccounts, body.Active, body.Description)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "failed to update hosting server")
				return
			}
		} else {
			_, err := h.dbOf(r.Context()).Exec(r.Context(), `
				UPDATE core.hosting_servers SET
					name=$3, hostname=$4, ip_address=$5, port=$6, panel_type=$7, api_url=$8,
					api_username=$9, use_ssl=$10, max_accounts=$11, active=$12, description=$13, updated_at=now()
				WHERE id=$1::uuid AND tenant_id=$2
			`, body.ID, claims.TenantID, body.Name, body.Hostname, body.IPAddress, body.Port, body.PanelType,
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
		INSERT INTO core.hosting_servers (
			tenant_id, name, hostname, ip_address, port, panel_type, api_url,
			api_username, api_token_enc, use_ssl, max_accounts, active, description
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING id::text
	`, claims.TenantID, body.Name, body.Hostname, body.IPAddress, body.Port, body.PanelType, body.APIURL,
		body.APIUsername, apiToken, body.UseSSL, body.MaxAccounts, body.Active, body.Description).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create hosting server")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) GetHostingServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
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
		FROM core.hosting_servers WHERE id = $1::uuid AND tenant_id = $2
	`, id, claims.TenantID).Scan(&name, &hostname, &ipAddress, &port, &panelType, &apiURL,
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
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	// Текст ошибки обещал отказ при живых аккаунтах, но внешний ключ стоит с
	// каскадом: удаление молча сносило аккаунты клиентов вместе с их сайтами.
	var accounts int
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT count(*) FROM core.hosting_accounts WHERE hosting_server_id = $1::uuid AND tenant_id = $2
	`, id, claims.TenantID).Scan(&accounts); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось проверить аккаунты сервера")
		return
	}
	if accounts > 0 {
		writeError(w, http.StatusConflict,
			fmt.Sprintf("на сервере %d аккаунт(ов): перенесите или удалите их, прежде чем удалять сервер", accounts))
		return
	}

	_, err := h.dbOf(ctx).Exec(ctx, `DELETE FROM core.hosting_servers WHERE id = $1::uuid AND tenant_id = $2`, id, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось удалить сервер")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) CreateHostingPlan(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
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
				name=$3, panel_package_name=$4, disk_mb=$5, bandwidth_mb=$6, max_domains=$7,
				max_databases=$8, max_email_accounts=$9, has_ssl=$10, has_ssh=$11, has_cron=$12,
				has_backup=$13, php_version=NULLIF($14,''), price_monthly=$15, position=$16,
				active=$17, description=$18, updated_at=now()
			WHERE id=$1::uuid AND tenant_id=$2
		`, body.ID, claims.TenantID, body.Name, body.PanelPackageName, body.DiskMB, body.BandwidthMB,
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
		INSERT INTO core.hosting_plans (
			tenant_id, hosting_server_id, name, panel_package_name, disk_mb, bandwidth_mb,
			max_domains, max_databases, max_email_accounts, has_ssl, has_ssh, has_cron, has_backup,
			php_version, price_monthly, position, active, description
		) VALUES ($1,$2::uuid,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NULLIF($14,''),$15,$16,$17,$18)
		RETURNING id::text
	`, claims.TenantID, body.HostingServerID, body.Name, body.PanelPackageName, body.DiskMB, body.BandwidthMB,
		body.MaxDomains, body.MaxDatabases, body.MaxEmailAccounts, body.HasSSL, body.HasSSH, body.HasCron,
		body.HasBackup, body.PHPVersion, body.PriceMonthly, body.Position, body.Active, body.Description).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create hosting plan (check hosting_server_id)")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) DeleteHostingPlan(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	_, err := h.dbOf(r.Context()).Exec(r.Context(), `DELETE FROM core.hosting_plans WHERE id = $1::uuid AND tenant_id = $2`, id, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to delete (plan may still be in use)")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
