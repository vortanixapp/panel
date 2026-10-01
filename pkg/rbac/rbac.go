package rbac

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

const (
	RoleUser    = "user"
	RoleSupport = "support"
	RoleAdmin   = "admin"
	RoleOwner   = "owner"
)

type Querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func AllKeys() []string {
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

func DefaultRole(role string) map[string]bool {
	keys := AllKeys()
	perms := make(map[string]bool, len(keys))
	for _, k := range keys {
		perms[k] = false
	}

	switch role {
	case RoleAdmin, RoleOwner:
		for _, k := range keys {
			perms[k] = true
		}
	case RoleSupport:
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

func NormalizeRole(role string) string {
	switch role {
	case RoleSupport, RoleAdmin:
		return role
	case RoleOwner:
		return RoleAdmin
	default:
		return RoleUser
	}
}

func Normalize(source map[string]bool) map[string]bool {
	keys := AllKeys()
	out := make(map[string]bool, len(keys))
	for _, k := range keys {
		out[k] = source[k]
	}
	return out
}

func LoadRole(ctx context.Context, db Querier, role string) map[string]bool {
	defaults := DefaultRole(role)
	key := "rbac.permissions." + role

	var raw []byte
	err := db.QueryRow(ctx, `SELECT value FROM core.tenant_settings WHERE key = $1`, key).Scan(&raw)
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
	if db.QueryRow(ctx, `SELECT value FROM core.tenant_settings WHERE key = 'rbac.permissions'`).Scan(&nestedRaw) == nil && len(nestedRaw) > 0 {
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

func ForUser(ctx context.Context, db Querier, userID, role string) map[string]bool {
	if role == RoleSupport && userID != "" {
		var raw []byte
		err := db.QueryRow(ctx, `
			SELECT g.permissions
			FROM core.users u
			JOIN core.staff_groups g ON g.id = u.staff_group_id
			WHERE u.id::text = $1 AND u.role = 'support'
		`, userID).Scan(&raw)
		if err == nil {
			stored := map[string]bool{}
			_ = json.Unmarshal(raw, &stored)
			return Normalize(stored)
		}
	}
	return LoadRole(ctx, db, NormalizeRole(role))
}
