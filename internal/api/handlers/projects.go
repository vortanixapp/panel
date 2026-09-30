package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

const (
	projectNameMax    = 64
	projectCommentMax = 240
	projectsPerUser   = 30
)

type projectRow struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Comment  string  `json:"comment"`
	Servers  int     `json:"servers"`
	Members  int     `json:"members"`
	Monthly  float64 `json:"monthly_cost"`
	Currency string  `json:"currency"`
	Created  string  `json:"created_at"`
}

func projectName(raw string) string {
	name := strings.TrimSpace(raw)
	if len([]rune(name)) > projectNameMax {
		name = string([]rune(name)[:projectNameMax])
	}
	return name
}

func projectComment(raw string) string {
	comment := strings.TrimSpace(raw)
	if len([]rune(comment)) > projectCommentMax {
		comment = string([]rune(comment)[:projectCommentMax])
	}
	return comment
}

func (h *Handler) projectOwned(ctx context.Context, projectID, userID string) bool {
	var exists bool
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT true FROM core.projects WHERE id = $1::uuid AND user_id = $2::uuid
	`, projectID, userID).Scan(&exists)
	return err == nil && exists
}

func (h *Handler) ProjectsList(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT p.id::text, p.name, p.comment, p.created_at::text,
			(SELECT COUNT(*)::int FROM core.servers s WHERE s.project_id = p.id),
			(SELECT COUNT(*)::int FROM core.project_members m WHERE m.project_id = p.id),
			COALESCE((
				SELECT SUM(COALESCE(t.price_monthly, 0))
				FROM core.servers s
				LEFT JOIN core.tariffs t ON t.id = s.tariff_id
				WHERE s.project_id = p.id
			), 0)
		FROM core.projects p
		WHERE p.user_id = $1::uuid
		ORDER BY p.created_at
	`, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	list := []projectRow{}
	for rows.Next() {
		var item projectRow
		if rows.Scan(&item.ID, &item.Name, &item.Comment, &item.Created,
			&item.Servers, &item.Members, &item.Monthly) != nil {
			continue
		}
		item.Currency = h.defaultCurrency(r.Context())
		list = append(list, item)
	}
	var unassigned int
	_ = h.readerOf(r.Context()).QueryRow(r.Context(), `
		SELECT COUNT(*)::int FROM core.servers
		WHERE user_id = $1::uuid AND project_id IS NULL
	`, claims.UserID).Scan(&unassigned)
	writeJSON(w, http.StatusOK, map[string]any{
		"projects":           list,
		"unassigned_servers": unassigned,
	})
}

