package handlers

import (
	"context"
	"strings"

	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

var auditAlertActions = map[string]bool{
	"groups.update":                   true,
	"groups.create":                   true,
	"groups.delete":                   true,
	"user.delete":                     true,
	"refund.request":                  true,
	"user.impersonate":                true,
	"api_key.create":                  true,
	"webhook.create":                  true,
	"payment.refund":                  true,
	"billing.adjust":                  true,
	"ip.block":                        true,
	"server.migrate":                  true,
	"server.delete":                   true,
	"node.bulk.stop":                  true,
	"node.bulk.extend":                true,
	"location.delete":                 true,
	"location.agent_token_regenerate": true,
	"license.bind":                    true,
	"settings.update":                 true,
}

func (h *Handler) auditAlert(ctx context.Context, actorID, actorEmail, action, resource string) {
	if !auditAlertActions[action] {
		return
	}

	who := i18n.Raw(strings.TrimSpace(actorEmail))
	if who.Raw == "" {
		who = i18n.Key("notify.audit.staff")
	}
	params := i18n.Params{
		"action":   i18n.Key("notify.audit.action." + action),
		"who":      who,
		"resource": resource,
	}
	title := i18n.Key("notify.audit.title", params)
	body := i18n.Key("notify.audit.body", params)

	h.notifyStaff(ctx, notify.StaffAdmins, actorID, notify.Event{
		Kind:  notify.KindStaffAudit,
		Title: title,
		Body:  body,
		Meta:  map[string]any{"action": action, "resource": resource},
	})
}
