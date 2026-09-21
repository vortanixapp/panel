package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/internal/api/relay"
	"github.com/vortanixapp/panel/pkg/buildinfo"
	"github.com/vortanixapp/panel/pkg/dockerapi"
	"github.com/vortanixapp/panel/pkg/gamecatalog"
	"github.com/vortanixapp/panel/pkg/protocol"
	"github.com/vortanixapp/panel/pkg/updates"
)

var nodeTaskDeadlines = map[string]time.Duration{
	protocol.ActionDiskUsage:      35 * time.Minute,
	protocol.ActionCleanupPreview: 6 * time.Minute,
	protocol.ActionCleanupApply:   35 * time.Minute,
	protocol.ActionDiagnostics:    6 * time.Minute,
	protocol.ActionAgentRestart:   3 * time.Minute,
}

type agentLink struct {
	Online   bool
	Proto    int
	Caps     map[string]bool
	Version  string
	RTTms    *int
	LastSeen *time.Time
}

func (l agentLink) can(capability string) bool {
	return l.Online && l.Proto >= protocol.ProtoVersion && l.Caps[capability]
}

func (h *Handler) loadAgentLink(ctx context.Context, nodeID string) (agentLink, error) {
	var l agentLink
	var status string
	var caps []string
	err := h.readerOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(d.status, ''), d.last_seen_at, COALESCE(d.proto, 1), COALESCE(d.caps, '{}'),
			COALESCE(d.version, ''), d.rtt_ms
		FROM core.nodes n
		LEFT JOIN core.node_daemons d ON d.node_id = n.id
		WHERE n.id::text = $1
	`, nodeID).Scan(&status, &l.LastSeen, &l.Proto, &caps, &l.Version, &l.RTTms)
	if err != nil {
		return l, err
	}
	l.Online = agentDaemonOnline(status, l.LastSeen)
	l.Caps = map[string]bool{}
	for _, c := range caps {
		l.Caps[c] = true
	}
	l.Version = buildinfo.Normalize(l.Version)
	return l, nil
}

func (h *Handler) agentLinkOr404(w http.ResponseWriter, r *http.Request) (string, agentLink, bool) {
	id := chi.URLParam(r, "id")
	l, err := h.loadAgentLink(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "location not found")
		return id, l, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return id, l, false
	}
	return id, l, true
}

func requireAgentCap(w http.ResponseWriter, l agentLink, capability string) bool {
	switch {
	case !l.Online:
		writeCodedError(w, http.StatusConflict, protocol.CodeNodeOffline, "агент не в сети")
	case !l.can(capability):
		writeCodedError(w, http.StatusConflict, protocol.CodeAgentTooOld, "агент на ноде устарел и не умеет эту операцию — обновите его")
	default:
		return true
	}
	return false
}

func writeRelayError(w http.ResponseWriter, err error) {
	code := relay.ErrorCode(err)
	msg := err.Error()
	var re *relay.Error
	if errors.As(err, &re) {
		msg = re.Message
	}
	switch code {
	case protocol.CodeAgentTooOld, protocol.CodeNodeOffline, protocol.CodeRestartPolicy:
		writeCodedError(w, http.StatusConflict, code, msg)
	case protocol.CodeAgentBusy:
		writeCodedError(w, http.StatusTooManyRequests, code, msg)
	case protocol.CodeTaskTimeout:
		writeCodedError(w, http.StatusGatewayTimeout, code, msg)
	case "":
		writeCodedError(w, http.StatusBadGateway, "relay_error", msg)
	default:
		writeCodedError(w, http.StatusBadGateway, code, msg)
	}
}

func (h *Handler) startNodeTask(ctx context.Context, nodeID, action string, payload map[string]any, userID string) (string, error) {
	id := uuid.NewString()
	params, _ := json.Marshal(payload)
	deadline := nodeTaskDeadlines[action]
	if deadline == 0 {
		deadline = 10 * time.Minute
	}
	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.node_tasks (id, node_id, action, method, status, params, requested_by, deadline_at)
		VALUES ($1::uuid, $2::uuid, $3, 'relay', 'queued', $4::jsonb, NULLIF($5, '')::uuid, now() + $6 * interval '1 second')
	`, id, nodeID, action, string(params), userID, int(deadline.Seconds())); err != nil {
		return "", err
	}
	boot, err := h.relay.SendTask(ctx, nodeID, relay.CommandRequest{CommandID: id, Action: action, Payload: payload})
	if err != nil {
		_, _ = h.dbOf(ctx).Exec(ctx, `
			UPDATE core.node_tasks
			SET status = 'failed', error = $2, error_code = NULLIF($3, ''), finished_at = now(), updated_at = now()
			WHERE id = $1::uuid AND status = 'queued'
		`, id, err.Error(), relay.ErrorCode(err))
		return id, err
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.node_tasks
		SET status = CASE WHEN status = 'queued' THEN 'sent' ELSE status END,
			boot_id = COALESCE(boot_id, NULLIF($2, '')), updated_at = now()
		WHERE id = $1::uuid
	`, id, boot)
	return id, nil
}

const nodeTaskColumns = `id::text, action, method, status, params, progress, result, COALESCE(error, ''),
	COALESCE(error_code, ''), created_at, updated_at, finished_at, deadline_at`

func scanNodeTask(row pgx.Row) (map[string]any, error) {
	var id, action, method, status, errMsg, code string
	var params, progress, result []byte
	var created, updated, deadline time.Time
	var finished *time.Time
	if err := row.Scan(&id, &action, &method, &status, &params, &progress, &result, &errMsg, &code,
		&created, &updated, &finished, &deadline); err != nil {
		return nil, err
	}
	task := map[string]any{
		"id": id, "action": action, "method": method, "status": status,
		"created_at": created.UTC().Format(time.RFC3339), "updated_at": updated.UTC().Format(time.RFC3339),
		"deadline_at": deadline.UTC().Format(time.RFC3339),
	}
	for key, raw := range map[string][]byte{"params": params, "progress": progress, "result": result} {
		if len(raw) > 0 {
			task[key] = json.RawMessage(raw)
		}
	}
	if errMsg != "" {
		task["error"] = errMsg
	}
	if code != "" {
		task["error_code"] = code
	}
	if finished != nil {
		task["finished_at"] = finished.UTC().Format(time.RFC3339)
	}
	return task, nil
}

func (h *Handler) latestNodeTasks(ctx context.Context, nodeID, action string) map[string]any {
	out := map[string]any{"running": nil, "last": nil}
	if t, err := scanNodeTask(h.readerOf(ctx).QueryRow(ctx, `
		SELECT `+nodeTaskColumns+` FROM core.node_tasks
		WHERE node_id::text = $1 AND action = $2 AND status IN ('queued', 'sent', 'running')
		ORDER BY created_at DESC LIMIT 1
	`, nodeID, action)); err == nil {
		out["running"] = t
	}
	if t, err := scanNodeTask(h.readerOf(ctx).QueryRow(ctx, `
		SELECT `+nodeTaskColumns+` FROM core.node_tasks
		WHERE node_id::text = $1 AND action = $2 AND status IN ('done', 'failed', 'expired')
		ORDER BY created_at DESC LIMIT 1
	`, nodeID, action)); err == nil {
		out["last"] = t
	}
	return out
}

func (h *Handler) GetAdminAgentTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	task, err := scanNodeTask(h.readerOf(r.Context()).QueryRow(r.Context(), `
		SELECT `+nodeTaskColumns+` FROM core.node_tasks WHERE node_id::text = $1 AND id::text = $2
	`, id, chi.URLParam(r, "taskId")))
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": task})
}

func (h *Handler) nodeServerIDs(ctx context.Context, nodeID string) []string {
	ids := []string{}
	rows, err := h.readerOf(ctx).Query(ctx, `SELECT id::text FROM core.servers WHERE node_id::text = $1`, nodeID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	if rows.Err() != nil {
		return nil
	}
	return ids
}

func (h *Handler) gameRepositories(ctx context.Context) []string {
	seen := map[string]bool{}
	var out []string
	add := func(repo string) {
		if repo = strings.TrimSpace(repo); repo != "" && !seen[repo] {
			seen[repo] = true
			out = append(out, repo)
		}
	}
	for _, g := range gamecatalog.All() {
		add(gamecatalog.Repository(g.Key))
	}
	rows, err := h.readerOf(ctx).Query(ctx, `SELECT DISTINCT docker_image FROM core.game_versions WHERE COALESCE(docker_image, '') <> ''`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var img string
			if rows.Scan(&img) == nil {
				repo, _ := dockerapi.SplitRef(img)
				add(repo)
			}
		}
	}
	return out
}

func (h *Handler) startAgentTaskHTTP(w http.ResponseWriter, r *http.Request, action string, payload map[string]any, auditAction string) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	nodeID, link, ok := h.agentLinkOr404(w, r)
	if !ok || !requireAgentCap(w, link, protocol.RequiredCap(action)) {
		return
	}
	var running string
	_ = h.readerOf(r.Context()).QueryRow(r.Context(), `
		SELECT id::text FROM core.node_tasks
		WHERE node_id::text = $1 AND action = $2 AND status IN ('queued', 'sent', 'running') AND deadline_at > now()
		ORDER BY created_at DESC LIMIT 1
	`, nodeID, action).Scan(&running)
	if running != "" && action != protocol.ActionCleanupApply {
		writeJSON(w, http.StatusAccepted, map[string]any{"task_id": running, "already_running": true})
		return
	}
	taskID, err := h.startNodeTask(r.Context(), nodeID, action, payload, claims.UserID)
	if err != nil {
		writeRelayError(w, err)
		return
	}
	if auditAction != "" {
		audit(r.Context(), h.dbOf(r.Context()), claims.UserID, auditAction, "node:"+nodeID, map[string]any{"task_id": taskID, "params": payload})
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"task_id": taskID})
}

func (h *Handler) PostAdminAgentDisk(w http.ResponseWriter, r *http.Request) {
	payload := map[string]any{}
	if ids := h.nodeServerIDs(r.Context(), chi.URLParam(r, "id")); ids != nil {
		payload["known_servers"] = ids
	}
	h.startAgentTaskHTTP(w, r, protocol.ActionDiskUsage, payload, "")
}

func (h *Handler) GetAdminAgentDisk(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.latestNodeTasks(r.Context(), chi.URLParam(r, "id"), protocol.ActionDiskUsage))
}

func (h *Handler) cleanupPayload(ctx context.Context, nodeID string) (map[string]any, bool) {
	ids := h.nodeServerIDs(ctx, nodeID)
	if ids == nil {
		return nil, false
	}
	return map[string]any{"known_servers": ids, "game_repos": h.gameRepositories(ctx)}, true
}

func (h *Handler) PostAdminAgentCleanupPreview(w http.ResponseWriter, r *http.Request) {
	payload, ok := h.cleanupPayload(r.Context(), chi.URLParam(r, "id"))
	if !ok {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	h.startAgentTaskHTTP(w, r, protocol.ActionCleanupPreview, payload, "")
}

func (h *Handler) PostAdminAgentCleanupApply(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Items []string `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	items := make([]string, 0, len(body.Items))
	for _, it := range body.Items {
		if it = strings.TrimSpace(it); it != "" && len(it) <= 200 {
			items = append(items, it)
		}
	}
	if len(items) == 0 || len(items) > 2000 {
		writeError(w, http.StatusUnprocessableEntity, "Не выбрано, что удалить")
		return
	}
	payload, ok := h.cleanupPayload(r.Context(), chi.URLParam(r, "id"))
	if !ok {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	payload["items"] = items
	h.startAgentTaskHTTP(w, r, protocol.ActionCleanupApply, payload, "agent.cleanup")
}

func (h *Handler) GetAdminAgentCleanup(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	writeJSON(w, http.StatusOK, map[string]any{
		"preview": h.latestNodeTasks(r.Context(), id, protocol.ActionCleanupPreview),
		"apply":   h.latestNodeTasks(r.Context(), id, protocol.ActionCleanupApply),
	})
}

func (h *Handler) PostAdminAgentDiagnostics(w http.ResponseWriter, r *http.Request) {
	h.startAgentTaskHTTP(w, r, protocol.ActionDiagnostics, map[string]any{
		"target_image": updates.AgentImage(buildinfo.Current()),
	}, "")
}

func (h *Handler) GetAdminAgentDiagnostics(w http.ResponseWriter, r *http.Request) {
	nodeID, link, ok := h.agentLinkOr404(w, r)
	if !ok {
		return
	}
	out := h.latestNodeTasks(r.Context(), nodeID, protocol.ActionDiagnostics)
	out["panel_checks"] = panelAgentChecks(link)
	writeJSON(w, http.StatusOK, out)
}

func panelAgentChecks(l agentLink) []map[string]any {
	check := func(id, status string, data map[string]any) map[string]any {
		return map[string]any{"id": id, "status": status, "data": data}
	}
	var checks []map[string]any
	if l.LastSeen == nil {
		checks = append(checks, check("heartbeat", "fail", map[string]any{}))
	} else {
		age := time.Since(*l.LastSeen).Seconds()
		st := "ok"
		switch {
		case !l.Online:
			st = "fail"
		case age > 60:
			st = "warn"
		}
		checks = append(checks, check("heartbeat", st, map[string]any{"age_sec": int(age)}))
	}
	if l.RTTms != nil {
		st := "ok"
		switch {
		case *l.RTTms > 1000:
			st = "fail"
		case *l.RTTms > 300:
			st = "warn"
		}
		checks = append(checks, check("relay_rtt", st, map[string]any{"rtt_ms": *l.RTTms}))
	} else {
		checks = append(checks, check("relay_rtt", "skip", map[string]any{}))
	}
	target := buildinfo.Current()
	st := "ok"
	if l.Version == "" {
		st = "skip"
	} else if updates.AgentOutdated(l.Version, target) {
		st = "warn"
	}
	checks = append(checks, check("agent_version", st, map[string]any{"version": l.Version, "target": target}))
	st = "ok"
	if l.Proto < protocol.ProtoVersion {
		st = "warn"
	}
	checks = append(checks, check("agent_protocol", st, map[string]any{"proto": l.Proto, "expected": protocol.ProtoVersion}))
	return checks
}

func (h *Handler) agentSync(ctx context.Context, nodeID, action string, payload map[string]any) (map[string]any, error) {
	if payload == nil {
		payload = map[string]any{}
	}
	cmdCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	resp, err := h.relay.CommandSync(cmdCtx, nodeID, relay.CommandRequest{
		CommandID: uuid.NewString(), Action: action, Payload: payload,
	})
	if err != nil {
		return nil, err
	}
	if resp.Result[protocol.ResultVersionKey] == nil {
		return nil, &relay.Error{Status: http.StatusConflict, Code: protocol.CodeAgentTooOld, Message: "агент ответил в старом формате — обновите его"}
	}
	return resp.Result, nil
}

func (h *Handler) GetAdminAgentInfo(w http.ResponseWriter, r *http.Request) {
	nodeID, link, ok := h.agentLinkOr404(w, r)
	if !ok {
		return
	}
	if link.can(protocol.CapNodeInfo) {
		if res, err := h.agentSync(r.Context(), nodeID, protocol.ActionNodeInfo, nil); err == nil {
			res["source"] = "agent"
			writeJSON(w, http.StatusOK, res)
			return
		}
	}
	var host []byte
	var stats []byte
	_ = h.readerOf(r.Context()).QueryRow(r.Context(), `
		SELECT COALESCE(host, '{}'::jsonb), COALESCE(stats, '{}'::jsonb) FROM core.node_daemons WHERE node_id::text = $1
	`, nodeID).Scan(&host, &stats)
	out := map[string]any{"source": "snapshot"}
	if len(host) > 0 {
		out["host"] = json.RawMessage(host)
	}
	if len(stats) > 0 {
		out["stats"] = json.RawMessage(stats)
	}
	writeJSON(w, http.StatusOK, out)
}

type panelServer struct {
	ID, Name, Status, Runtime, Owner string
}

func (h *Handler) nodePanelServers(ctx context.Context, nodeID string) (map[string]panelServer, []string, error) {
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT s.id::text, s.name, COALESCE(s.status, ''), COALESCE(s.runtime_status, ''), COALESCE(u.email, '')
		FROM core.servers s
		LEFT JOIN core.users u ON u.id = s.user_id
		WHERE s.node_id::text = $1
		ORDER BY s.created_at
	`, nodeID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	byID := map[string]panelServer{}
	var order []string
	for rows.Next() {
		var s panelServer
		if rows.Scan(&s.ID, &s.Name, &s.Status, &s.Runtime, &s.Owner) == nil {
			byID[s.ID] = s
			order = append(order, s.ID)
		}
	}
	return byID, order, rows.Err()
}

func serverStatusDiffers(panelStatus, state string) bool {
	running := state == "running" || state == "restarting"
	switch panelStatus {
	case "running", "starting":
		return !running
	case "stopped", "offline", "error":
		return running
	}
	return false
}

func (h *Handler) GetAdminAgentContainers(w http.ResponseWriter, r *http.Request) {
	nodeID, link, ok := h.agentLinkOr404(w, r)
	if !ok {
		return
	}
	servers, order, err := h.nodePanelServers(r.Context(), nodeID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	serverJSON := func(s panelServer) map[string]any {
		return map[string]any{"id": s.ID, "name": s.Name, "status": s.Status, "runtime_status": s.Runtime, "owner": s.Owner}
	}
	resp := map[string]any{"limited": true, "containers": []any{}, "missing": []any{}}
	var agentErr error
	if link.can(protocol.CapContainers) {
		var res map[string]any
		res, agentErr = h.agentSync(r.Context(), nodeID, protocol.ActionContainersList, nil)
		if agentErr == nil {
			raw, _ := json.Marshal(res["containers"])
			var list []map[string]any
			_ = json.Unmarshal(raw, &list)
			seen := map[string]bool{}
			for _, c := range list {
				sid, _ := c["server_id"].(string)
				if kind, _ := c["kind"].(string); kind == "server" && sid != "" {
					seen[sid] = true
					if s, ok := servers[sid]; ok {
						c["server"] = serverJSON(s)
						state, _ := c["state"].(string)
						c["status_mismatch"] = serverStatusDiffers(s.Status, state)
					} else {
						c["abandoned"] = true
					}
				}
			}
			missing := []any{}
			for _, id := range order {
				if !seen[id] {
					missing = append(missing, serverJSON(servers[id]))
				}
			}
			resp = map[string]any{"limited": false, "containers": list, "missing": missing}
		}
	}
	if resp["limited"] == true {
		list := []any{}
		for _, id := range order {
			list = append(list, serverJSON(servers[id]))
		}
		resp["servers"] = list
		switch {
		case agentErr != nil:
			resp["reason"] = relay.ErrorCode(agentErr)
			resp["error"] = agentErr.Error()
		case !link.Online:
			resp["reason"] = protocol.CodeNodeOffline
		default:
			resp["reason"] = protocol.CodeAgentTooOld
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) PostAdminAgentRestart(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	nodeID, link, ok := h.agentLinkOr404(w, r)
	if !ok {
		return
	}
	var body struct {
		Force bool `json:"force"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if link.can(protocol.CapAgentRestart) {
		taskID, err := h.startNodeTask(r.Context(), nodeID, protocol.ActionAgentRestart, map[string]any{"force": body.Force}, claims.UserID)
		if err != nil {
			writeRelayError(w, err)
			return
		}
		audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "agent.restart", "node:"+nodeID, map[string]any{"method": "relay", "force": body.Force})
		writeJSON(w, http.StatusAccepted, map[string]any{"task_id": taskID, "method": "relay"})
		return
	}
	loc, err := h.loadLocationRow(r, nodeID)
	if err != nil || !hasSSHConfigured(loc, parseMetaMap(loc.Meta)) {
		writeCodedError(w, http.StatusConflict, "ssh_not_configured",
			"агент не может перезапуститься сам, а SSH для ноды не настроен")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "agent.restart", "node:"+nodeID, map[string]any{"method": "ssh"})
	h.enqueueDaemonJob(w, r, "restart", nil)
}

func (h *Handler) agentLogsViaRelay(r *http.Request, nodeID string) (map[string]any, error) {
	q := r.URL.Query()
	payload := map[string]any{}
	if v := q.Get("cursor"); v != "" {
		payload["since"] = v
	}
	if v := q.Get("before"); v != "" {
		payload["until"] = v
	}
	if n, err := strconv.Atoi(q.Get("tail")); err == nil && n > 0 {
		payload["tail"] = n
	}
	res, err := h.agentSync(r.Context(), nodeID, protocol.ActionAgentLogs, payload)
	if err != nil {
		return nil, err
	}
	res["source"] = "relay"
	var texts []string
	if lines, ok := res["lines"].([]any); ok {
		for _, l := range lines {
			if m, ok := l.(map[string]any); ok {
				t, _ := m["text"].(string)
				texts = append(texts, t)
			}
		}
	}
	res["logs"] = texts
	res["container_found"] = true
	delete(res, protocol.ResultVersionKey)
	return res, nil
}
