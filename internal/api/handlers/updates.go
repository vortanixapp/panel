package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/internal/api/jobwake"
	"github.com/vortanixapp/panel/internal/api/relay"
	"github.com/vortanixapp/panel/pkg/buildinfo"
	"github.com/vortanixapp/panel/pkg/protocol"
	"github.com/vortanixapp/panel/pkg/updates"
)

const (
	updateCacheKey   = "updates:latest"
	releasesCacheKey = "updates:releases"
	updateCacheTTL   = time.Hour
	releasesLimit    = 30
)

var errUpdatesDisabled = errors.New("проверка обновлений выключена: не задан UPDATE_REPO")

func agentImageRef() string {
	return updates.AgentImage(buildinfo.Current())
}

func updatesStaff(w http.ResponseWriter, r *http.Request) bool {
	claims, ok := tenantClaims(r.Context())
	if !ok || !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return false
	}
	return true
}

func (h *Handler) latestRelease(ctx context.Context, refresh bool) (*updates.Release, bool, error) {
	repo := updates.Repo()
	if repo == "" {
		return nil, false, errUpdatesDisabled
	}
	var rel updates.Release
	if !refresh {
		if found, err := h.cache.GetJSON(ctx, updateCacheKey, &rel); err == nil && found {
			return &rel, true, nil
		}
	}
	fetched, err := updates.LatestRelease(ctx, repo)
	if err != nil {
		return nil, false, err
	}
	if err := h.cache.SetJSON(ctx, updateCacheKey, fetched, updateCacheTTL); err != nil {
		log.Printf("проверка обновлений: ответ не закеширован: %v", err)
	}
	return fetched, false, nil
}

func (h *Handler) recentReleases(ctx context.Context, refresh bool) ([]updates.Release, error) {
	repo := updates.Repo()
	if repo == "" {
		return nil, errUpdatesDisabled
	}
	var list []updates.Release
	if !refresh {
		if found, err := h.cache.GetJSON(ctx, releasesCacheKey, &list); err == nil && found {
			return list, nil
		}
	}
	fetched, err := updates.Releases(ctx, repo, releasesLimit)
	if err != nil {
		return nil, err
	}
	if err := h.cache.SetJSON(ctx, releasesCacheKey, fetched, updateCacheTTL); err != nil {
		log.Printf("проверка обновлений: список выпусков не закеширован: %v", err)
	}
	return fetched, nil
}

func updaterStatus(ctx context.Context) *updates.Status {
	u := updates.NewUpdater()
	if !u.Configured() {
		return &updates.Status{Mode: "unknown", Reason: updates.ErrUpdaterMissing.Error()}
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	st, err := u.Status(ctx)
	if err != nil {
		return &updates.Status{Mode: "unknown", Reason: err.Error()}
	}
	return st
}

type panelAutoSettings struct {
	Enabled        bool   `json:"enabled"`
	WindowStart    int    `json:"window_start"`
	WindowEnd      int    `json:"window_end"`
	SkippedVersion string `json:"skipped_version,omitempty"`
}

func hourSetting(v string) int {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n < 0 || n > 23 {
		return -1
	}
	return n
}

func (h *Handler) loadPanelAuto(ctx context.Context) panelAutoSettings {
	return panelAutoSettings{
		Enabled:        h.tenantSettingString(ctx, updates.SettingPanelAuto) == "1",
		WindowStart:    hourSetting(h.tenantSettingString(ctx, updates.SettingPanelWindowStart)),
		WindowEnd:      hourSetting(h.tenantSettingString(ctx, updates.SettingPanelWindowEnd)),
		SkippedVersion: h.tenantSettingString(ctx, updates.SettingPanelSkipVersion),
	}
}

func (h *Handler) AdminUpdates(w http.ResponseWriter, r *http.Request) {
	if !updatesStaff(w, r) {
		return
	}
	ctx := r.Context()
	current := buildinfo.Current()
	out := map[string]any{
		"current_version":  current,
		"update_available": false,
		"repo":             updates.Repo(),
		"auto":             h.loadPanelAuto(ctx),
		"updater":          updaterStatus(ctx),
	}
	if updates.Repo() == "" {
		out["checks_disabled"] = true
		writeJSON(w, http.StatusOK, out)
		return
	}
	refresh := r.URL.Query().Get("refresh") == "1"
	rel, cached, err := h.latestRelease(ctx, refresh)
	if err != nil {
		out["error"] = err.Error()
		writeJSON(w, http.StatusOK, out)
		return
	}
	out["latest_version"] = rel.Version
	out["update_available"] = updates.IsNewer(rel.Version, current)
	out["notes"] = rel.Notes
	out["release_url"] = rel.URL
	out["published_at"] = rel.PublishedAt
	out["prerelease"] = rel.Prerelease
	out["checked_at"] = rel.CheckedAt
	out["from_cache"] = cached

	newer := []updates.Release{}
	if list, err := h.recentReleases(ctx, refresh); err == nil {
		for _, item := range list {
			switch {
			case item.Prerelease:
			case updates.IsNewer(item.Version, current):
				if !updates.IsNewer(item.Version, rel.Version) {
					newer = append(newer, item)
				}
			case updates.SameVersion(item.Version, current):
				installed := item
				out["installed_release"] = installed
			}
		}
		out["releases_limited"] = len(list) >= releasesLimit && len(newer) > 0 && out["installed_release"] == nil
	} else {
		log.Printf("проверка обновлений: список выпусков не получен: %v", err)
	}
	out["releases"] = newer
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) AdminPanelUpdateStatus(w http.ResponseWriter, r *http.Request) {
	if !updatesStaff(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"current_version": buildinfo.Current(),
		"updater":         updaterStatus(r.Context()),
	})
}

