package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vortanixapp/panel/internal/api/jobwake"
)

var ftpSuffixRe = regexp.MustCompile(`^[a-z0-9_-]{0,12}$`)

const defaultFTPAccountLimit = 1

// ftpUsernameIDLen — сколько символов id сервера попадает в имя доступа.
//
// Было восемь, то есть 32 бита: при совпадении у двух серверов useradd на ноде
// молча пропускается по проверке `id -u`, и дальше usermod с перемонтированием
// уводят уже существующего пользователя на чужой каталог — прежний клиент
// теряет доступ, а новый получает его файлы. Двенадцать символов дают столько
// же, сколько у группы сервера, и оставляют имя в прежнем потолке длины:
// 3 + 12 + 1 + 12 = 28 символов, как и раньше при суффиксе в 16.
const ftpUsernameIDLen = 12

type ftpAccountRow struct {
	ID       string  `json:"id"`
	Username string  `json:"username"`
	Password string  `json:"password"`
	Status   string  `json:"status"`
	Error    *string `json:"error_message,omitempty"`
}

func ftpUsernameFor(serverID, suffix string) string {
	short := strings.ReplaceAll(serverID, "-", "")
	if len(short) > ftpUsernameIDLen {
		short = short[:ftpUsernameIDLen]
	}
	base := "vtx" + short
	if suffix == "" {
		return base
	}
	return base + "_" + suffix
}

func (h *Handler) ServerFtpList(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	if !h.authorizeServerAction(w, r, claims, serverID, "files_list") {
		return
	}

	accounts := h.loadFtpAccounts(r, claims.TenantID, serverID)
	host, port := h.ftpEndpoint(r, claims.TenantID, serverID)
	writeJSON(w, http.StatusOK, map[string]any{
		"accounts": accounts,
		"host":     host,
		"port":     port,
		"protocol": "sftp",
		"limit":    h.ftpAccountLimit(r, claims.TenantID, serverID),
		"root":     "/data",
	})
}

func (h *Handler) ServerFtpCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	if !h.authorizeServerAction(w, r, claims, serverID, "files_write") {
		return
	}

	var body struct {
		Suffix string `json:"suffix"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	suffix := strings.ToLower(strings.TrimSpace(body.Suffix))
	if !ftpSuffixRe.MatchString(suffix) {
		writeError(w, http.StatusUnprocessableEntity, "суффикс: до 12 символов [a-z0-9_-]")
		return
	}

	existing := h.loadFtpAccounts(r, claims.TenantID, serverID)
	limit := h.ftpAccountLimit(r, claims.TenantID, serverID)
	if len(existing) >= limit {
		writeError(w, http.StatusUnprocessableEntity,
			fmt.Sprintf("тариф допускает не более %d FTP-аккаунтов", limit))
		return
	}

	username := ftpUsernameFor(serverID, suffix)
	password := randomPassword(16)

	var accountID string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.server_ftp_accounts (tenant_id, server_id, username, password, status)
		VALUES ($1, $2, $3, $4, 'pending')
		RETURNING id::text
	`, claims.TenantID, serverID, username, password).Scan(&accountID)
	if err != nil {
		writeError(w, http.StatusConflict, "такой FTP-аккаунт уже существует")
		return
	}

	h.enqueueFtpJob(r, claims.TenantID, map[string]any{
		"account_id": accountID,
		"server_id":  serverID,
		"action":     "create",
		"username":   username,
		"password":   password,
	})
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "server.ftp_create", "server:"+serverID,
		map[string]any{"username": username})
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":   "pending",
		"username": username,
		"password": password,
	})
}

func (h *Handler) ServerFtpResetPassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	if !h.authorizeServerAction(w, r, claims, serverID, "files_write") {
		return
	}

	var body struct {
		Username string `json:"username"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	accountID, username, err := h.ftpAccountByName(r, claims.TenantID, serverID, body.Username)
	if err != nil {
		writeError(w, http.StatusNotFound, "FTP-аккаунт не найден")
		return
	}

	password := randomPassword(16)
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.server_ftp_accounts
		SET password = $2, status = 'pending', error_message = NULL, updated_at = now()
		WHERE id = $1
	`, accountID, password)

	h.enqueueFtpJob(r, claims.TenantID, map[string]any{
		"account_id": accountID,
		"server_id":  serverID,
		"action":     "password",
		"username":   username,
		"password":   password,
	})
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "server.ftp_password", "server:"+serverID,
		map[string]any{"username": username})
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":   "pending",
		"username": username,
		"password": password,
	})
}

