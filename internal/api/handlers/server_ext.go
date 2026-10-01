package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/pkg/backupname"
	"github.com/vortanixapp/panel/pkg/cronexpr"
)

const serverNameMax = 100

func (h *Handler) PatchServer(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	claims, ok := h.authorizeServerTab(w, r, serverID, "settings_write")
	if !ok {
		return
	}
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)

	var cfg, lim []byte
	fields := []string{}
	if v, has := body["name"]; has && v != nil {
		name, isText := v.(string)
		name = strings.TrimSpace(name)
		if !isText || name == "" || utf8.RuneCountInString(name) > serverNameMax {
			writeError(w, http.StatusUnprocessableEntity, "Название сервера — от 1 до 100 символов")
			return
		}
		body["name"] = name
		fields = append(fields, "name")
	}
	if v, has := body["config"]; has {
		if !isStaffRole(claims.Role) {
			writeError(w, http.StatusForbidden, "Изменение конфигурации доступно только сотрудникам")
			return
		}
		cfg, _ = json.Marshal(v)
		fields = append(fields, "config")
	}
	if v, has := body["limits"]; has {
		if !isStaffRole(claims.Role) {
			writeError(w, http.StatusForbidden, "Изменение лимитов доступно только сотрудникам")
			return
		}
		lim, _ = json.Marshal(v)
		fields = append(fields, "limits")
	}

	ctx := r.Context()
	_, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.servers SET config = COALESCE($2::jsonb, config), limits = COALESCE($3::jsonb, limits), name = COALESCE($4, name)
		WHERE id = $1
	`, serverID, cfg, lim, body["name"])
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	if v, has := body["comment"]; has {
		comment, _ := v.(string)
		_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.servers SET comment = $2 WHERE id = $1`,
			serverID, projectComment(comment))
		fields = append(fields, "comment")
	}
	if v, has := body["delete_protection"]; has {
		_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.servers SET delete_protection = $2 WHERE id = $1`,
			serverID, truthy(v))
		fields = append(fields, "delete_protection")
	}
	if v, has := body["project_id"]; has {
		h.moveServerToProject(ctx, serverID, claims.UserID, strings.TrimSpace(anyString(v)))
		fields = append(fields, "project_id")
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "server.patch", "server:"+serverID,
		map[string]any{"fields": fields})
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) GetServerLogs(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	tail := 200
	if q := r.URL.Query().Get("tail"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			tail = n
		}
	}
	result, ok := h.agentCommandForServer(w, r, serverID, "logs", map[string]any{"tail": tail})
	if !ok {
		return
	}
	lines := []string{}
	if raw, exists := result["lines"]; exists {
		if arr, ok := raw.([]any); ok {
			for _, item := range arr {
				lines = append(lines, toString(item))
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"lines": lines})
}

func (h *Handler) ServerFilesList(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	var body struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	result, ok := h.agentCommandForServer(w, r, serverID, "files_list", map[string]any{"path": body.Path})
	if !ok {
		return
	}
	files := result["files"]
	if files == nil {
		files = []any{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": files, "path": body.Path})
}

func (h *Handler) ServerFilesRead(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	var body struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	result, ok := h.agentCommandForServer(w, r, serverID, "files_read", map[string]any{"path": body.Path})
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":    body.Path,
		"content": toString(result["content"]),
	})
}

func (h *Handler) ServerFilesWrite(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	var body struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	_, ok := h.agentCommandForServer(w, r, serverID, "files_write", map[string]any{
		"path":    body.Path,
		"content": body.Content,
	})
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func toString(v any) string {
	switch s := v.(type) {
	case string:
		return s
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func (h *Handler) ServerCronList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, id, "cron_list"); !ok {
		return
	}
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `SELECT id::text, schedule, command, enabled FROM core.server_cron_jobs WHERE server_id = $1`, id)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	tz := h.serverCronTimezone(r.Context(), id)
	loc, _ := time.LoadLocation(tz)
	now := time.Now()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var cid, sched, cmd string
			var en bool
			if rows.Scan(&cid, &sched, &cmd, &en) == nil {
				item := map[string]any{"id": cid, "schedule": sched, "command": cmd, "enabled": en}
				if expr, err := cronexpr.Parse(sched); err != nil {
					item["invalid"] = true
				} else if en {
					if next := expr.Next(now, loc); !next.IsZero() {
						item["next_run"] = next.UTC().Format(time.RFC3339)
					}
				}
				list = append(list, item)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": list, "timezone": tz})
}

func (h *Handler) ServerCronCreate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, id, "cron_create"); !ok {
		return
	}
	var body struct {
		Schedule string `json:"schedule"`
		Command  string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Schedule == "" || body.Command == "" {
		writeError(w, http.StatusBadRequest, "schedule and command required")
		return
	}
	body.Schedule = strings.Join(strings.Fields(body.Schedule), " ")
	body.Command = strings.TrimSpace(body.Command)
	if _, err := cronexpr.Parse(body.Schedule); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Расписание: "+err.Error())
		return
	}
	if len(body.Command) > 2000 || strings.ContainsAny(body.Command, "\r\n") {
		writeError(w, http.StatusUnprocessableEntity, "Команда должна быть одной строкой не длиннее 2000 символов")
		return
	}
	var cid string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.server_cron_jobs (server_id, schedule, command) VALUES ($1, $2, $3) RETURNING id::text
	`, id, body.Schedule, body.Command).Scan(&cid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	if !h.syncCronForServerHTTP(w, r, id) {
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `DELETE FROM core.server_cron_jobs WHERE id = $1 AND server_id = $2`, cid, id)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": cid})
}

