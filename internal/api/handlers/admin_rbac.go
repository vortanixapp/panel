package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/vortanixapp/panel/internal/api/paneljwt"
)

const (
	rbacRoleUser    = "user"
	rbacRoleSupport = "support"
	rbacRoleAdmin   = "admin"
)

func rbacAllPermissionKeys() []string {
	return []string{
		"admin.dashboard.read",
		"admin.billing.read",
		"admin.billing.write",
		"admin.users.read",
		"admin.users.write",
		"admin.servers.read",
		"admin.servers.write",
		"admin.support.read",
		"admin.support.write",
		"admin.kb.read",
		"admin.kb.write",
		"admin.notifications.read",
		"admin.bug_report.read",
		"admin.bug_report.write",
		"admin.locations.read",
		"admin.locations.write",
		"admin.mysql.read",
		"admin.mysql.write",
		"admin.games.read",
		"admin.games.write",
		"admin.tariffs.read",
		"admin.tariffs.write",
		"admin.daemons.read",
		"admin.daemons.write",
		"admin.plugins.read",
		"admin.plugins.write",
		"admin.maps.read",
		"admin.maps.write",
		"admin.news.read",
		"admin.news.write",
		"admin.promotions.read",
		"admin.promotions.write",
		"admin.bonuses.read",
		"admin.mailings.read",
		"admin.mailings.write",
		"admin.settings.write",
		"admin.payment_providers.write",
		"admin.language.write",
		"admin.logs.read",
		"admin.jobs.read",
		"admin.jobs.write",
		"admin.updates.read",
		"admin.updates.write",
		"admin.groups.write",
		"admin.hosting.read",
		"admin.hosting.write",
		"admin.whmcs.write",
		"admin.template.write",
	}
}

func rbacPermissionLabels() map[string]string {
	return map[string]string{
		"admin.dashboard.read":          "Главная (просмотр)",
		"admin.billing.read":            "Биллинг (просмотр)",
		"admin.billing.write":           "Биллинг (изменение)",
		"admin.users.read":              "Пользователи (просмотр)",
		"admin.users.write":             "Пользователи (изменение)",
		"admin.servers.read":            "Серверы (просмотр)",
		"admin.servers.write":           "Серверы (изменение)",
		"admin.support.read":            "Тех. поддержка (просмотр)",
		"admin.support.write":           "Тех. поддержка (ответы/действия)",
		"admin.kb.read":                 "База знаний (просмотр)",
		"admin.kb.write":                "База знаний (изменение)",
		"admin.notifications.read":      "Уведомления (просмотр)",
		"admin.bug_report.read":         "Баг-репорт (просмотр)",
		"admin.bug_report.write":        "Баг-репорт (отправка)",
		"admin.locations.read":          "Локации (просмотр)",
		"admin.locations.write":         "Локации (изменение/установка)",
		"admin.mysql.read":              "MySQL (просмотр)",
		"admin.mysql.write":             "MySQL (управление)",
		"admin.games.read":              "Игры (просмотр)",
		"admin.games.write":             "Игры (изменение)",
		"admin.tariffs.read":            "Тарифы (просмотр)",
		"admin.tariffs.write":           "Тарифы (изменение)",
		"admin.daemons.read":            "Daemons (просмотр)",
		"admin.daemons.write":           "Daemons (управление)",
		"admin.plugins.read":            "Плагины (просмотр)",
		"admin.plugins.write":           "Плагины (изменение)",
		"admin.maps.read":               "Карты (просмотр)",
		"admin.maps.write":              "Карты (изменение)",
		"admin.news.read":               "Новости (просмотр)",
		"admin.news.write":              "Новости (изменение)",
		"admin.promotions.read":         "Акции (просмотр)",
		"admin.promotions.write":        "Акции (изменение)",
		"admin.bonuses.read":            "Бонусы (просмотр/управление)",
		"admin.mailings.read":           "Рассылка (просмотр)",
		"admin.mailings.write":          "Рассылка (изменение/запуск)",
		"admin.settings.write":          "Настройки (изменение)",
		"admin.payment_providers.write": "Платёжные провайдеры (изменение)",
		"admin.language.write":          "Языки и переводы (изменение)",
		"admin.logs.read":               "Логи (просмотр)",
		"admin.jobs.read":               "Очередь задач (просмотр)",
		"admin.jobs.write":              "Очередь задач (перезапуск/отмена)",
		"admin.updates.read":            "Обновления (просмотр)",
		"admin.updates.write":           "Обновления (установка и автообновление)",
		"admin.groups.write":            "Группы (управление правами)",
		"admin.hosting.read":            "Веб-хостинг (просмотр)",
		"admin.hosting.write":           "Веб-хостинг (управление)",
		"admin.whmcs.write":             "WHMCS (подключение модуля биллинга)",
		"admin.template.write":          "Шаблон сайта и меню (изменение)",
	}
}

