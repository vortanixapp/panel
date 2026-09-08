package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func hostingStatusMeta(status string) (label, color string) {
	switch status {
	case "active":
		return "Активен", "emerald"
	case "pending":
		return "Ожидание", "amber"
	case "suspended":
		return "Приостановлен", "rose"
	case "terminated":
		return "Удалён", "rose"
	case "error":
		return "Ошибка", "rose"
	default:
		return status, "sky"
	}
}

func formatExpiresAt(exp *string) (iso, date string) {
	if exp == nil || *exp == "" {
		return "", ""
	}
	t, err := time.Parse(time.RFC3339Nano, *exp)
	if err != nil {
		t, err = time.Parse("2006-01-02 15:04:05-07", *exp)
	}
	if err != nil {
		return *exp, *exp
	}
	return t.Format(time.RFC3339), t.Format("2006-01-02")
}

func (h *Handler) MyHosting(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT ha.id::text, ha.username, ha.primary_domain, ha.status, ha.expires_at::text,
		       ha.ip_address, hp.name, hp.disk_mb
		FROM core.hosting_accounts ha
		LEFT JOIN core.hosting_plans hp ON hp.id = ha.hosting_plan_id
		WHERE ha.user_id = $1
		ORDER BY ha.created_at DESC
	`, claims.UserID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var id, user, status string
			var primaryDomain, exp, ip, planName *string
			var diskMB *int
			if rows.Scan(&id, &user, &primaryDomain, &status, &exp, &ip, &planName, &diskMB) == nil {
				label, color := hostingStatusMeta(status)
				expISO, expDate := formatExpiresAt(exp)
				domain := user
				if primaryDomain != nil && *primaryDomain != "" {
					domain = *primaryDomain
				}
				item := map[string]any{
					"id":              id,
					"username":        user,
					"domain":          domain,
					"primary_domain":  primaryDomain,
					"status":          status,
					"status_label":    label,
					"status_color":    color,
					"expires_at":      expISO,
					"expires_at_date": expDate,
					"ip_address":      ip,
				}
				if planName != nil && *planName != "" {
					plan := map[string]any{"name": *planName}
					if diskMB != nil {
						plan["disk_mb"] = *diskMB
						if *diskMB >= 1024 {
							plan["disk_gb"] = *diskMB / 1024
						}
					}
					item["hosting_plan"] = plan
				}
				list = append(list, item)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"accounts": list})
}

func (h *Handler) HostingRentForm(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, name, price_monthly, disk_mb, max_domains, max_databases, max_email_accounts,
		       has_ssl, has_ssh, has_cron, has_backup, rental_periods
		FROM core.hosting_plans WHERE tenant_id = $1 AND active = true
		ORDER BY position, name
	`, claims.TenantID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	plans := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var id, name string
			var price float64
			var diskMB, maxDomains, maxDB, maxEmail int
			var hasSSL, hasSSH, hasCron, hasBackup bool
			var periods []byte
			if rows.Scan(&id, &name, &price, &diskMB, &maxDomains, &maxDB, &maxEmail,
				&hasSSL, &hasSSH, &hasCron, &hasBackup, &periods) == nil {
				plan := map[string]any{
					"id": id, "name": name, "price_monthly": price,
					"disk_mb": diskMB, "has_ssl": hasSSL, "has_ssh": hasSSH,
					"has_cron": hasCron, "has_backup": hasBackup,
					"features": map[string]any{
						"sites": maxDomains, "databases": maxDB, "email_domains": maxEmail,
					},
				}
				if diskMB >= 1024 {
					plan["disk_gb"] = diskMB / 1024
				}
				if len(periods) > 0 {
					var rentalPeriods []any
					if json.Unmarshal(periods, &rentalPeriods) == nil {
						plan["rental_periods"] = rentalPeriods
					}
				}
				plans = append(plans, plan)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"plans": plans})
}

