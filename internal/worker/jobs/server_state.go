package jobs

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/vortanixapp/panel/internal/worker/relay"
	"github.com/vortanixapp/panel/pkg/protocol"
)

func (r *Runner) moveServerAgentState(ctx context.Context, serverID, fromNodeID, toNodeID string) {
	jobs := r.serverCronJobs(ctx, serverID)
	rules := r.serverFirewallRules(ctx, serverID)
	send := func(nodeID, action string, payload map[string]any) {
		err := r.relay.SendCommand(ctx, nodeID, relay.CommandRequest{
			CommandID: uuid.NewString(), Action: action, ServerID: serverID, Payload: payload,
		})
		if err != nil {
			log.Printf("migrate: %s сервера %s на ноду %s не отправлен: %v", action, serverID, nodeID, err)
		}
	}
	if len(jobs) > 0 {
		send(toNodeID, protocol.ActionCronSync, map[string]any{"jobs": jobs, "tz": r.serverCronTimezone(ctx, serverID)})
		send(fromNodeID, protocol.ActionCronSync, map[string]any{"jobs": []any{}})
	}
	if len(rules) > 0 {
		send(toNodeID, protocol.ActionFirewallSync, map[string]any{"rules": rules})
		send(fromNodeID, protocol.ActionFirewallSync, map[string]any{"rules": []any{}})
	}
}

func (r *Runner) serverCronJobs(ctx context.Context, serverID string) []map[string]any {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, schedule, command, enabled FROM core.server_cron_jobs WHERE server_id = $1 ORDER BY created_at
	`, serverID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var list []map[string]any
	for rows.Next() {
		var id, schedule, command string
		var enabled bool
		if rows.Scan(&id, &schedule, &command, &enabled) == nil {
			list = append(list, map[string]any{"id": id, "schedule": schedule, "command": command, "enabled": enabled})
		}
	}
	return list
}

func (r *Runner) serverFirewallRules(ctx context.Context, serverID string) []map[string]any {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, protocol, port_from, port_to, enabled
		FROM core.server_firewall_rules WHERE server_id = $1 ORDER BY created_at
	`, serverID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var list []map[string]any
	for rows.Next() {
		var id, proto string
		var portFrom int
		var portTo *int
		var enabled bool
		if rows.Scan(&id, &proto, &portFrom, &portTo, &enabled) != nil {
			continue
		}
		item := map[string]any{"id": id, "protocol": proto, "port_from": portFrom, "enabled": enabled}
		if portTo != nil {
			item["port_to"] = *portTo
		}
		list = append(list, item)
	}
	return list
}

func (r *Runner) serverCronTimezone(ctx context.Context, serverID string) string {
	var tz string
	_ = r.db.QueryRow(ctx, `
		SELECT COALESCE(p.timezone, '')
		FROM core.servers s
		LEFT JOIN core.user_profiles p ON p.user_id = s.user_id
		WHERE s.id = $1
	`, serverID).Scan(&tz)
	if _, err := time.LoadLocation(tz); tz == "" || err != nil {
		return "UTC"
	}
	return tz
}