func (h *Handler) AdminStartPanelUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok || !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	ctx := r.Context()
	var body struct {
		Version string `json:"version"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	version := buildinfo.Normalize(body.Version)
	if version == "" {
		rel, _, err := h.latestRelease(ctx, true)
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		version = rel.Version
	}
	current := buildinfo.Current()
	if updates.IsSemver(current) && !updates.IsNewer(version, current) {
		writeError(w, http.StatusConflict, fmt.Sprintf("Установлена версия %s, выпуск %s не новее", current, version))
		return
	}
	job, err := updates.NewUpdater().Start(ctx, version)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, updates.ErrUpdaterMissing) {
			status = http.StatusServiceUnavailable
		}
		writeError(w, status, err.Error())
		return
	}
	h.setTenantSettingString(ctx, updates.SettingPanelSkipVersion, "")
	audit(ctx, h.dbOf(ctx), claims.UserID, "updates.panel.start", "panel", map[string]any{"from": current, "to": version})
	writeJSON(w, http.StatusAccepted, map[string]any{"job": job})
}

func (h *Handler) AdminUpdateSettings(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok || !isStaffRole(claims.Role) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	ctx := r.Context()
	var body struct {
		AutoEnabled       *bool `json:"auto_enabled"`
		AgentsAutoEnabled *bool `json:"agents_auto_enabled"`
		WindowStart       *int  `json:"window_start"`
		WindowEnd         *int  `json:"window_end"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	for _, v := range []*int{body.WindowStart, body.WindowEnd} {
		if v != nil && (*v < -1 || *v > 23) {
			writeError(w, http.StatusUnprocessableEntity, "Час окна обновления должен быть от 0 до 23")
			return
		}
	}
	if body.AutoEnabled != nil {
		value := "0"
		if *body.AutoEnabled {
			value = "1"
		}
		h.setTenantSettingString(ctx, updates.SettingPanelAuto, value)
	}
	if body.AgentsAutoEnabled != nil {
		value := "0"
		if *body.AgentsAutoEnabled {
			value = "1"
		}
		h.setTenantSettingString(ctx, updates.SettingAgentsAuto, value)
	}
	if body.WindowStart != nil {
		h.setTenantSettingString(ctx, updates.SettingPanelWindowStart, strconv.Itoa(*body.WindowStart))
	}
	if body.WindowEnd != nil {
		h.setTenantSettingString(ctx, updates.SettingPanelWindowEnd, strconv.Itoa(*body.WindowEnd))
	}
	settings := h.loadPanelAuto(ctx)
	audit(ctx, h.dbOf(ctx), claims.UserID, "updates.settings", "panel", map[string]any{
		"auto": settings.Enabled, "window_start": settings.WindowStart, "window_end": settings.WindowEnd,
		"agents_auto": h.tenantSettingString(ctx, updates.SettingAgentsAuto) == "1",
	})
	writeJSON(w, http.StatusOK, settings)
}

