package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/vortanix/vortanix/internal/api/licensestate"
)

func (h *Handler) checkServerQuota(ctx context.Context, tenantID string, w http.ResponseWriter) bool {
	return h.checkQuota(ctx, w, quotaCheck{
		resource: "серверов",
		countSQL: `SELECT COUNT(*) FROM core.servers WHERE tenant_id = $1`,
		tenantID: tenantID,
		limitOf:  func(l licensestate.Limits) int { return l.MaxServers },
	})
}

func (h *Handler) checkNodeQuota(ctx context.Context, tenantID string, w http.ResponseWriter) bool {
	return h.checkQuota(ctx, w, quotaCheck{
		resource: "нод",
		countSQL: `SELECT COUNT(*) FROM core.nodes WHERE tenant_id = $1`,
		tenantID: tenantID,
		limitOf:  func(l licensestate.Limits) int { return l.MaxNodes },
	})
}

func (h *Handler) checkAdminQuota(ctx context.Context, tenantID string, w http.ResponseWriter) bool {
	return h.checkQuota(ctx, w, quotaCheck{
		resource: "администраторов",
		countSQL: `SELECT COUNT(*) FROM core.users
		           WHERE tenant_id = $1 AND role IN ('owner', 'admin') AND status = 'active'`,
		tenantID: tenantID,
		limitOf:  func(l licensestate.Limits) int { return l.MaxAdmins },
	})
}

type quotaCheck struct {
	resource string
	countSQL string
	tenantID string
	limitOf  func(licensestate.Limits) int
}

func (h *Handler) checkQuota(ctx context.Context, w http.ResponseWriter, c quotaCheck) bool {
	st := h.licenseFor(ctx)

	if st.BlocksCreation() {
		writeCodedError(w, http.StatusForbidden, "license_"+string(st.Mode),
			"создание недоступно: "+st.Reason)
		return false
	}
	if !st.LimitsKnown {
		writeCodedError(w, http.StatusServiceUnavailable, "license_unavailable",
			"лимиты тарифа ещё не получены, повторите через минуту")
		return false
	}

	limit := c.limitOf(st.Limits)
	if limit == 0 {
		return true
	}

	var used int
	if err := h.dbOf(ctx).QueryRow(ctx, c.countSQL, c.tenantID).Scan(&used); err != nil {
		writeCodedError(w, http.StatusServiceUnavailable, "quota_check_failed",
			"не удалось проверить лимит "+c.resource)
		return false
	}

	if !st.Limits.Allows(used, limit) {
		writeCodedError(w, http.StatusForbidden, "quota_exceeded",
			fmt.Sprintf("достигнут лимит тарифа: %d %s", limit, c.resource))
		return false
	}
	return true
}

func (h *Handler) CountUsage(ctx context.Context) (servers, nodes, admins, cpu int) {
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM core.servers),
			(SELECT COUNT(*) FROM core.nodes),
			(SELECT COUNT(*) FROM core.users WHERE role IN ('owner','admin') AND status = 'active'),
			COALESCE((SELECT ROUND(AVG(cpu_pct))::int FROM core.server_metric_points
			          WHERE ts > now() - interval '5 minutes'), 0)
	`).Scan(&servers, &nodes, &admins, &cpu)
	return servers, nodes, admins, cpu
}

func writeCodedError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
		"code":  code,
	})
}