func (h *Handler) HostingRentSubmit(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		PlanID   string `json:"plan_id"`
		Username string `json:"username"`
		Domain   string `json:"domain"`
		Period   int    `json:"period"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	username := body.Username
	if username == "" {
		username = body.Domain
	}
	periodDays := 30
	if body.Period > 0 {
		periodDays = body.Period
	}
	var id string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.hosting_accounts (
			tenant_id, user_id, hosting_server_id, hosting_plan_id, username, primary_domain, expires_at
		)
		SELECT $1, $2, p.hosting_server_id, p.id, $4, NULLIF($5, ''), now() + make_interval(days => $6)
		FROM core.hosting_plans p
		WHERE p.id = NULLIF($3, '')::uuid AND p.tenant_id = $1
		RETURNING id::text
	`, claims.TenantID, claims.UserID, body.PlanID, username, body.Domain, periodDays).Scan(&id)
	if err != nil || id == "" {
		writeError(w, http.StatusBadRequest, "invalid plan or failed to create account")
		return
	}
	var hostingServerID, planPackage string
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT hs.id::text, hp.panel_package_name
		FROM core.hosting_accounts ha
		JOIN core.hosting_servers hs ON hs.id = ha.hosting_server_id
		JOIN core.hosting_plans hp ON hp.id = ha.hosting_plan_id
		WHERE ha.id = $1
	`, id).Scan(&hostingServerID, &planPackage)
	h.provisionHostingAccount(r, claims.TenantID, id, username, body.Domain, hostingServerID, planPackage)
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) GetHostingAccount(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	var username, status string
	var primaryDomain, exp, ip, panelURL, created *string
	var planName *string
	var diskMB, maxDomains, maxDB, maxEmail *int
	var hasSSL, hasSSH, hasCron, hasBackup *bool
	var serverPanelType *string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT ha.username, ha.primary_domain, ha.status, ha.expires_at::text, ha.ip_address,
		       ha.panel_login_url, ha.created_at::text, hp.name, hp.disk_mb, hp.max_domains, hp.max_databases,
		       hp.max_email_accounts, hp.has_ssl, hp.has_ssh, hp.has_cron, hp.has_backup,
		       hs.panel_type
		FROM core.hosting_accounts ha
		LEFT JOIN core.hosting_plans hp ON hp.id = ha.hosting_plan_id
		LEFT JOIN core.hosting_servers hs ON hs.id = ha.hosting_server_id
		WHERE ha.id = $1 AND ha.user_id = $2
	`, id, claims.UserID).Scan(
		&username, &primaryDomain, &status, &exp, &ip, &panelURL, &created, &planName,
		&diskMB, &maxDomains, &maxDB, &maxEmail, &hasSSL, &hasSSH, &hasCron, &hasBackup,
		&serverPanelType,
	)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	label, color := hostingStatusMeta(status)
	expISO, expDate := formatExpiresAt(exp)
	domain := username
	if primaryDomain != nil && *primaryDomain != "" {
		domain = *primaryDomain
	}

	account := map[string]any{
		"id":              id,
		"username":        username,
		"domain":          domain,
		"primary_domain":  primaryDomain,
		"status":          status,
		"status_label":    label,
		"status_color":    color,
		"expires_at":      expISO,
		"expires_at_date": expDate,
		"ip_address":      ip,
		"panel_login_url": panelURL,
	}
	if created != nil && *created != "" {
		account["created_at"] = *created
	}
	if serverPanelType != nil && *serverPanelType != "" {
		pt := *serverPanelType
		ptLabel := pt
		switch pt {
		case "cpanel":
			ptLabel = "cPanel"
		case "ispmanager":
			ptLabel = "ISPmanager"
		case "fastpanel":
			ptLabel = "FastPanel"
		}
		account["hosting_server"] = map[string]any{"panel_type_label": ptLabel}
	}

	if planName != nil {
		plan := map[string]any{"name": *planName}
		if diskMB != nil {
			plan["disk_mb"] = *diskMB
			if *diskMB >= 1024 {
				plan["disk_gb"] = *diskMB / 1024
			}
		}
		if maxDomains != nil {
			plan["max_domains"] = *maxDomains
		}
		if maxDB != nil {
			plan["max_databases"] = *maxDB
		}
		if maxEmail != nil {
			plan["max_email_accounts"] = *maxEmail
		}
		if hasSSL != nil {
			plan["has_ssl"] = *hasSSL
		}
		if hasSSH != nil {
			plan["has_ssh"] = *hasSSH
		}
		if hasCron != nil {
			plan["has_cron"] = *hasCron
		}
		if hasBackup != nil {
			plan["has_backup"] = *hasBackup
		}
		account["hosting_plan"] = plan
	}

	account["domains"] = h.listHostingDomains(r, id)
	account["databases"] = h.listHostingDatabases(r, id)
	account["emails"] = h.listHostingEmails(r, id)

	writeJSON(w, http.StatusOK, account)
}