func (h *Handler) ProjectAvailableServers(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT s.id::text, s.name, s.game_id, COALESCE(g.name, s.game_id), COALESCE(s.status, 'stopped')
		FROM core.servers s
		LEFT JOIN core.games g ON g.slug = s.game_id
		WHERE s.user_id = $1::uuid AND s.project_id IS NULL
		ORDER BY s.created_at DESC
	`, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, name, gameID, gameName, status string
		if rows.Scan(&id, &name, &gameID, &gameName, &status) != nil {
			continue
		}
		list = append(list, map[string]any{
			"id": id, "name": name, "game_id": gameID, "game_name": gameName, "status": status,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"servers": list})
}

func (h *Handler) ProjectCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Name    string `json:"name"`
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := projectName(body.Name)
	if name == "" {
		writeCodedError(w, http.StatusBadRequest, "project_name_required", "Укажите название проекта")
		return
	}
	var count int
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT COUNT(*)::int FROM core.projects WHERE user_id = $1::uuid
	`, claims.UserID).Scan(&count)
	if count >= projectsPerUser {
		writeCodedError(w, http.StatusConflict, "projects_limit", "Достигнут предел числа проектов")
		return
	}
	var id string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.projects (user_id, name, comment)
		VALUES ($1::uuid, $2, $3)
		RETURNING id::text
	`, claims.UserID, name, projectComment(body.Comment)).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create project")
		return
	}
	auditWithIP(r.Context(), r, h.dbOf(r.Context()), claims.UserID, "project.create", "project:"+id, map[string]any{"name": name})
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "name": name})
}

func (h *Handler) ProjectDetail(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	var name, comment, created string
	err := h.readerOf(r.Context()).QueryRow(r.Context(), `
		SELECT name, comment, created_at::text FROM core.projects
		WHERE id = $1::uuid AND user_id = $2::uuid
	`, id, claims.UserID).Scan(&name, &comment, &created)
	if err != nil {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project": map[string]any{
			"id": id, "name": name, "comment": comment, "created_at": created,
			"currency": h.defaultCurrency(r.Context()),
		},
		"servers": h.projectServers(r.Context(), id),
		"members": h.projectMembers(r.Context(), id),
	})
}

func (h *Handler) projectServers(ctx context.Context, projectID string) []map[string]any {
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT s.id::text, s.name, COALESCE(s.comment, ''), s.game_id,
			COALESCE(g.name, s.game_id), COALESCE(s.status, 'stopped'),
			COALESCE(s.ip_address, ''), COALESCE(s.primary_port, 0),
			COALESCE(n.name, ''), COALESCE(n.country, ''),
			COALESCE(t.name, ''), COALESCE(t.price_monthly, 0),
			s.expires_at::text
		FROM core.servers s
		LEFT JOIN core.games g ON g.slug = s.game_id
		LEFT JOIN core.nodes n ON n.id = s.node_id
		LEFT JOIN core.tariffs t ON t.id = s.tariff_id
		WHERE s.project_id = $1::uuid
		ORDER BY s.created_at
	`, projectID)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, name, comment, gameID, gameName, status, ip, nodeName, country, tariffName string
		var port int
		var monthly float64
		var expires *string
		if rows.Scan(&id, &name, &comment, &gameID, &gameName, &status, &ip, &port,
			&nodeName, &country, &tariffName, &monthly, &expires) != nil {
			continue
		}
		item := map[string]any{
			"id": id, "name": name, "comment": comment,
			"game_id": gameID, "game_name": gameName, "status": status,
			"ip": ip, "port": port, "location": nodeName, "country": country,
			"tariff": tariffName, "monthly_cost": monthly,
		}
		if expires != nil {
			item["expires_at"] = *expires
		}
		list = append(list, item)
	}
	return list
}

