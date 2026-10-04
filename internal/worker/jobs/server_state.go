package jobs

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/vortanixapp/panel/internal/worker/relay"
	"github.com/vortanixapp/panel/pkg/gamecatalog"
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
		send(toNodeID, protocol.ActionFirewallSync, r.firewallPayloadFor(ctx, toNodeID, serverID, rules))
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
		SELECT id::text, protocol, port_from, port_to, enabled, action, COALESCE(source, '')
		FROM core.server_firewall_rules WHERE server_id = $1 ORDER BY created_at
	`, serverID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var list []map[string]any
	for rows.Next() {
		var id, proto, action, source string
		var portFrom int
		var portTo *int
		var enabled bool
		if rows.Scan(&id, &proto, &portFrom, &portTo, &enabled, &action, &source) != nil {
			continue
		}
		item := map[string]any{"id": id, "protocol": proto, "port_from": portFrom, "enabled": enabled, "action": action, "source": source}
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

const firewallRulesMinAgent = "0.1.95"

func (r *Runner) serverFirewallLimit(ctx context.Context, serverID string) (int, []int) {
	var limit, primary int
	_ = r.db.QueryRow(ctx, `
		SELECT COALESCE((config->>'firewall_conn_limit')::int, 0), COALESCE(primary_port, 0)
		FROM core.servers WHERE id = $1
	`, serverID).Scan(&limit, &primary)
	ports := []int{}
	if limit <= 0 {
		return 0, ports
	}
	seen := map[int]bool{}
	if primary > 0 {
		seen[primary] = true
		ports = append(ports, primary)
	}
	rows, err := r.db.Query(ctx, `
		SELECT port FROM core.server_ports WHERE server_id = $1 AND lower(protocol) = 'tcp'
	`, serverID)
	if err != nil {
		return limit, ports
	}
	defer rows.Close()
	for rows.Next() {
		var port int
		if rows.Scan(&port) == nil && !seen[port] {
			seen[port] = true
			ports = append(ports, port)
		}
	}
	return limit, ports
}

func (r *Runner) firewallPayloadFor(ctx context.Context, nodeID, serverID string, rules []map[string]any) map[string]any {
	var version string
	_ = r.db.QueryRow(ctx, `
		SELECT COALESCE(version, '') FROM core.node_daemons WHERE node_id = $1::uuid
	`, nodeID).Scan(&version)
	if !gamecatalog.AgentAtLeast(version, firewallRulesMinAgent) {
		kept := make([]map[string]any, 0, len(rules))
		for _, rule := range rules {
			action, _ := rule["action"].(string)
			source, _ := rule["source"].(string)
			if action == "allow" || source != "" {
				log.Printf("migrate: правило файрвола сервера %s по адресу не перенесено: агент на ноде %s старше %s", serverID, nodeID, firewallRulesMinAgent)
				continue
			}
			kept = append(kept, rule)
		}
		return map[string]any{"rules": kept}
	}
	limit, ports := r.serverFirewallLimit(ctx, serverID)
	return map[string]any{"rules": rules, "conn_limit": limit, "limit_ports": ports}
}