func rbacGroupLabels() map[string]string {
	return map[string]string{
		rbacRoleUser:    "Пользователь",
		rbacRoleSupport: "Тех поддержка",
		rbacRoleAdmin:   "Админ",
	}
}

func rbacDefaultRolePermissions(role string) map[string]bool {
	keys := rbacAllPermissionKeys()
	perms := make(map[string]bool, len(keys))
	for _, k := range keys {
		perms[k] = false
	}

	switch role {
	case rbacRoleAdmin, "owner":
		for _, k := range keys {
			perms[k] = true
		}
	case rbacRoleSupport:
		for _, k := range []string{
			"admin.dashboard.read",
			"admin.support.read",
			"admin.support.write",
			"admin.logs.read",
			"admin.jobs.read",
			"admin.notifications.read",
		} {
			perms[k] = true
		}
	}
	return perms
}

func rbacNormalizeRole(role string) string {
	switch role {
	case rbacRoleSupport, rbacRoleAdmin:
		return role
	case "owner":
		return rbacRoleAdmin
	default:
		return rbacRoleUser
	}
}

func rbacNormalizePermissions(source map[string]bool) map[string]bool {
	keys := rbacAllPermissionKeys()
	out := make(map[string]bool, len(keys))
	for _, k := range keys {
		out[k] = source[k]
	}
	return out
}

func rbacCustomGroupKey(key string) bool {
	_, err := uuid.Parse(key)
	return err == nil
}

func (h *Handler) rbacLoadRolePermissions(ctx context.Context, role string) map[string]bool {
	defaults := rbacDefaultRolePermissions(role)
	key := "rbac.permissions." + role

	var raw []byte
	err := h.dbOf(ctx).QueryRow(ctx, `SELECT value FROM core.tenant_settings WHERE key = $1`, key).Scan(&raw)
	if err == nil && len(raw) > 0 {
		var stored map[string]bool
		if json.Unmarshal(raw, &stored) == nil {
			for k, v := range stored {
				defaults[k] = v
			}
			return defaults
		}
	}

	var nestedRaw []byte
	if h.dbOf(ctx).QueryRow(ctx, `SELECT value FROM core.tenant_settings WHERE key = 'rbac.permissions'`).Scan(&nestedRaw) == nil && len(nestedRaw) > 0 {
		var nested map[string]map[string]bool
		if json.Unmarshal(nestedRaw, &nested) == nil {
			if stored, ok := nested[role]; ok {
				for k, v := range stored {
					defaults[k] = v
				}
			}
		}
	}

	return defaults
}