func (h *Handler) projectMembers(ctx context.Context, projectID string) []map[string]any {
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT m.user_id::text, COALESCE(u.email, ''), m.permissions, m.created_at::text
		FROM core.project_members m
		LEFT JOIN core.users u ON u.id = m.user_id
		WHERE m.project_id = $1::uuid
		ORDER BY m.created_at
	`, projectID)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var userID, email, created string
		var permsRaw []byte
		if rows.Scan(&userID, &email, &permsRaw, &created) != nil {
			continue
		}
		perms := map[string]any{}
		_ = json.Unmarshal(permsRaw, &perms)
		list = append(list, map[string]any{
			"user_id": userID, "email": email,
			"permissions": friendViewerPermissions(perms),
			"created_at":  created,
		})
	}
	return list
}

func (h *Handler) ProjectUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	if !h.projectOwned(r.Context(), id, claims.UserID) {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	var body struct {
		Name    *string `json:"name"`
		Comment *string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Name != nil {
		name := projectName(*body.Name)
		if name == "" {
			writeCodedError(w, http.StatusBadRequest, "project_name_required", "Укажите название проекта")
			return
		}
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
			UPDATE core.projects SET name = $2, updated_at = now() WHERE id = $1::uuid
		`, id, name)
	}
	if body.Comment != nil {
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
			UPDATE core.projects SET comment = $2, updated_at = now() WHERE id = $1::uuid
		`, id, projectComment(*body.Comment))
	}
	auditWithIP(r.Context(), r, h.dbOf(r.Context()), claims.UserID, "project.update", "project:"+id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ProjectDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	if !h.projectOwned(r.Context(), id, claims.UserID) {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	for _, server := range h.projectServers(r.Context(), id) {
		serverID, _ := server["id"].(string)
		h.revokeProjectAccess(r.Context(), id, serverID, "")
	}
	_, err := h.dbOf(r.Context()).Exec(r.Context(), `
		DELETE FROM core.projects WHERE id = $1::uuid AND user_id = $2::uuid
	`, id, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete project")
		return
	}
	auditWithIP(r.Context(), r, h.dbOf(r.Context()), claims.UserID, "project.delete", "project:"+id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ProjectServersAssign(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	if !h.projectOwned(r.Context(), id, claims.UserID) {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	var body struct {
		ServerIDs []string `json:"server_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.ServerIDs) == 0 {
		writeError(w, http.StatusBadRequest, "server_ids required")
		return
	}
	moved := 0
	for _, serverID := range body.ServerIDs {
		res, err := h.dbOf(r.Context()).Exec(r.Context(), `
			UPDATE core.servers SET project_id = $1::uuid
			WHERE id = $2::uuid AND user_id = $3::uuid
		`, id, serverID, claims.UserID)
		if err != nil || res.RowsAffected() == 0 {
			continue
		}
		moved++
		h.grantProjectAccess(r.Context(), id, serverID)
	}
	auditWithIP(r.Context(), r, h.dbOf(r.Context()), claims.UserID, "project.servers", "project:"+id, map[string]any{"moved": moved})
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "moved": moved})
}

