package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

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
		"admin.groups.write",
		"admin.hosting.read",
		"admin.hosting.write",
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
		"admin.language.write":          "Язык (изменение)",
		"admin.logs.read":               "Логи (просмотр)",
		"admin.jobs.read":               "Очередь задач (просмотр)",
		"admin.jobs.write":              "Очередь задач (перезапуск/отмена)",
		"admin.updates.read":            "Обновления (просмотр)",
		"admin.groups.write":            "Группы (управление правами)",
		"admin.hosting.read":            "Веб-хостинг (просмотр)",
		"admin.hosting.write":           "Веб-хостинг (управление)",
	}
}

func rbacGroupLabels() map[string]string {
	return map[string]string{
		rbacRoleUser:    "Пользователь",
		rbacRoleSupport: "Тех поддержка",
		rbacRoleAdmin:   "Админ",
	}
}

func rbacRoles() []string {
	return []string{rbacRoleUser, rbacRoleSupport, rbacRoleAdmin}
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
	keys := rbacAllPermissionKeys()
	filtered := make(map[string]bool, len(keys))
	for _, k := range keys {
		filtered[k] = permissions[k]
	}
	b, err := json.Marshal(filtered)
	if err != nil {
		return err
	}
	_, err = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.tenant_settings (key, value) VALUES ($1, $2::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
	`, "rbac.permissions."+role, b)
	return err
}

func (h *Handler) rbacPermissionMatrix(ctx context.Context) map[string]map[string]bool {
	keys := rbacAllPermissionKeys()
	matrix := make(map[string]map[string]bool, len(rbacRoles()))
	for _, role := range rbacRoles() {
		rolePerms := h.rbacLoadRolePermissions(ctx, role)
		normalized := make(map[string]bool, len(keys))
		for _, k := range keys {
			normalized[k] = rolePerms[k]
		}
		matrix[role] = normalized
	}
	return matrix
}

func (h *Handler) GetGroups(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"groups":           rbacGroupLabels(),
		"keys":             rbacAllPermissionKeys(),
		"matrix":           h.rbacPermissionMatrix(r.Context()),
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

	keys := rbacAllPermissionKeys()
	for _, role := range rbacRoles() {
		incoming := body.Permissions[role]
		if incoming == nil {
			incoming = map[string]any{}
		}
		normalized := make(map[string]bool, len(keys))
		for _, key := range keys {
			normalized[key] = rbacTruthy(incoming[key])
		}
		if err := h.rbacSaveRolePermissions(r.Context(), role, normalized); err != nil {
			writeError(w, http.StatusInternalServerError, "save failed")
			return
		}
	}

	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "groups.update", "rbac", nil)
	h.auditAlert(r.Context(), claims.UserID, claims.Email, "groups.update", "права ролей")

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"message": "Права групп обновлены.",
	})
}

func rbacIsWriteMethod(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func (h *Handler) rbacCan(ctx context.Context, role, permission string) bool {
	if permission == "" {
		return false
	}
	perms := h.rbacLoadRolePermissions(ctx, rbacNormalizeRole(role))
	return perms[permission]
}

func (h *Handler) requirePermission(w http.ResponseWriter, r *http.Request, claims *paneljwt.Claims, permission string) bool {
	if claims == nil || !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return false
	}
	if h.rbacCan(r.Context(), claims.Role, permission) {
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

	if section == "locations" {
		if strings.Contains(slashPath, "/daemon") || strings.Contains(slashPath, "/pull-daemon") {
			return readWrite("admin.daemons.read", "admin.daemons.write")
		}
		if strings.Contains(slashPath, "/setup") {
			return readWrite("admin.locations.read", "admin.locations.write")
		}
	}
	if section == "daemons" {
		return readWrite("admin.daemons.read", "admin.daemons.write")
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
		return "admin.updates.read", false
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
	case "promotions", "promo":
		return readWrite("admin.promotions.read", "admin.promotions.write")
	case "mailings":
		return readWrite("admin.mailings.read", "admin.mailings.write")
	case "hosting":
		return readWrite("admin.hosting.read", "admin.hosting.write")
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
				writeError(w, http.StatusForbidden, "ключ не имеет права admin.dashboard.read")
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
				writeError(w, http.StatusForbidden, "ключ не имеет права "+permission)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		if permission == "admin.groups.write" && isAdminRole(claims.Role) {
			next.ServeHTTP(w, r)
			return
		}

		if !h.rbacCan(r.Context(), claims.Role, permission) {
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
		if !h.rbacCan(r.Context(), claims.Role, permission) {
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