func (h *Handler) rbacSaveRolePermissions(ctx context.Context, role string, permissions map[string]bool) error {
	b, err := json.Marshal(rbacNormalizePermissions(permissions))
	if err != nil {
		return err
	}
	_, err = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.tenant_settings (key, value) VALUES ($1, $2::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
	`, "rbac.permissions."+role, b)
	return err
}

type rbacGroup struct {
	Key         string          `json:"key"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	System      bool            `json:"system"`
	Members     int             `json:"members"`
	Permissions map[string]bool `json:"permissions"`
}

func (h *Handler) rbacGroups(ctx context.Context) ([]rbacGroup, error) {
	db := h.dbOf(ctx)
	var admins, support int
	if err := db.QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE role IN ('admin', 'owner'))::int,
		       COUNT(*) FILTER (WHERE role = 'support' AND staff_group_id IS NULL)::int
		FROM core.users
	`).Scan(&admins, &support); err != nil {
		return nil, err
	}
	labels := rbacGroupLabels()
	groups := []rbacGroup{
		{
			Key: rbacRoleAdmin, Name: labels[rbacRoleAdmin], System: true, Members: admins,
			Permissions: rbacNormalizePermissions(h.rbacLoadRolePermissions(ctx, rbacRoleAdmin)),
		},
		{
			Key: rbacRoleSupport, Name: labels[rbacRoleSupport], System: true, Members: support,
			Permissions: rbacNormalizePermissions(h.rbacLoadRolePermissions(ctx, rbacRoleSupport)),
		},
	}

	rows, err := db.Query(ctx, `
		SELECT g.id::text, g.name, g.description, g.permissions,
		       (SELECT COUNT(*)::int FROM core.users u WHERE u.staff_group_id = g.id)
		FROM core.staff_groups g
		ORDER BY g.position, g.created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var g rbacGroup
		var raw []byte
		if err := rows.Scan(&g.Key, &g.Name, &g.Description, &raw, &g.Members); err != nil {
			return nil, err
		}
		stored := map[string]bool{}
		_ = json.Unmarshal(raw, &stored)
		g.Permissions = rbacNormalizePermissions(stored)
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (h *Handler) rbacFindGroup(ctx context.Context, key string) (rbacGroup, bool) {
	groups, err := h.rbacGroups(ctx)
	if err != nil {
		return rbacGroup{}, false
	}
	for _, g := range groups {
		if g.Key == key {
			return g, true
		}
	}
	return rbacGroup{}, false
}

func (h *Handler) rbacClaimsPermissions(ctx context.Context, claims *paneljwt.Claims) map[string]bool {
	if claims.Role == rbacRoleSupport && claims.UserID != "" {
		var raw []byte
		err := h.dbOf(ctx).QueryRow(ctx, `
			SELECT g.permissions
			FROM core.users u
			JOIN core.staff_groups g ON g.id = u.staff_group_id
			WHERE u.id::text = $1 AND u.role = 'support'
		`, claims.UserID).Scan(&raw)
		if err == nil {
			stored := map[string]bool{}
			_ = json.Unmarshal(raw, &stored)
			return rbacNormalizePermissions(stored)
		}
	}
	return h.rbacLoadRolePermissions(ctx, rbacNormalizeRole(claims.Role))
}

func rbacGroupFields(name, description string) (string, string, string) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	if name == "" || utf8.RuneCountInString(name) > 64 {
		return "", "", "название группы — от 1 до 64 символов"
	}
	if utf8.RuneCountInString(description) > 255 {
		return "", "", "описание группы — не длиннее 255 символов"
	}
	return name, description, ""
}

func rbacDuplicateName(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (h *Handler) GetGroups(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}

	groups, err := h.rbacGroups(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось загрузить группы")
		return
	}
	labels := make(map[string]string, len(groups))
	matrix := make(map[string]map[string]bool, len(groups))
	for _, g := range groups {
		labels[g.Key] = g.Name
		matrix[g.Key] = g.Permissions
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items":            groups,
		"groups":           labels,
		"keys":             rbacAllPermissionKeys(),
		"matrix":           matrix,
		"permissionLabels": rbacPermissionLabels(),
	})
}