func (h *Handler) ServerCronDelete(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, serverID, "cron_delete"); !ok {
		return
	}
	var body struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	var sched, cmd string
	var enabled bool
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT schedule, command, enabled FROM core.server_cron_jobs WHERE id = $1 AND server_id = $2
	`, body.ID, serverID).Scan(&sched, &cmd, &enabled)
	if err != nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	tag, _ := h.dbOf(r.Context()).Exec(r.Context(), `DELETE FROM core.server_cron_jobs WHERE id = $1 AND server_id = $2`, body.ID, serverID)
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	if !h.syncCronForServerHTTP(w, r, serverID) {
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
			INSERT INTO core.server_cron_jobs (id, server_id, schedule, command, enabled)
			VALUES ($1, $2, $3, $4, $5)
		`, body.ID, serverID, sched, cmd, enabled)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ServerCronToggle(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, serverID, "cron_toggle"); !ok {
		return
	}
	var body struct {
		ID      string `json:"id"`
		Enabled bool   `json:"enabled"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	var prevEnabled bool
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT enabled FROM core.server_cron_jobs WHERE id = $1 AND server_id = $2
	`, body.ID, serverID).Scan(&prevEnabled)
	if err != nil {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	tag, _ := h.dbOf(r.Context()).Exec(r.Context(), `UPDATE core.server_cron_jobs SET enabled = $3 WHERE id = $1 AND server_id = $2`, body.ID, serverID, body.Enabled)
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	if !h.syncCronForServerHTTP(w, r, serverID) {
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `UPDATE core.server_cron_jobs SET enabled = $3 WHERE id = $1 AND server_id = $2`, body.ID, serverID, prevEnabled)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) ServerFirewallList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, id, "firewall_list"); !ok {
		return
	}
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `SELECT id::text, protocol, port_from, enabled FROM core.server_firewall_rules WHERE server_id = $1`, id)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var rid, proto string
			var port int
			var en bool
			if rows.Scan(&rid, &proto, &port, &en) == nil {
				list = append(list, map[string]any{"id": rid, "protocol": proto, "port_from": port, "enabled": en})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": list})
}

func (h *Handler) ServerFirewallCreate(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, serverID, "firewall_create"); !ok {
		return
	}
	var body struct {
		Protocol string `json:"protocol"`
		PortFrom int    `json:"port_from"`
		PortTo   *int   `json:"port_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.PortFrom < 1 {
		writeError(w, http.StatusBadRequest, "invalid rule")
		return
	}
	if body.Protocol == "" {
		body.Protocol = "tcp"
	}
	var rid string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.server_firewall_rules (server_id, protocol, port_from, port_to)
		VALUES ($1, $2, $3, $4) RETURNING id::text
	`, serverID, body.Protocol, body.PortFrom, body.PortTo).Scan(&rid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	if !h.syncFirewallForServerHTTP(w, r, serverID) {
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `DELETE FROM core.server_firewall_rules WHERE id = $1 AND server_id = $2`, rid, serverID)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": rid})
}

