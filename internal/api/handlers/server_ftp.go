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

	accounts := h.loadFtpAccounts(r, serverID)
	host, port := h.ftpEndpoint(r, serverID)
	writeJSON(w, http.StatusOK, map[string]any{
		"accounts": accounts,
		"host":     host,
		"port":     port,
		"protocol": "sftp",
		"limit":    h.ftpAccountLimit(r, serverID),
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

	existing := h.loadFtpAccounts(r, serverID)
	limit := h.ftpAccountLimit(r, serverID)
	if len(existing) >= limit {
		writeError(w, http.StatusUnprocessableEntity,
			fmt.Sprintf("тариф допускает не более %d FTP-аккаунтов", limit))
		return
	}

	username := ftpUsernameFor(serverID, suffix)
	password := randomPassword(16)

	var accountID string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.server_ftp_accounts ( server_id, username, password, status)
		VALUES ( $1, $2, $3, 'pending')
		RETURNING id::text
	`, serverID, username, password).Scan(&accountID)
	if err != nil {
		writeError(w, http.StatusConflict, "такой FTP-аккаунт уже существует")
		return
	}

	h.enqueueFtpJob(r, map[string]any{
		"account_id": accountID,
		"server_id":  serverID,
		"action":     "create",
		"username":   username,
		"password":   password,
	})
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "server.ftp_create", "server:"+serverID,
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

	accountID, username, err := h.ftpAccountByName(r, serverID, body.Username)
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

	h.enqueueFtpJob(r, map[string]any{
		"account_id": accountID,
		"server_id":  serverID,
		"action":     "password",
		"username":   username,
		"password":   password,
	})
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "server.ftp_password", "server:"+serverID,
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

	accountID, username, err := h.ftpAccountByName(r, serverID, body.Username)
	if err != nil {
		writeError(w, http.StatusNotFound, "FTP-аккаунт не найден")
		return
	}

	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.server_ftp_accounts SET status = 'deleting', updated_at = now() WHERE id = $1
	`, accountID)
	h.enqueueFtpJob(r, map[string]any{
		"account_id": accountID,
		"server_id":  serverID,
		"action":     "delete",
		"username":   username,
	})
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "server.ftp_delete", "server:"+serverID,
		map[string]any{"username": username})
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "deleting", "username": username})
}

func (h *Handler) loadFtpAccounts(r *http.Request, serverID string) []ftpAccountRow {
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, username, password, status, error_message
		FROM core.server_ftp_accounts
		WHERE server_id = $1
		ORDER BY created_at
	`, serverID)
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

func (h *Handler) ftpAccountByName(r *http.Request, serverID, username string) (string, string, error) {
	var id, name string
	query := `
		SELECT id::text, username FROM core.server_ftp_accounts
		WHERE server_id = $1 AND ($2 = '' OR username = $2)
		ORDER BY created_at LIMIT 1
	`
	err := h.dbOf(r.Context()).QueryRow(r.Context(), query, serverID, strings.TrimSpace(username)).Scan(&id, &name)
	if err != nil {
		return "", "", err
	}
	return id, name, nil
}

const sftpPort = 2222

func (h *Handler) ftpEndpoint(r *http.Request, serverID string) (string, int) {
	var host string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(n.ssh_host, '')
		FROM core.servers s JOIN core.nodes n ON n.id = s.node_id
		WHERE s.id = $1
	`, serverID).Scan(&host)
	if err != nil {
		return "", sftpPort
	}
	return host, sftpPort
}

func (h *Handler) ftpAccountLimit(r *http.Request, serverID string) int {
	var limits []byte
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(t.meta, '{}'::jsonb)
		FROM core.servers s
		LEFT JOIN core.tariffs t ON t.id = s.tariff_id
		WHERE s.id = $1
	`, serverID).Scan(&limits)
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

func (h *Handler) enqueueFtpCleanup(r *http.Request, serverID, nodeID string) {
	for _, a := range h.loadFtpAccounts(r, serverID) {
		h.enqueueFtpJob(r, map[string]any{
			"account_id": a.ID,
			"server_id":  serverID,
			"node_id":    nodeID,
			"action":     "delete",
			"username":   a.Username,
		})
	}
}

func (h *Handler) enqueueFtpJob(r *http.Request, payload map[string]any) {
	raw, _ := json.Marshal(payload)
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.jobs ( type, status, payload)
		VALUES ( 'ftp_account', 'pending', $1::jsonb)
	`, raw)
	jobwake.Notify("ftp_account")
}