func (h *Handler) UpdateGroups(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}

	var body struct {
		Permissions map[string]map[string]any `json:"permissions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}

	ctx := r.Context()
	keys := rbacAllPermissionKeys()
	for groupKey, incoming := range body.Permissions {
		normalized := make(map[string]bool, len(keys))
		for _, key := range keys {
			normalized[key] = rbacTruthy(incoming[key])
		}
		switch {
		case groupKey == rbacRoleAdmin || groupKey == rbacRoleSupport:
			if err := h.rbacSaveRolePermissions(ctx, groupKey, normalized); err != nil {
				writeError(w, http.StatusInternalServerError, "save failed")
				return
			}
		case rbacCustomGroupKey(groupKey):
			raw, _ := json.Marshal(normalized)
			if _, err := h.dbOf(ctx).Exec(ctx, `
				UPDATE core.staff_groups SET permissions = $2::jsonb, updated_at = now() WHERE id = $1::uuid
			`, groupKey, raw); err != nil {
				writeError(w, http.StatusInternalServerError, "save failed")
				return
			}
		}
	}

	audit(ctx, h.dbOf(ctx), claims.UserID, "groups.update", "rbac", nil)
	h.auditAlert(ctx, claims.UserID, claims.Email, "groups.update", "права ролей")

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "Права групп обновлены.",
	})
}

func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		CopyFrom    string `json:"copy_from"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	name, description, msg := rbacGroupFields(body.Name, body.Description)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	ctx := r.Context()
	permissions := rbacNormalizePermissions(nil)
	if from := strings.TrimSpace(body.CopyFrom); from != "" {
		source, found := h.rbacFindGroup(ctx, from)
		if !found {
			writeError(w, http.StatusBadRequest, "Группа для копирования прав не найдена")
			return
		}
		permissions = source.Permissions
	}
	raw, _ := json.Marshal(permissions)

	var id string
	err := h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.staff_groups (name, description, permissions, position)
		VALUES ($1, $2, $3::jsonb, COALESCE((SELECT MAX(position) + 1 FROM core.staff_groups), 0))
		RETURNING id::text
	`, name, description, raw).Scan(&id)
	if err != nil {
		if rbacDuplicateName(err) {
			writeError(w, http.StatusConflict, "Группа с таким названием уже есть")
			return
		}
		writeError(w, http.StatusInternalServerError, "Не удалось создать группу")
		return
	}

	audit(ctx, h.dbOf(ctx), claims.UserID, "groups.create", id, map[string]any{"name": name})
	h.auditAlert(ctx, claims.UserID, claims.Email, "groups.create", name)

	writeJSON(w, http.StatusCreated, map[string]any{
		"ok":    true,
		"group": rbacGroup{Key: id, Name: name, Description: description, Permissions: permissions},
	})
}

func (h *Handler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	key := chi.URLParam(r, "key")
	if !rbacCustomGroupKey(key) {
		writeError(w, http.StatusBadRequest, "Системные группы не переименовываются")
		return
	}

	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	name, description, msg := rbacGroupFields(body.Name, body.Description)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	ctx := r.Context()
	tag, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.staff_groups SET name = $2, description = $3, updated_at = now() WHERE id = $1::uuid
	`, key, name, description)
	if err != nil {
		if rbacDuplicateName(err) {
			writeError(w, http.StatusConflict, "Группа с таким названием уже есть")
			return
		}
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить группу")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Группа не найдена")
		return
	}

	audit(ctx, h.dbOf(ctx), claims.UserID, "groups.rename", key, map[string]any{"name": name})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	key := chi.URLParam(r, "key")
	if !rbacCustomGroupKey(key) {
		writeError(w, http.StatusBadRequest, "Системные группы не удаляются")
		return
	}

	ctx := r.Context()
	var name string
	var members int
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT g.name, (SELECT COUNT(*)::int FROM core.users u WHERE u.staff_group_id = g.id)
		FROM core.staff_groups g WHERE g.id = $1::uuid
	`, key).Scan(&name, &members)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "Группа не найдена")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось удалить группу")
		return
	}
	if members > 0 {
		writeError(w, http.StatusConflict,
			fmt.Sprintf("в группе %d сотрудник(ов): переведите их в другую группу перед удалением", members))
		return
	}
	if _, err := h.dbOf(ctx).Exec(ctx, `DELETE FROM core.staff_groups WHERE id = $1::uuid`, key); err != nil {
		writeError(w, http.StatusConflict, "В группе появились сотрудники: переведите их в другую группу перед удалением")
		return
	}

	audit(ctx, h.dbOf(ctx), claims.UserID, "groups.delete", key, map[string]any{"name": name})
	h.auditAlert(ctx, claims.UserID, claims.Email, "groups.delete", name)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func staffRank(role string) int {
	switch role {
	case "owner":
		return 3
	case rbacRoleAdmin:
		return 2
	case rbacRoleSupport:
		return 1
	default:
		return 0
	}
}

func canManageUser(actorRole, targetRole string) bool {
	if actorRole == "owner" {
		return true
	}
	return staffRank(actorRole) > staffRank(targetRole)
}

func rbacIsWriteMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func (h *Handler) rbacCan(ctx context.Context, claims *paneljwt.Claims, permission string) bool {
	if permission == "" || claims == nil {
		return false
	}
	return h.rbacClaimsPermissions(ctx, claims)[permission]
}

func (h *Handler) requirePermission(w http.ResponseWriter, r *http.Request, claims *paneljwt.Claims, permission string) bool {
	if claims == nil || !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return false
	}
	if h.rbacCan(r.Context(), claims, permission) {
		return true
	}
	writeError(w, http.StatusForbidden, "forbidden")
	return false
}

func rbacResolveAdminPermission(path, method string) (permission string, bypass bool) {
	normalized := strings.Trim(path, "/")
	parts := strings.Split(normalized, "/")

	adminIdx := -1
	for i, p := range parts {
		if p == "admin" {
			adminIdx = i
			break
		}
	}
	if adminIdx < 0 || adminIdx+1 >= len(parts) {
		return "", false
	}

	afterAdmin := parts[adminIdx+1:]
	section := afterAdmin[0]
	isWrite := rbacIsWriteMethod(method)
	slashPath := "/" + normalized + "/"

	readWrite := func(readKey, writeKey string) (string, bool) {
		if isWrite {
			return writeKey, false
		}
		return readKey, false
	}

	if section == "games" && strings.Contains(slashPath, "/versions") {
		return readWrite("admin.games.read", "admin.games.write")
	}

	switch section {
	case "dashboard":
		return "", true
	case "search":
		return "", true
	case "servers":
		return readWrite("admin.servers.read", "admin.servers.write")
	case "support":
		return readWrite("admin.support.read", "admin.support.write")
	case "bug-report":
		return readWrite("admin.bug_report.read", "admin.bug_report.write")
	case "kb":
		return readWrite("admin.kb.read", "admin.kb.write")
	case "billing":
		return readWrite("admin.billing.read", "admin.billing.write")
	case "analytics":
		return "admin.billing.read", false
	case "accounting":
		return readWrite("admin.billing.read", "admin.billing.write")
	case "legal":
		return "admin.settings.write", false
	case "abuse":
		return readWrite("admin.servers.read", "admin.servers.write")
	case "payment-providers":
		return "admin.payment_providers.write", false
	case "settings":
		return "admin.settings.write", false
	case "api-keys", "webhooks":
		return "admin.settings.write", false
	case "language":
		return "admin.language.write", false
	case "logs":
		return "admin.logs.read", false
	case "security":
		if isWrite {
			return "admin.settings.write", false
		}
		return "admin.logs.read", false
	case "jobs":
		return readWrite("admin.jobs.read", "admin.jobs.write")
	case "updates":
		return readWrite("admin.updates.read", "admin.updates.write")
	case "groups":
		return "admin.groups.write", false
	case "locations", "images":
		return readWrite("admin.locations.read", "admin.locations.write")
	case "daemons":
		return readWrite("admin.daemons.read", "admin.daemons.write")
	case "mysql":
		return readWrite("admin.mysql.read", "admin.mysql.write")
	case "games":
		return readWrite("admin.games.read", "admin.games.write")
	case "tariffs":
		return readWrite("admin.tariffs.read", "admin.tariffs.write")
	case "plugins":
		return readWrite("admin.plugins.read", "admin.plugins.write")
	case "maps":
		return readWrite("admin.maps.read", "admin.maps.write")
	case "news":
		return readWrite("admin.news.read", "admin.news.write")
	case "promotions", "promo", "daily-bonus":
		return readWrite("admin.promotions.read", "admin.promotions.write")
	case "mailings", "mail-templates", "mail-log":
		return readWrite("admin.mailings.read", "admin.mailings.write")
	case "hosting":
		return readWrite("admin.hosting.read", "admin.hosting.write")
	case "whmcs":
		return "admin.whmcs.write", false
	case "template":
		return "admin.template.write", false
	case "site-menu":
		return "", true
	default:
		return "", false
	}
}

func (h *Handler) adminRBACMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := tenantClaims(r.Context())
		if !ok || !isStaffRole(claims.Role) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		permission, bypass := rbacResolveAdminPermission(r.URL.Path, r.Method)
		if bypass {
			if key, isKey := apiKeyFromContext(r.Context()); isKey && !key.Scopes["admin.dashboard.read"] {
				writeError(w, http.StatusForbidden, "Ключ не имеет права admin.dashboard.read")
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		if permission == "" {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		if key, ok := apiKeyFromContext(r.Context()); ok {
			if !key.Scopes[permission] {
				writeError(w, http.StatusForbidden, "Ключ не имеет права "+permission)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		if permission == "admin.groups.write" && isAdminRole(claims.Role) {
			next.ServeHTTP(w, r)
			return
		}

		if !h.rbacCan(r.Context(), claims, permission) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *Handler) usersRBACMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := tenantClaims(r.Context())
		if !ok || !isStaffRole(claims.Role) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		permission := "admin.users.read"
		if rbacIsWriteMethod(r.Method) {
			permission = "admin.users.write"
		}
		if !h.rbacCan(r.Context(), claims, permission) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func rbacTruthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case int:
		return t != 0
	case json.Number:
		n, _ := t.Int64()
		return n != 0
	case string:
		return t == "1" || t == "true"
	default:
		return false
	}
}