func (h *Handler) ServerFirewallDelete(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, serverID, "firewall_delete"); !ok {
		return
	}
	var body struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	var proto string
	var portFrom int
	var portTo *int
	var enabled bool
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT protocol, port_from, port_to, enabled FROM core.server_firewall_rules WHERE id = $1 AND server_id = $2
	`, body.ID, serverID).Scan(&proto, &portFrom, &portTo, &enabled)
	if err != nil {
		writeError(w, http.StatusNotFound, "rule not found")
		return
	}
	tag, _ := h.dbOf(r.Context()).Exec(r.Context(), `DELETE FROM core.server_firewall_rules WHERE id = $1 AND server_id = $2`, body.ID, serverID)
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "rule not found")
		return
	}
	if !h.syncFirewallForServerHTTP(w, r, serverID) {
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
			INSERT INTO core.server_firewall_rules (id, server_id, protocol, port_from, port_to, enabled)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, body.ID, serverID, proto, portFrom, portTo, enabled)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ServerFirewallToggle(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, serverID, "firewall_toggle"); !ok {
		return
	}
	var body struct {
		ID      string `json:"id"`
		Enabled bool   `json:"enabled"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	var prevEnabled bool
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT enabled FROM core.server_firewall_rules WHERE id = $1 AND server_id = $2
	`, body.ID, serverID).Scan(&prevEnabled)
	if err != nil {
		writeError(w, http.StatusNotFound, "rule not found")
		return
	}
	tag, _ := h.dbOf(r.Context()).Exec(r.Context(), `UPDATE core.server_firewall_rules SET enabled = $3 WHERE id = $1 AND server_id = $2`, body.ID, serverID, body.Enabled)
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "rule not found")
		return
	}
	if !h.syncFirewallForServerHTTP(w, r, serverID) {
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `UPDATE core.server_firewall_rules SET enabled = $3 WHERE id = $1 AND server_id = $2`, body.ID, serverID, prevEnabled)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) loadServerPortRows(r *http.Request, serverID string) ([]map[string]any, int, error) {
	var primaryPort int
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(primary_port, 0) FROM core.servers
		WHERE id = $1
	`, serverID).Scan(&primaryPort)
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, port, protocol, purpose, created_at::text
		FROM core.server_ports
		WHERE server_id = $1
		ORDER BY port ASC, created_at ASC
	`, serverID)
	if err != nil {
		return nil, primaryPort, err
	}
	defer rows.Close()
	ports := make([]map[string]any, 0)
	for rows.Next() {
		var id, protocol, purpose, createdAt string
		var port int
		if scanErr := rows.Scan(&id, &port, &protocol, &purpose, &createdAt); scanErr == nil {
			ports = append(ports, map[string]any{
				"id":         id,
				"port":       port,
				"protocol":   protocol,
				"purpose":    purpose,
				"is_primary": port == primaryPort,
				"created_at": createdAt,
			})
		}
	}
	return ports, primaryPort, rows.Err()
}

func (h *Handler) ServerPortsList(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	_, ok := h.authorizeServerTab(w, r, serverID, "ports_list")
	if !ok {
		return
	}
	ports, primaryPort, err := h.loadServerPortRows(r, serverID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ports load failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ports":        ports,
		"primary_port": primaryPort,
		// Порт выдаёт персонал по обращению — клиент порты только видит.
		"can_edit_ports": false,
	})
}

func (h *Handler) ServerFriendsList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, id, "friends_list"); !ok {
		return
	}
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT f.id::text, f.user_id::text, u.email,
		       COALESCE(p.first_name, ''), COALESCE(p.last_name, ''),
		       f.permissions FROM core.server_friends f
		JOIN core.users u ON u.id = f.user_id
		LEFT JOIN core.user_profiles p ON p.user_id = u.id
		WHERE f.server_id = $1
	`, id)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var fid, uid, email, firstName, lastName string
			var perms []byte
			if rows.Scan(&fid, &uid, &email, &firstName, &lastName, &perms) == nil {
				entry := map[string]any{
					"id": fid, "user_id": uid, "email": email,
					"user_email": email, "user_name": firstName, "user_last_name": lastName,
					"permissions": json.RawMessage(perms),
				}
				var permMap map[string]bool
				if json.Unmarshal(perms, &permMap) == nil {
					for k, v := range permMap {
						entry[k] = v
					}
				}
				list = append(list, entry)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"friends": list})
}