func agentUpdateView(raw []byte) map[string]any {
	st := map[string]any{}
	if len(raw) == 0 || json.Unmarshal(raw, &st) != nil || len(st) == 0 {
		return nil
	}
	switch st["status"] {
	case "pending", "pulling", "restarting":
		stamp, _ := st["updated_at"].(string)
		if stamp == "" {
			stamp, _ = st["started_at"].(string)
		}
		if at, err := time.Parse(time.RFC3339Nano, stamp); err == nil && time.Since(at) > updates.AgentUpdateTimeout {
			st["status"] = "failed"
			st["error"] = "агент не сообщил о результате за 15 минут"
		}
	}
	return st
}

func (h *Handler) markAgentUpdate(ctx context.Context, nodeID, status, target, source, method, from, errMsg string) {
	if _, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.nodes SET agent_update = jsonb_build_object(
			'status', $2::text, 'target', $3::text, 'source', $4::text, 'method', $5::text,
			'from', $6::text, 'error', NULLIF($7::text, ''), 'started_at', now(), 'updated_at', now())
			|| CASE WHEN $2::text = 'failed' THEN jsonb_build_object('finished_at', now()) ELSE '{}'::jsonb END
		WHERE id = $1
	`, nodeID, status, target, source, method, from, errMsg); err != nil {
		log.Printf("обновление агента %s: состояние не записано: %v", nodeID, err)
	}
}

func (h *Handler) startAgentUpdate(ctx context.Context, nodeID, target, image, source string) (string, error) {
	var online, ssh bool
	var version string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(d.status = 'online' AND d.last_seen_at > now() - interval '90 seconds', false),
			COALESCE(d.version, ''),
			(COALESCE(n.ssh_host, '') <> '' AND COALESCE(n.ssh_user, '') <> '' AND COALESCE(n.ssh_password_enc, '') <> '')
		FROM core.nodes n
		LEFT JOIN core.node_daemons d ON d.node_id = n.id
		WHERE n.id::text = $1
	`, nodeID).Scan(&online, &version, &ssh)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("локация не найдена")
	}
	if err != nil {
		return "", errors.New("database error")
	}
	version = buildinfo.Normalize(version)

	if online {
		h.markAgentUpdate(ctx, nodeID, "pending", target, source, "relay", version, "")
		cmdCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
		resp, cmdErr := h.relay.CommandSync(cmdCtx, nodeID, relay.CommandRequest{
			Action:  protocol.ActionAgentUpdate,
			Payload: map[string]any{"image": image, "version": target},
		})
		cancel()
		if cmdErr == nil {
			if accepted, _ := resp.Result["accepted"].(bool); accepted {
				return "relay", nil
			}
			cmdErr = errors.New("агент этой версии не умеет обновляться сам: переустановите его командой из раздела «Локации»")
		}
		if !ssh {
			h.markAgentUpdate(ctx, nodeID, "failed", target, source, "relay", version, cmdErr.Error())
			return "", cmdErr
		}
	} else if !ssh {
		return "", errors.New("агент не в сети, а доступ по SSH для локации не настроен")
	}

	params := map[string]any{}
	if updates.IsSemver(target) {
		params["version"] = target
	}
	payload, _ := json.Marshal(map[string]any{"node_id": nodeID, "action": "update", "params": params})
	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.jobs (type, status, payload) VALUES ('daemon_action', 'pending', $1::jsonb)
	`, payload); err != nil {
		h.markAgentUpdate(ctx, nodeID, "failed", target, source, "ssh", version, "задача не поставлена в очередь")
		return "", fmt.Errorf("задача не поставлена в очередь: %w", err)
	}
	h.markAgentUpdate(ctx, nodeID, "pending", target, source, "ssh", version, "")
	jobwake.Notify("daemon_action")
	return "ssh", nil
}