func (h *Handler) listHostingDomains(r *http.Request, accountID string) []map[string]any {
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, domain, status, created_at::text FROM core.hosting_domains
		WHERE hosting_account_id = $1 AND status != 'removed' ORDER BY is_primary DESC, domain
	`, accountID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var id, name, status string
			var created *string
			if rows.Scan(&id, &name, &status, &created) == nil {
				item := map[string]any{"id": id, "name": name, "status": status}
				if created != nil {
					item["created_at"] = *created
				}
				list = append(list, item)
			}
		}
	}
	return list
}

func (h *Handler) listHostingDatabases(r *http.Request, accountID string) []map[string]any {
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, name, db_user, status, created_at::text FROM core.hosting_databases
		WHERE hosting_account_id = $1 AND status != 'removed' ORDER BY name
	`, accountID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var id, name, status string
			var user, created *string
			if rows.Scan(&id, &name, &user, &status, &created) == nil {
				item := map[string]any{"id": id, "name": name, "status": status}
				if user != nil {
					item["user"] = *user
				}
				if created != nil {
					item["created_at"] = *created
				}
				list = append(list, item)
			}
		}
	}
	return list
}

func (h *Handler) listHostingEmails(r *http.Request, accountID string) []map[string]any {
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, address, status, created_at::text FROM core.hosting_emails
		WHERE hosting_account_id = $1 AND status != 'removed' ORDER BY address
	`, accountID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var id, address, status string
			var created *string
			if rows.Scan(&id, &address, &status, &created) == nil {
				item := map[string]any{"id": id, "address": address, "status": status}
				if created != nil {
					item["created_at"] = *created
				}
				list = append(list, item)
			}
		}
	}
	return list
}

func (h *Handler) AdminHostingServers(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `SELECT id::text, name, panel_type, hostname, active FROM core.hosting_servers WHERE tenant_id = $1`, claims.TenantID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var id, name, panel, host string
			var active bool
			if rows.Scan(&id, &name, &panel, &host, &active) == nil {
				list = append(list, map[string]any{"id": id, "name": name, "panel_type": panel, "host": host, "active": active})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"servers": list})
}

func (h *Handler) AdminHostingPlans(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	// Полный набор полей, а не витрина списка: тариф правят той же формой,
	// которой создают, и отдельной ручки за подробностями нет.
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, hosting_server_id::text, name, panel_package_name, price_monthly, active,
		       disk_mb, bandwidth_mb, max_domains, max_databases, max_email_accounts,
		       has_ssl, has_ssh, has_cron, has_backup, COALESCE(php_version, ''),
		       position, COALESCE(description, '')
		FROM core.hosting_plans WHERE tenant_id = $1 ORDER BY position, name
	`, claims.TenantID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var id, serverID, name, pkg, phpVersion, description string
			var price float64
			var active, hasSSL, hasSSH, hasCron, hasBackup bool
			var diskMB, bandwidthMB, maxDomains, maxDatabases, maxEmails, position int
			if rows.Scan(&id, &serverID, &name, &pkg, &price, &active,
				&diskMB, &bandwidthMB, &maxDomains, &maxDatabases, &maxEmails,
				&hasSSL, &hasSSH, &hasCron, &hasBackup, &phpVersion,
				&position, &description) == nil {
				list = append(list, map[string]any{
					"id": id, "hosting_server_id": serverID, "name": name,
					"panel_package_name": pkg, "price_monthly": price, "active": active,
					"disk_mb": diskMB, "bandwidth_mb": bandwidthMB, "max_domains": maxDomains,
					"max_databases": maxDatabases, "max_email_accounts": maxEmails,
					"has_ssl": hasSSL, "has_ssh": hasSSH, "has_cron": hasCron,
					"has_backup": hasBackup, "php_version": phpVersion,
					"position": position, "description": description,
				})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"plans": list})
}

func (h *Handler) AdminHostingAccounts(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT ha.id::text, ha.user_id::text, COALESCE(u.email, ''), ha.username,
		       COALESCE(ha.primary_domain, ''), ha.status,
		       ha.expires_at::text, ha.suspended_at::text,
		       COALESCE(hp.name, ''), COALESCE(hs.name, ''), ha.created_at::text
		FROM core.hosting_accounts ha
		LEFT JOIN core.users u ON u.id = ha.user_id
		LEFT JOIN core.hosting_plans hp ON hp.id = ha.hosting_plan_id
		LEFT JOIN core.hosting_servers hs ON hs.id = ha.hosting_server_id
		WHERE ha.tenant_id = $1
		ORDER BY ha.created_at DESC
	`, claims.TenantID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var id, uid, email, user, domain, status, plan, server, created string
			var expires, suspended *string
			if rows.Scan(&id, &uid, &email, &user, &domain, &status, &expires,
				&suspended, &plan, &server, &created) == nil {
				list = append(list, map[string]any{
					"id": id, "user_id": uid, "user_email": email, "username": user,
					"primary_domain": domain, "status": status,
					"expires_at": expires, "suspended_at": suspended,
					"plan": plan, "server": server, "created_at": created,
				})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"accounts": list})
}
