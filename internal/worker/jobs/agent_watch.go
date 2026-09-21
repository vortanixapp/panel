package jobs

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/buildinfo"
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
	"github.com/vortanixapp/panel/pkg/updates"
)

const (
	agentWatchInterval    = 5 * time.Minute
	agentOutdatedGrace    = time.Hour
	settingAgentTargetAt  = "updates.agents.target_seen"
	agentOutdatedMaxNames = 10
)

func (r *Runner) AgentWatchLoop(ctx context.Context) {
	ticker := time.NewTicker(agentWatchInterval)
	defer ticker.Stop()
	for {
		r.watchAgents(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *Runner) watchAgents(ctx context.Context) {
	r.expireAgentUpdates(ctx)
	r.notifyFailedAgentUpdates(ctx)
	r.notifyOutdatedAgents(ctx)
}

func (r *Runner) expireAgentUpdates(ctx context.Context) {
	if _, err := r.db.Exec(ctx, `
		UPDATE core.nodes SET agent_update = agent_update || jsonb_build_object(
			'status', 'failed', 'error', 'агент не сообщил о результате за 15 минут',
			'finished_at', now(), 'updated_at', now())
		WHERE agent_update->>'status' IN ('pending', 'pulling', 'restarting')
		  AND COALESCE(NULLIF(agent_update->>'updated_at', ''), agent_update->>'started_at')::timestamptz
		      < now() - make_interval(secs => $1)
	`, updates.AgentUpdateTimeout.Seconds()); err != nil {
		log.Printf("агенты: зависшие обновления не закрыты: %v", err)
	}
}

func (r *Runner) notifyFailedAgentUpdates(ctx context.Context) {
	rows, err := r.db.Query(ctx, `
		UPDATE core.nodes SET agent_update = agent_update || '{"notified": true}'::jsonb
		WHERE agent_update->>'status' = 'failed'
		  AND NOT COALESCE((agent_update->>'notified')::boolean, false)
		  AND COALESCE(NULLIF(agent_update->>'finished_at', ''), agent_update->>'updated_at')::timestamptz > now() - interval '1 day'
		RETURNING id::text, name, COALESCE(agent_update->>'target', ''), COALESCE(agent_update->>'error', '')
	`)
	if err != nil {
		log.Printf("агенты: сбои обновления не прочитаны: %v", err)
		return
	}
	type failed struct{ id, name, target, reason string }
	var list []failed
	for rows.Next() {
		var f failed
		if rows.Scan(&f.id, &f.name, &f.target, &f.reason) == nil {
			list = append(list, f)
		}
	}
	rows.Close()
	for _, f := range list {
		reason := i18n.Raw(f.reason)
		if f.reason == "" {
			reason = i18n.Key("notify.panel_update_failed.no_reason")
		}
		r.notifyStaff(ctx, notify.Event{
			Kind:  notify.KindAgentUpdateFailed,
			Title: i18n.Key("notify.agent_update_failed.title", i18n.Params{"node": i18n.Raw(f.name)}),
			Body: i18n.Key("notify.agent_update_failed.body", i18n.Params{
				"node": i18n.Raw(f.name), "target": i18n.Raw(f.target), "reason": reason,
			}),
			Action:    r.panelAction("notify.action.agent", "/admin/daemons/"+f.id),
			Meta:      map[string]any{"node_id": f.id, "target": f.target},
			DedupeKey: "agent.update_failed:" + f.id + ":" + f.target + ":" + time.Now().Format("2006-01-02"),
		})
	}
}

func (r *Runner) agentTargetAge(ctx context.Context, target string) time.Duration {
	seen := r.updateSetting(ctx, settingAgentTargetAt)
	version, stamp, _ := strings.Cut(seen, "|")
	if version == target {
		if at, err := time.Parse(time.RFC3339, stamp); err == nil {
			return time.Since(at)
		}
	}
	r.storeUpdateSetting(ctx, settingAgentTargetAt, target+"|"+time.Now().UTC().Format(time.RFC3339))
	return 0
}

func (r *Runner) notifyOutdatedAgents(ctx context.Context) {
	target := buildinfo.Current()
	if !updates.IsSemver(target) || r.agentTargetAge(ctx, target) < agentOutdatedGrace {
		return
	}
	autoAll := r.updateSetting(ctx, updates.SettingAgentsAuto) == "1"
	rows, err := r.db.Query(ctx, `
		SELECT n.name, COALESCE(d.version, ''), n.agent_auto_update, COALESCE(n.agent_update->>'status', '')
		FROM core.nodes n
		JOIN core.node_daemons d ON d.node_id = n.id
		WHERE COALESCE(d.version, '') <> ''
		ORDER BY n.name
	`)
	if err != nil {
		log.Printf("агенты: версии не прочитаны: %v", err)
		return
	}
	var names []string
	for rows.Next() {
		var name, version, status string
		var auto bool
		if rows.Scan(&name, &version, &auto, &status) != nil {
			continue
		}
		version = buildinfo.Normalize(version)
		if !updates.AgentOutdated(version, target) {
			continue
		}
		switch status {
		case "pending", "pulling", "restarting":
			continue
		}
		if autoAll && auto && status != "failed" {
			continue
		}
		names = append(names, name)
	}
	rows.Close()
	if len(names) == 0 {
		return
	}
	shown := names
	if len(shown) > agentOutdatedMaxNames {
		shown = append(append([]string{}, shown[:agentOutdatedMaxNames]...), "…")
	}
	r.notifyStaff(ctx, notify.Event{
		Kind:  notify.KindAgentOutdated,
		Title: i18n.Key("notify.agent_outdated.title", i18n.Params{"count": i18n.Raw(strconv.Itoa(len(names)))}),
		Body: i18n.Key("notify.agent_outdated.body", i18n.Params{
			"target": i18n.Raw(target), "nodes": i18n.Raw(strings.Join(shown, ", ")),
		}),
		Action:    r.panelAction("notify.action.agents", "/admin/daemons?filter=outdated"),
		Meta:      map[string]any{"target": target, "count": len(names)},
		DedupeKey: "agent.outdated:" + target + ":" + time.Now().Format("2006-01-02"),
	})
}