func (h *Handler) ServerFriendsAdd(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	_, ok := h.authorizeServerTab(w, r, id, "friends_add")
	if !ok {
		return
	}
	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
		writeError(w, http.StatusBadRequest, "email required")
		return
	}
	var uid string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `SELECT id FROM core.users WHERE email = $1`, body.Email).Scan(&uid)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `INSERT INTO core.server_friends (server_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, id, uid)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ServerFriendsRemove(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, serverID, "friends_remove"); !ok {
		return
	}
	var body struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	var tag int64
	if body.ID != "" {
		res, _ := h.dbOf(r.Context()).Exec(r.Context(), `DELETE FROM core.server_friends WHERE id = $1 AND server_id = $2`, body.ID, serverID)
		tag = res.RowsAffected()
	} else if body.Email != "" {
		res, _ := h.dbOf(r.Context()).Exec(r.Context(), `
			DELETE FROM core.server_friends f
			USING core.users u
			WHERE f.user_id = u.id AND f.server_id = $1 AND u.email = $2
		`, serverID, body.Email)
		tag = res.RowsAffected()
	}
	if tag == 0 {
		writeError(w, http.StatusNotFound, "friend not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (h *Handler) ServerFriendsUpdate(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	_, ok := h.authorizeServerTab(w, r, serverID, "friends_update")
	if !ok {
		return
	}

	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	userID, _ := body["user_id"].(string)
	if strings.TrimSpace(userID) == "" {
		writeError(w, http.StatusBadRequest, "user_id required")
		return
	}

	permKeys := []string{
		"can_view_console", "can_view_logs", "can_view_metrics", "can_view_ftp",
		"can_view_mysql", "can_view_cron", "can_view_firewall", "can_view_ports",
		"can_view_settings", "can_view_friends",
		"can_start", "can_stop", "can_restart", "can_reinstall", "can_console_command",
		"can_files", "can_cron_manage", "can_firewall_manage",
		"can_settings_edit",
	}
	perms := map[string]bool{}
	for _, key := range permKeys {
		if v, ok := body[key]; ok {
			perms[key] = truthy(v)
		}
	}
	permsJSON, _ := json.Marshal(perms)

	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.server_friends SET permissions = $3::jsonb
		WHERE server_id = $1 AND user_id = $2::uuid
	`, serverID, userID, string(permsJSON))
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "friend not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		return t == "1" || strings.EqualFold(t, "true")
	default:
		return false
	}
}

const backupsDir = "/backups"

func (h *Handler) ServerBackupCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	if !h.authorizeServerAction(w, r, claims, id, "backup_create") {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if name := strings.TrimSpace(body.Name); name != "" {
		if err := backupname.Check(name); err != nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
	}

	var bid string
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.server_backups (server_id, status, source)
		VALUES ($1, 'running', 'manual') RETURNING id::text
	`, id).Scan(&bid)

	result, ok := h.agentCommandForServer(w, r, id, "backup_create", map[string]any{"name": body.Name})
	if !ok {
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `UPDATE core.server_backups SET status = 'failed' WHERE id = $1`, bid)
		return
	}
	filename, _ := result["filename"].(string)
	size := int64(0)
	if v, ok := result["size_bytes"].(float64); ok {
		size = int64(v)
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.server_backups SET status = 'completed', filename = $2, size_bytes = $3, completed_at = now() WHERE id = $1
	`, bid, filename, size)
	h.enqueueBackupUpload(r.Context(), id, bid, filename)
	writeJSON(w, http.StatusCreated, map[string]any{"backup_id": bid, "status": "completed", "filename": filename})
}

func (h *Handler) ServerBackupList(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	ctx := r.Context()
	if !h.authorizeServerAction(w, r, claims, id, "files_list") {
		return
	}
	nodeID, err := h.serverNodeID(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	_, _ = h.agentCommand(ctx, nodeID, id, "files_mkdir", map[string]any{"path": backupsDir})
	result, err := h.agentCommand(ctx, nodeID, id, "files_list", map[string]any{"path": backupsDir})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	files := result["files"]
	if files == nil {
		files = []any{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"backups": files, "entries": files})
}

func (h *Handler) ServerBackupDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := h.ensureServerForTenant(w, r, id); !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := backupname.Check(strings.TrimSpace(body.Name)); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	path := backupsDir + "/" + body.Name
	_, ok := h.agentCommandForServer(w, r, id, "files_delete", map[string]any{"path": path})
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ServerBackupRestore(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := h.ensureServerForTenant(w, r, id); !ok {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	if err := backupname.Check(strings.TrimSpace(body.Name)); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	_, ok := h.agentCommandForServer(w, r, id, "backup_restore", map[string]any{"name": body.Name})
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restored"})
}