func (h *Handler) ServerFtpDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	if !h.authorizeServerAction(w, r, claims, serverID, "files_write") {
		return
	}

	var body struct {
		Username string `json:"username"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	accountID, username, err := h.ftpAccountByName(r, claims.TenantID, serverID, body.Username)
	if err != nil {
		writeError(w, http.StatusNotFound, "FTP-аккаунт не найден")
		return
	}

	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.server_ftp_accounts SET status = 'deleting', updated_at = now() WHERE id = $1
	`, accountID)
	h.enqueueFtpJob(r, claims.TenantID, map[string]any{
		"account_id": accountID,
		"server_id":  serverID,
		"action":     "delete",
		"username":   username,
	})
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "server.ftp_delete", "server:"+serverID,
		map[string]any{"username": username})
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "deleting", "username": username})
}

func (h *Handler) loadFtpAccounts(r *http.Request, tenantID, serverID string) []ftpAccountRow {
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, username, password, status, error_message
		FROM core.server_ftp_accounts
		WHERE tenant_id = $1 AND server_id = $2
		ORDER BY created_at
	`, tenantID, serverID)
	if err != nil {
		return []ftpAccountRow{}
	}
	defer rows.Close()
	out := []ftpAccountRow{}
	for rows.Next() {
		var a ftpAccountRow
		if rows.Scan(&a.ID, &a.Username, &a.Password, &a.Status, &a.Error) == nil {
			out = append(out, a)
		}
	}
	return out
}

func (h *Handler) ftpAccountByName(r *http.Request, tenantID, serverID, username string) (string, string, error) {
	var id, name string
	query := `
		SELECT id::text, username FROM core.server_ftp_accounts
		WHERE tenant_id = $1 AND server_id = $2 AND ($3 = '' OR username = $3)
		ORDER BY created_at LIMIT 1
	`
	err := h.dbOf(r.Context()).QueryRow(r.Context(), query, tenantID, serverID, strings.TrimSpace(username)).Scan(&id, &name)
	if err != nil {
		return "", "", err
	}
	return id, name, nil
}

// sftpPort — порт файлового доступа на ноде. Не 22: там админский SSH и
// fail2ban, и подборы пароля клиентами закрывали бы доступ нам самим.
const sftpPort = 2222

func (h *Handler) ftpEndpoint(r *http.Request, tenantID, serverID string) (string, int) {
	var host string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(n.ssh_host, '')
		FROM core.servers s JOIN core.nodes n ON n.id = s.node_id
		WHERE s.id = $1 AND s.tenant_id = $2
	`, serverID, tenantID).Scan(&host)
	if err != nil {
		return "", sftpPort
	}
	return host, sftpPort
}

func (h *Handler) ftpAccountLimit(r *http.Request, tenantID, serverID string) int {
	var limits []byte
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(t.meta, '{}'::jsonb)
		FROM core.servers s
		LEFT JOIN core.tariffs t ON t.id = s.tariff_id
		WHERE s.id = $1 AND s.tenant_id = $2
	`, serverID, tenantID).Scan(&limits)
	if err != nil {
		return defaultFTPAccountLimit
	}
	m := map[string]any{}
	_ = json.Unmarshal(limits, &m)
	switch v := m["ftp_accounts"].(type) {
	case float64:
		if int(v) > 0 {
			return int(v)
		}
	case int:
		if v > 0 {
			return v
		}
	}
	return defaultFTPAccountLimit
}

// enqueueFtpCleanup снимает доступы удаляемого сервера с ноды.
//
// Строки аккаунтов уедут каскадом вместе с сервером, но на самой ноде
// останутся пользователь Linux, смонтированный каталог и строка в fstab —
// поэтому задания ставим до удаления и кладём в них node_id: искать ноду через
// core.servers будет уже негде.
func (h *Handler) enqueueFtpCleanup(r *http.Request, tenantID, serverID, nodeID string) {
	for _, a := range h.loadFtpAccounts(r, tenantID, serverID) {
		h.enqueueFtpJob(r, tenantID, map[string]any{
			"account_id": a.ID,
			"server_id":  serverID,
			"node_id":    nodeID,
			"action":     "delete",
			"username":   a.Username,
		})
	}
}

func (h *Handler) enqueueFtpJob(r *http.Request, tenantID string, payload map[string]any) {
	raw, _ := json.Marshal(payload)
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.jobs (tenant_id, type, status, payload)
		VALUES ($1, 'ftp_account', 'pending', $2::jsonb)
	`, tenantID, raw)
	jobwake.Notify("ftp_account")
}