func (h *Handler) ProjectServersRemove(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	if !h.projectOwned(r.Context(), id, claims.UserID) {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	var body struct {
		ServerIDs []string `json:"server_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.ServerIDs) == 0 {
		writeError(w, http.StatusBadRequest, "server_ids required")
		return
	}
	for _, serverID := range body.ServerIDs {
		res, err := h.dbOf(r.Context()).Exec(r.Context(), `
			UPDATE core.servers SET project_id = NULL
			WHERE id = $1::uuid AND project_id = $2::uuid AND user_id = $3::uuid
		`, serverID, id, claims.UserID)
		if err != nil || res.RowsAffected() == 0 {
			continue
		}
		h.revokeProjectAccess(r.Context(), id, serverID, "")
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ProjectMemberAdd(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	if !h.projectOwned(r.Context(), id, claims.UserID) {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	var body struct {
		Email       string         `json:"email"`
		Permissions map[string]any `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Email) == "" {
		writeError(w, http.StatusBadRequest, "email required")
		return
	}
	var memberID string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT id::text FROM core.users WHERE lower(email) = lower($1)
	`, strings.TrimSpace(body.Email)).Scan(&memberID)
	if err != nil {
		if err == pgx.ErrNoRows {
			writeCodedError(w, http.StatusNotFound, "user_unknown",
				"Пользователь с такой почтой не зарегистрирован")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if memberID == claims.UserID {
		writeCodedError(w, http.StatusBadRequest, "member_self", "Вы и так владелец проекта")
		return
	}
	perms := friendViewerPermissions(body.Permissions)
	permsJSON, _ := json.Marshal(perms)
	_, err = h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.project_members (project_id, user_id, permissions)
		VALUES ($1::uuid, $2::uuid, $3::jsonb)
		ON CONFLICT (project_id, user_id) DO UPDATE SET permissions = EXCLUDED.permissions
	`, id, memberID, permsJSON)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to add member")
		return
	}
	h.syncProjectMemberAccess(r.Context(), id, memberID, permsJSON)
	auditWithIP(r.Context(), r, h.dbOf(r.Context()), claims.UserID, "project.member.add", "project:"+id, map[string]any{"user_id": memberID})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) ProjectMemberRemove(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	if !h.projectOwned(r.Context(), id, claims.UserID) {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == "" {
		writeError(w, http.StatusBadRequest, "user_id required")
		return
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		DELETE FROM core.project_members WHERE project_id = $1::uuid AND user_id = $2::uuid
	`, id, body.UserID)
	for _, server := range h.projectServers(r.Context(), id) {
		serverID, _ := server["id"].(string)
		h.revokeProjectAccess(r.Context(), id, serverID, body.UserID)
	}
	auditWithIP(r.Context(), r, h.dbOf(r.Context()), claims.UserID, "project.member.remove", "project:"+id, map[string]any{"user_id": body.UserID})
	writeJSON(w, http.StatusOK, map[string]string{"status": "removed"})
}

func (h *Handler) moveServerToProject(ctx context.Context, serverID, userID, projectID string) {
	var current string
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(project_id::text, '') FROM core.servers WHERE id = $1::uuid AND user_id = $2::uuid
	`, serverID, userID).Scan(&current)
	if current != "" {
		h.revokeProjectAccess(ctx, current, serverID, "")
	}
	if projectID == "" {
		_, _ = h.dbOf(ctx).Exec(ctx, `
			UPDATE core.servers SET project_id = NULL WHERE id = $1::uuid AND user_id = $2::uuid
		`, serverID, userID)
		return
	}
	if !h.projectOwned(ctx, projectID, userID) {
		return
	}
	res, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.servers SET project_id = $3::uuid WHERE id = $1::uuid AND user_id = $2::uuid
	`, serverID, userID, projectID)
	if err != nil || res.RowsAffected() == 0 {
		return
	}
	h.grantProjectAccess(ctx, projectID, serverID)
}

func (h *Handler) grantProjectAccess(ctx context.Context, projectID, serverID string) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT user_id::text, permissions FROM core.project_members WHERE project_id = $1::uuid
	`, projectID)
	if err != nil {
		return
	}
	defer rows.Close()
	type member struct {
		id    string
		perms []byte
	}
	members := []member{}
	for rows.Next() {
		var m member
		if rows.Scan(&m.id, &m.perms) != nil {
			continue
		}
		members = append(members, m)
	}
	for _, m := range members {
		_, _ = h.dbOf(ctx).Exec(ctx, `
			INSERT INTO core.server_friends (server_id, user_id, permissions)
			VALUES ($1::uuid, $2::uuid, $3::jsonb)
			ON CONFLICT (server_id, user_id) DO UPDATE SET permissions = EXCLUDED.permissions
		`, serverID, m.id, m.perms)
	}
}

func (h *Handler) syncProjectMemberAccess(ctx context.Context, projectID, memberID string, permsJSON []byte) {
	for _, server := range h.projectServers(ctx, projectID) {
		serverID, _ := server["id"].(string)
		if serverID == "" {
			continue
		}
		_, _ = h.dbOf(ctx).Exec(ctx, `
			INSERT INTO core.server_friends (server_id, user_id, permissions)
			VALUES ($1::uuid, $2::uuid, $3::jsonb)
			ON CONFLICT (server_id, user_id) DO UPDATE SET permissions = EXCLUDED.permissions
		`, serverID, memberID, permsJSON)
	}
}

func (h *Handler) revokeProjectAccess(ctx context.Context, projectID, serverID, memberID string) {
	if serverID == "" {
		return
	}
	if memberID != "" {
		_, _ = h.dbOf(ctx).Exec(ctx, `
			DELETE FROM core.server_friends WHERE server_id = $1::uuid AND user_id = $2::uuid
		`, serverID, memberID)
		return
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `
		DELETE FROM core.server_friends f
		USING core.project_members m
		WHERE f.server_id = $1::uuid AND m.project_id = $2::uuid AND m.user_id = f.user_id
	`, serverID, projectID)
}
