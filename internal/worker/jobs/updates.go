package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/vortanixapp/panel/internal/worker/relay"
	"github.com/vortanixapp/panel/pkg/buildinfo"
	"github.com/vortanixapp/panel/pkg/protocol"
	"github.com/vortanixapp/panel/pkg/updates"
)

const (
	updatesInterval        = 5 * time.Minute
	releaseRecheckInterval = 30 * time.Minute
	agentRetryAfter        = 6 * time.Hour
	agentUpdatesPerTick    = 5
)

type updateLoopState struct {
	release   *updates.Release
	checkedAt time.Time
}

func (r *Runner) UpdatesLoop(ctx context.Context) {
	state := &updateLoopState{}
	timer := time.NewTimer(time.Minute)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		r.heartbeat.Beat(LoopUpdates)
		r.autoUpdatePanel(ctx, state)
		r.autoUpdateAgents(ctx)
		timer.Reset(updatesInterval)
	}
}

func (r *Runner) updateSetting(ctx context.Context, key string) string {
	var raw []byte
	if r.db.QueryRow(ctx, `SELECT value FROM core.tenant_settings WHERE key = $1`, key).Scan(&raw) != nil {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	return strings.Trim(string(raw), `"`)
}

func (r *Runner) storeUpdateSetting(ctx context.Context, key, value string) {
	b, _ := json.Marshal(value)
	if _, err := r.db.Exec(ctx, `
		INSERT INTO core.tenant_settings (key, value) VALUES ($1, $2::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
	`, key, b); err != nil {
		log.Printf("автообновление: настройка %s не сохранена: %v", key, err)
	}
}

func settingHour(v string) int {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return -1
	}
	return n
}

func (r *Runner) autoUpdatePanel(ctx context.Context, state *updateLoopState) {
	if r.updateSetting(ctx, updates.SettingPanelAuto) != "1" {
		return
	}
	updater := updates.NewUpdater()
	if !updater.Configured() {
		return
	}
	now := time.Now()
	start := settingHour(r.updateSetting(ctx, updates.SettingPanelWindowStart))
	end := settingHour(r.updateSetting(ctx, updates.SettingPanelWindowEnd))
	if !updates.InWindow(now, start, end) {
		return
	}

	if state.release == nil || now.Sub(state.checkedAt) >= releaseRecheckInterval {
		repo := updates.Repo()
		if repo == "" {
			return
		}
		rel, err := updates.LatestRelease(ctx, repo)
		state.checkedAt = now
		if err != nil {
			log.Printf("автообновление панели: %v", err)
			return
		}
		state.release = rel
	}
	rel := state.release
	current := buildinfo.Current()
	if rel.Prerelease || !updates.IsSemver(current) || !updates.IsNewer(rel.Version, current) {
		return
	}
	if r.updateSetting(ctx, updates.SettingPanelSkipVersion) == rel.Version {
		return
	}

	status, err := updater.Status(ctx)
	if err != nil || !status.Available {
		return
	}
	if job := status.Job; job != nil && job.Target == rel.Version {
		switch job.State {
		case "running", "succeeded":
			return
		case "failed":
			r.storeUpdateSetting(ctx, updates.SettingPanelSkipVersion, rel.Version)
			log.Printf("автообновление панели до %s не удалось, повторять не буду: %s", rel.Version, job.Error)
			return
		}
	}
	if job := status.Job; job != nil && job.State == "running" {
		return
	}

	if _, err := updater.Start(ctx, rel.Version); err != nil {
		log.Printf("автообновление панели до %s не запущено: %v", rel.Version, err)
		return
	}
	log.Printf("автообновление панели: %s → %s", current, rel.Version)
}

func (r *Runner) markAgentUpdate(ctx context.Context, nodeID, status, target, from, errMsg string) {
	if _, err := r.db.Exec(ctx, `
		UPDATE core.nodes SET agent_update = jsonb_build_object(
			'status', $2::text, 'target', $3::text, 'source', 'auto', 'method', 'relay',
			'from', $4::text, 'error', NULLIF($5::text, ''), 'started_at', now(), 'updated_at', now())
			|| CASE WHEN $2::text = 'failed' THEN jsonb_build_object('finished_at', now()) ELSE '{}'::jsonb END
		WHERE id = $1
	`, nodeID, status, target, from, errMsg); err != nil {
		log.Printf("автообновление агента %s: состояние не записано: %v", nodeID, err)
	}
}

func (r *Runner) agentUpdateDue(ctx context.Context, nodeID string, raw []byte, target string) bool {
	var st struct {
		Status    string `json:"status"`
		Target    string `json:"target"`
		UpdatedAt string `json:"updated_at"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &st) != nil || st.Status == "" {
		return true
	}
	if buildinfo.Normalize(st.Target) != target {
		return true
	}
	at, err := time.Parse(time.RFC3339Nano, st.UpdatedAt)
	if err != nil {
		return true
	}
	switch st.Status {
	case "pending", "pulling", "restarting":
		if time.Since(at) > updates.AgentUpdateTimeout {
			if _, err := r.db.Exec(ctx, `
				UPDATE core.nodes SET agent_update = agent_update || jsonb_build_object(
					'status', 'failed', 'error', 'агент не сообщил о результате за 15 минут',
					'finished_at', now(), 'updated_at', now())
				WHERE id = $1
			`, nodeID); err != nil {
				log.Printf("автообновление агента %s: состояние не записано: %v", nodeID, err)
			}
		}
		return false
	default:
		return time.Since(at) > agentRetryAfter
	}
}

func (r *Runner) autoUpdateAgents(ctx context.Context) {
	if r.updateSetting(ctx, updates.SettingAgentsAuto) != "1" {
		return
	}
	target := buildinfo.Current()
	if !updates.IsSemver(target) {
		return
	}
	rows, err := r.db.Query(ctx, `
		SELECT n.id::text, COALESCE(d.version, ''), n.agent_update
		FROM core.nodes n
		JOIN core.node_daemons d ON d.node_id = n.id
		WHERE n.agent_auto_update
		  AND d.status = 'online'
		  AND d.last_seen_at > now() - interval '90 seconds'
		ORDER BY n.name
	`)
	if err != nil {
		log.Printf("автообновление агентов: выборка не прошла: %v", err)
		return
	}
	type candidate struct {
		id, version string
		update      []byte
	}
	list := []candidate{}
	for rows.Next() {
		var c candidate
		if rows.Scan(&c.id, &c.version, &c.update) == nil {
			list = append(list, c)
		}
	}
	rows.Close()

	image := updates.AgentImage(target)
	started := 0
	for _, c := range list {
		if started >= agentUpdatesPerTick {
			return
		}
		version := buildinfo.Normalize(c.version)
		if version == "" || !updates.AgentOutdated(version, target) || !r.agentUpdateDue(ctx, c.id, c.update, target) {
			continue
		}
		started++
		r.markAgentUpdate(ctx, c.id, "pending", target, version, "")
		cmdCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		resp, err := r.relay.CommandSync(cmdCtx, c.id, relay.CommandRequest{
			Action:  protocol.ActionAgentUpdate,
			Payload: map[string]any{"image": image, "version": target},
		})
		cancel()
		if err == nil {
			if accepted, _ := resp.Result["accepted"].(bool); accepted {
				log.Printf("автообновление агента %s: %s → %s", c.id, version, target)
				continue
			}
			err = errors.New("агент этой версии не умеет обновляться сам: переустановите его командой из раздела «Локации»")
		}
		r.markAgentUpdate(ctx, c.id, "failed", target, version, err.Error())
	}
}
