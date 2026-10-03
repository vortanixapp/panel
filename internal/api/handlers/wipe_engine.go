package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/pkg/gamesettings"
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
	"github.com/vortanixapp/panel/pkg/settingsreg"
	"github.com/vortanixapp/panel/pkg/wipe"
)

const (
	lockWipeScheduler int64 = 7410010001
	lockTelegramBot   int64 = 7410010002

	wipeTickEvery      = 20 * time.Second
	wipeMissedAfter    = 10 * time.Minute
	wipeStaleAfter     = 10 * time.Minute
	wipeLateAnnounce   = 90 * time.Second
	wipeHistoryLimit   = 20
	wipeSeedMax        = 2147483646
	wipeBackupNameTime = "20060102-1504"
)

var (
	wipeActive sync.Map

	errWipeBusy   = errors.New("на сервере уже идёт вайп")
	errWipeNoPlan = errors.New("вайп для этой игры недоступен")
)

type wipeRunRow struct {
	ID              string
	PlanID          string
	ServerID        string
	Kind            string
	Source          string
	CreatedBy       string
	StartsAt        time.Time
	Announced       []int32
	AnnounceMinutes []int32
	BackupBefore    bool
	NewSeed         bool
	NotifyOwner     bool
	AnnounceText    string
}

const wipeRunColumns = `r.id::text, COALESCE(r.plan_id::text, ''), r.server_id::text, r.kind, r.source,
	COALESCE(r.created_by::text, ''), r.starts_at, r.announced, r.announce_minutes,
	r.backup_before, r.new_seed, r.notify_owner, r.announce_text`

func scanWipeRun(row pgx.Row) (wipeRunRow, error) {
	var r wipeRunRow
	err := row.Scan(&r.ID, &r.PlanID, &r.ServerID, &r.Kind, &r.Source, &r.CreatedBy, &r.StartsAt,
		&r.Announced, &r.AnnounceMinutes, &r.BackupBefore, &r.NewSeed, &r.NotifyOwner, &r.AnnounceText)
	return r, err
}

func ints32(in []int) []int32 {
	out := make([]int32, 0, len(in))
	for _, v := range in {
		out = append(out, int32(v))
	}
	return out
}

func ints(in []int32) []int {
	out := make([]int, 0, len(in))
	for _, v := range in {
		out = append(out, int(v))
	}
	return out
}

func (h *Handler) leaderLoop(ctx context.Context, key int64, fn func(ctx context.Context)) {
	for ctx.Err() == nil {
		conn, err := h.dbOf(ctx).Acquire(ctx)
		if err != nil {
			sleepCtx(ctx, 15*time.Second)
			continue
		}
		var locked bool
		if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, key).Scan(&locked); err != nil || !locked {
			conn.Release()
			sleepCtx(ctx, 15*time.Second)
			continue
		}
		runCtx, cancel := context.WithCancel(ctx)
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-runCtx.Done():
					return
				case <-ticker.C:
					if conn.Conn().Ping(runCtx) != nil {
						cancel()
						return
					}
				}
			}
		}()
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("фоновая задача %d аварийно завершилась: %v", key, r)
				}
			}()
			fn(runCtx)
		}()
		cancel()
		_, _ = conn.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, key)
		conn.Release()
		sleepCtx(ctx, 5*time.Second)
	}
}

func sleepCtx(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

func (h *Handler) StartWipeScheduler(ctx context.Context) {
	go h.leaderLoop(ctx, lockWipeScheduler, func(ctx context.Context) {
		ticker := time.NewTicker(wipeTickEvery)
		defer ticker.Stop()
		for {
			h.wipeTick(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	})
}

func (h *Handler) wipeTick(ctx context.Context) {
	if h.freezeActive(ctx) {
		return
	}
	h.wipeReapStale(ctx)
	h.wipeSchedulePlans(ctx)
	h.wipeAnnounce(ctx)
	h.wipeStartDue(ctx)
}

type wipePlanRow struct {
	ID              string
	ServerID        string
	Name            string
	Kind            string
	Enabled         bool
	Schedule        wipe.Schedule
	BackupBefore    bool
	NewSeed         bool
	AnnounceMinutes []int32
	AnnounceText    string
	NotifyOwner     bool
	SkipNext        bool
	NextRunAt       *time.Time
	LastRunAt       *time.Time
	LastStatus      string
	CreatedBy       string
}

const wipePlanColumns = `p.id::text, p.server_id::text, p.name, p.kind, p.enabled, p.schedule_type, p.weekday, p.nth,
	p.time_of_day, p.cron_expr, p.run_at, p.timezone, p.backup_before, p.new_seed, p.announce_minutes,
	p.announce_text, p.notify_owner, p.skip_next, p.next_run_at, p.last_run_at, p.last_status,
	COALESCE(p.created_by::text, '')`

func scanWipePlan(row pgx.Row) (wipePlanRow, error) {
	var p wipePlanRow
	var weekday, nth int16
	var runAt *time.Time
	err := row.Scan(&p.ID, &p.ServerID, &p.Name, &p.Kind, &p.Enabled, &p.Schedule.Type, &weekday, &nth,
		&p.Schedule.TimeOfDay, &p.Schedule.Cron, &runAt, &p.Schedule.TZ, &p.BackupBefore, &p.NewSeed,
		&p.AnnounceMinutes, &p.AnnounceText, &p.NotifyOwner, &p.SkipNext, &p.NextRunAt, &p.LastRunAt,
		&p.LastStatus, &p.CreatedBy)
	if err != nil {
		return p, err
	}
	p.Schedule.Weekday = int(weekday)
	p.Schedule.Nth = int(nth)
	if runAt != nil {
		p.Schedule.RunAt = *runAt
	}
	return p, nil
}

func (h *Handler) wipeSchedulePlans(ctx context.Context) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT `+wipePlanColumns+`
		FROM core.server_wipe_plans p
		WHERE p.enabled AND p.next_run_at IS NOT NULL
		  AND p.next_run_at <= now() + COALESCE(
		        (SELECT max(m) FROM unnest(p.announce_minutes) AS m), 0) * interval '1 minute'
		  AND NOT EXISTS (
		        SELECT 1 FROM core.server_wipe_runs r
		        WHERE r.server_id = p.server_id AND r.status IN ('announcing', 'running'))
		ORDER BY p.next_run_at
		LIMIT 50
	`)
	if err != nil {
		log.Printf("вайпы: выбор планов: %v", err)
		return
	}
	var due []wipePlanRow
	for rows.Next() {
		if p, err := scanWipePlan(rows); err == nil {
			due = append(due, p)
		}
	}
	rows.Close()

	for _, p := range due {
		h.wipeAdvancePlan(ctx, p)
	}
}

func (h *Handler) wipeAdvancePlan(ctx context.Context, p wipePlanRow) {
	startsAt := *p.NextRunAt
	next, hasNext := p.Schedule.Next(startsAt)
	var nextArg any
	if hasNext {
		nextArg = next
	}
	finishOnce := p.Schedule.Type == wipe.ScheduleOnce

	if p.SkipNext {
		_, _ = h.dbOf(ctx).Exec(ctx, `
			UPDATE core.server_wipe_plans
			SET skip_next = false, next_run_at = $2, enabled = enabled AND NOT $3, last_status = 'skipped', updated_at = now()
			WHERE id = $1
		`, p.ID, nextArg, finishOnce)
		return
	}
	if time.Since(startsAt) > wipeMissedAfter {
		_, _ = h.dbOf(ctx).Exec(ctx, `
			UPDATE core.server_wipe_plans
			SET next_run_at = $2, enabled = enabled AND NOT $3, last_status = 'missed', updated_at = now()
			WHERE id = $1
		`, p.ID, nextArg, finishOnce)
		return
	}

	tag, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.server_wipe_runs
			(plan_id, server_id, kind, source, status, starts_at, backup_before, new_seed,
			 announce_minutes, announce_text, notify_owner, created_by)
		SELECT $1::uuid, p.server_id, p.kind, 'schedule', 'announcing', $2, p.backup_before, p.new_seed,
		       p.announce_minutes, p.announce_text, p.notify_owner, p.created_by
		FROM core.server_wipe_plans p
		WHERE p.id = $1
		ON CONFLICT (server_id) WHERE status IN ('announcing', 'running') DO NOTHING
	`, p.ID, startsAt)
	if err != nil {
		log.Printf("вайпы: создание запуска для плана %s: %v", p.ID, err)
		return
	}
	if tag.RowsAffected() == 0 {
		return
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.server_wipe_plans
		SET next_run_at = $2, enabled = enabled AND NOT $3, last_status = 'scheduled', updated_at = now()
		WHERE id = $1
	`, p.ID, nextArg, finishOnce)
}

func (h *Handler) wipeAnnounce(ctx context.Context) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT `+wipeRunColumns+`
		FROM core.server_wipe_runs r
		WHERE r.status = 'announcing' AND r.starts_at > now()
		  AND cardinality(r.announce_minutes) > 0
	`)
	if err != nil {
		return
	}
	var runs []wipeRunRow
	for rows.Next() {
		if r, err := scanWipeRun(rows); err == nil {
			runs = append(runs, r)
		}
	}
	rows.Close()

	for _, run := range runs {
		for _, m := range run.AnnounceMinutes {
			if containsInt32(run.Announced, m) {
				continue
			}
			at := run.StartsAt.Add(-time.Duration(m) * time.Minute)
			now := time.Now()
			if now.Before(at) {
				continue
			}
			_, _ = h.dbOf(ctx).Exec(ctx, `
				UPDATE core.server_wipe_runs SET announced = array_append(announced, $2::int)
				WHERE id = $1 AND status = 'announcing'
			`, run.ID, m)
			if now.Sub(at) > wipeLateAnnounce {
				continue
			}
			h.wipeSay(ctx, run, int(m))
		}
	}
}

func containsInt32(list []int32, v int32) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func (h *Handler) wipeServerBasics(ctx context.Context, serverID string) (nodeID, gameID, name, owner string, err error) {
	err = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(node_id::text, ''), COALESCE(game_id, ''), name, COALESCE(user_id::text, '')
		FROM core.servers WHERE id = $1
	`, serverID).Scan(&nodeID, &gameID, &name, &owner)
	return
}

func (h *Handler) wipeSay(ctx context.Context, run wipeRunRow, minutes int) {
	_, gameID, _, owner, err := h.wipeServerBasics(ctx, run.ServerID)
	if err != nil {
		return
	}
	recipe, ok := wipe.RecipeFor(gameID)
	if !ok || recipe.AnnounceTool == "" || !h.serverRunning(ctx, run.ServerID) {
		return
	}
	l := i18n.ForUser(ctx, h.dbOf(ctx), owner)
	text := wipe.RenderAnnouncement(run.AnnounceText, minutes, l.T("notify.wipe_say.default"))
	say, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if _, err := h.consoleTool(say, run.ServerID, gameID, recipe.AnnounceTool, map[string]string{"text": text}); err != nil {
		log.Printf("вайпы: объявление на сервере %s не отправлено: %v", run.ServerID, err)
	}
}

func (h *Handler) wipeStartDue(ctx context.Context) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		UPDATE core.server_wipe_runs r
		SET status = 'running', stage = 'start', started_at = now(), heartbeat_at = now()
		WHERE r.id IN (
			SELECT id FROM core.server_wipe_runs
			WHERE status = 'announcing' AND starts_at <= now()
			ORDER BY starts_at
			LIMIT 5
			FOR UPDATE SKIP LOCKED
		)
		RETURNING `+wipeRunColumns)
	if err != nil {
		log.Printf("вайпы: запуск: %v", err)
		return
	}
	var runs []wipeRunRow
	for rows.Next() {
		if r, err := scanWipeRun(rows); err == nil {
			runs = append(runs, r)
		}
	}
	rows.Close()

	for _, run := range runs {
		wipeActive.Store(run.ID, true)
		go func(run wipeRunRow) {
			defer wipeActive.Delete(run.ID)
			defer func() {
				if r := recover(); r != nil {
					h.wipeFail(context.Background(), run, "panic", fmt.Errorf("внутренняя ошибка: %v", r), false)
				}
			}()
			runCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), settingsreg.ServersWipeRunTimeout.Duration())
			defer cancel()
			h.executeWipe(runCtx, run)
		}(run)
	}
}

func (h *Handler) wipeReapStale(ctx context.Context) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT `+wipeRunColumns+`
		FROM core.server_wipe_runs r
		WHERE r.status = 'running' AND r.heartbeat_at < now() - $1::interval
	`, wipeStaleAfter.String())
	if err != nil {
		return
	}
	var runs []wipeRunRow
	for rows.Next() {
		if r, err := scanWipeRun(rows); err == nil {
			runs = append(runs, r)
		}
	}
	rows.Close()
	for _, run := range runs {
		if _, busy := wipeActive.Load(run.ID); busy {
			continue
		}
		h.wipeFail(ctx, run, "interrupted", errors.New("вайп прерван перезапуском панели"), false)
	}
}

func (h *Handler) wipeSetStage(ctx context.Context, runID, stage string) {
	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.server_wipe_runs SET stage = $2, heartbeat_at = now() WHERE id = $1
	`, runID, stage)
}

func (h *Handler) wipeSetDetail(ctx context.Context, runID, key string, value any) {
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.server_wipe_runs
		SET details = jsonb_set(details, ARRAY[$2]::text[], $3::jsonb, true), heartbeat_at = now()
		WHERE id = $1
	`, runID, key, string(raw))
}

func (h *Handler) wipeKeepAlive(ctx context.Context, runID string) func() {
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = h.dbOf(ctx).Exec(ctx, `UPDATE core.server_wipe_runs SET heartbeat_at = now() WHERE id = $1`, runID)
			}
		}
	}()
	return func() { close(done) }
}

func (h *Handler) wipeFail(ctx context.Context, run wipeRunRow, stage string, cause error, serverUntouched bool) {
	reason := cause.Error()
	if r := []rune(reason); len(r) > 500 {
		reason = string(r[:500])
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.server_wipe_runs
		SET status = 'failed', stage = $2, error = $3, finished_at = now()
		WHERE id = $1 AND status IN ('announcing', 'running')
	`, run.ID, stage, reason)
	if run.PlanID != "" {
		_, _ = h.dbOf(ctx).Exec(ctx, `
			UPDATE core.server_wipe_plans SET last_run_at = now(), last_status = 'failed', updated_at = now() WHERE id = $1
		`, run.PlanID)
	}
	audit(ctx, h.dbOf(ctx), run.CreatedBy, "server.wipe_failed", "server:"+run.ServerID, map[string]any{
		"kind": run.Kind, "stage": stage, "error": reason,
	})
	if !run.NotifyOwner {
		return
	}
	_, _, name, _, err := h.wipeServerBasics(ctx, run.ServerID)
	if err != nil {
		return
	}
	bodyKey := "notify.wipe_failed.body"
	if serverUntouched {
		bodyKey = "notify.wipe_failed.untouched"
	}
	h.notifyServerOwner(ctx, run.ServerID, notify.Event{
		Kind:     notify.KindWipeFailed,
		Title:    i18n.Key("notify.wipe_failed.title"),
		Body:     i18n.Key(bodyKey, i18n.Params{"name": name, "reason": reason}),
		Action:   h.serverAction("notify.action.open_server", run.ServerID, "/wipes"),
		Meta:     map[string]any{"server_id": run.ServerID, "run_id": run.ID},
		ServerID: run.ServerID,
	})
}

type agentWipeFS struct {
	h        *Handler
	nodeID   string
	serverID string
}

func (f agentWipeFS) List(ctx context.Context, dir string) ([]wipe.Entry, error) {
	res, err := f.h.agentCommand(ctx, f.nodeID, f.serverID, "files_list", map[string]any{"path": dir})
	if err != nil {
		return nil, err
	}
	var out []wipe.Entry
	if files, ok := res["files"].([]any); ok {
		for _, item := range files {
			entry, _ := item.(map[string]any)
			name, _ := entry["name"].(string)
			isDir, _ := entry["is_dir"].(bool)
			if name != "" {
				out = append(out, wipe.Entry{Name: name, IsDir: isDir})
			}
		}
	}
	return out, nil
}

func (f agentWipeFS) Delete(ctx context.Context, p string) error {
	_, err := f.h.agentCommandWait(ctx, f.nodeID, f.serverID, "files_delete", map[string]any{"path": p}, 5*time.Minute)
	return err
}

func (h *Handler) executeWipe(ctx context.Context, run wipeRunRow) {
	stopKeepAlive := h.wipeKeepAlive(ctx, run.ID)
	defer stopKeepAlive()

	nodeID, gameID, name, _, err := h.wipeServerBasics(ctx, run.ServerID)
	if err != nil {
		h.wipeFail(ctx, run, "prepare", errors.New("сервер не найден"), true)
		return
	}
	recipe, ok := wipe.RecipeFor(gameID)
	if !ok {
		h.wipeFail(ctx, run, "prepare", errWipeNoPlan, true)
		return
	}
	if nodeID == "" {
		h.wipeFail(ctx, run, "prepare", errors.New("сервер не привязан к узлу"), true)
		return
	}

	wasRunning := h.serverRunning(ctx, run.ServerID)
	h.wipeSetDetail(ctx, run.ID, "was_running", wasRunning)

	env := map[string]string{}
	if envPath := recipe.EnvFile(); envPath != "" {
		if content, err := h.wipeReadFile(ctx, nodeID, run.ServerID, envPath); err == nil {
			if codec, ok := gamesettings.CodecFor(gamesettings.FormatEnv); ok {
				env = codec.Parse(content)
			}
		}
	}
	targets, err := recipe.Targets(run.Kind, env)
	if err != nil {
		h.wipeFail(ctx, run, "prepare", err, true)
		return
	}

	if run.BackupBefore {
		h.wipeSetStage(ctx, run.ID, "backup")
		if !wasRunning {
			h.wipeSetDetail(ctx, run.ID, "backup", "skipped: server stopped")
		} else {
			if recipe.SaveTool != "" {
				saveCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				_, _ = h.consoleTool(saveCtx, run.ServerID, gameID, recipe.SaveTool, nil)
				cancel()
				sleepCtx(ctx, 5*time.Second)
			}
			backupName := "pre-wipe-" + time.Now().Format(wipeBackupNameTime)
			_, filename, size, err := h.createServerBackup(ctx, run.ServerID, backupName, "wipe")
			if err != nil {
				h.wipeFail(ctx, run, "backup", fmt.Errorf("резервная копия не создана: %w", err), true)
				return
			}
			h.wipeSetDetail(ctx, run.ID, "backup", filename)
			h.wipeSetDetail(ctx, run.ID, "backup_bytes", size)
		}
	}

	if wasRunning {
		h.wipeSetStage(ctx, run.ID, "stop")
		if _, _, err := h.dispatchServerPower(ctx, run.CreatedBy, run.ServerID, "stop", stopWait); err != nil {
			if _, _, killErr := h.dispatchServerPower(ctx, run.CreatedBy, run.ServerID, "kill", stopWait); killErr != nil {
				h.wipeFail(ctx, run, "stop", fmt.Errorf("сервер не остановлен: %w", err), true)
				return
			}
		}
		h.wipeSetDetail(ctx, run.ID, "stopped", true)
	}

	h.wipeSetStage(ctx, run.ID, "files")
	res, err := wipe.Execute(ctx, agentWipeFS{h: h, nodeID: nodeID, serverID: run.ServerID}, targets)
	h.wipeSetDetail(ctx, run.ID, "deleted", res.Deleted)
	h.wipeSetDetail(ctx, run.ID, "sample", res.Sample)
	if err != nil {
		h.wipeFail(ctx, run, "files", err, false)
		return
	}

	if recipe.Seed && run.NewSeed {
		h.wipeSetStage(ctx, run.ID, "seed")
		level := strings.TrimSpace(env["RUST_LEVEL"])
		if level == "" || strings.Contains(strings.ToLower(level), "procedural") {
			seed := strconv.FormatInt(rand.Int64N(wipeSeedMax)+1, 10)
			if err := h.wipeWriteSeed(ctx, nodeID, run.ServerID, recipe, seed); err != nil {
				h.wipeFail(ctx, run, "seed", fmt.Errorf("новое зерно карты не записано: %w", err), false)
				return
			}
			h.wipeSetDetail(ctx, run.ID, "seed", seed)
		} else {
			h.wipeSetDetail(ctx, run.ID, "seed", "skipped: custom map")
		}
	}

	if wasRunning {
		h.wipeSetStage(ctx, run.ID, "start")
		if _, _, err := h.sendServerPower(ctx, run.CreatedBy, run.ServerID, "start"); err != nil {
			h.wipeFail(ctx, run, "start", fmt.Errorf("вайп выполнен, но сервер не запустился: %w", err), false)
			return
		}
	}

	_, _ = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.server_wipe_runs SET status = 'completed', stage = 'done', finished_at = now() WHERE id = $1
	`, run.ID)
	if run.PlanID != "" {
		_, _ = h.dbOf(ctx).Exec(ctx, `
			UPDATE core.server_wipe_plans SET last_run_at = now(), last_status = 'completed', updated_at = now() WHERE id = $1
		`, run.PlanID)
	}
	audit(ctx, h.dbOf(ctx), run.CreatedBy, "server.wipe", "server:"+run.ServerID, map[string]any{
		"kind": run.Kind, "source": run.Source, "deleted": res.Deleted,
	})
	if run.NotifyOwner {
		h.notifyServerOwner(ctx, run.ServerID, notify.Event{
			Kind:  notify.KindWipeDone,
			Title: i18n.Key("notify.wipe_done.title"),
			Body: i18n.Key("notify.wipe_done.body", i18n.Params{
				"name": name,
				"kind": i18n.Key("notify.wipe_kind." + run.Kind),
			}),
			Action:   h.serverAction("notify.action.open_server", run.ServerID, "/wipes"),
			Meta:     map[string]any{"server_id": run.ServerID, "run_id": run.ID},
			ServerID: run.ServerID,
		})
	}
}

func (h *Handler) wipeReadFile(ctx context.Context, nodeID, serverID, path string) (string, error) {
	res, err := h.agentCommand(ctx, nodeID, serverID, "files_read", map[string]any{"path": path})
	if err != nil {
		return "", err
	}
	content, _ := res["content"].(string)
	return content, nil
}

func (h *Handler) wipeWriteSeed(ctx context.Context, nodeID, serverID string, recipe wipe.Recipe, seed string) error {
	path := recipe.EnvFile()
	content, _ := h.wipeReadFile(ctx, nodeID, serverID, path)
	updated, err := gamesettings.ApplyFile(
		gamesettings.ConfigFile{Path: strings.TrimPrefix(path, "/"), Format: gamesettings.FormatEnv},
		content, map[string]string{recipe.SeedKey(): seed})
	if err != nil {
		return err
	}
	_, err = h.agentCommand(ctx, nodeID, serverID, "files_write", map[string]any{"path": path, "content": updated})
	return err
}

type wipeRunParams struct {
	PlanID           string
	Kind             string
	BackupBefore     bool
	NewSeed          bool
	NotifyOwner      bool
	CountdownMinutes int
	AnnounceText     string
}

func manualAnnounceMinutes(countdown int) []int {
	if countdown <= 0 {
		return nil
	}
	out := []int{countdown}
	for _, m := range wipe.DefaultAnnounceMinutes() {
		if m < countdown {
			out = append(out, m)
		}
	}
	return wipe.NormalizeMinutes(out)
}

func (h *Handler) wipeCreateRun(ctx context.Context, serverID, source, userID string, p wipeRunParams) (string, error) {
	_, gameID, _, _, err := h.wipeServerBasics(ctx, serverID)
	if err != nil {
		return "", err
	}
	recipe, ok := wipe.RecipeFor(gameID)
	if !ok {
		return "", errWipeNoPlan
	}
	if !recipe.HasKind(p.Kind) {
		return "", wipe.ErrUnknownKind
	}
	countdown := min(max(p.CountdownMinutes, 0), 1440)
	var id string
	err = h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.server_wipe_runs
			(plan_id, server_id, kind, source, status, starts_at, backup_before, new_seed,
			 announce_minutes, announce_text, notify_owner, created_by)
		VALUES (NULLIF($1, '')::uuid, $2, $3, $4, 'announcing', now() + $5::interval, $6, $7, $8, $9, $10, NULLIF($11, '')::uuid)
		ON CONFLICT (server_id) WHERE status IN ('announcing', 'running') DO NOTHING
		RETURNING id::text
	`, p.PlanID, serverID, p.Kind, source, (time.Duration(countdown) * time.Minute).String(),
		p.BackupBefore, p.NewSeed && recipe.Seed, ints32(manualAnnounceMinutes(countdown)),
		strings.TrimSpace(p.AnnounceText), p.NotifyOwner, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errWipeBusy
	}
	if err != nil {
		return "", err
	}
	audit(ctx, h.dbOf(ctx), userID, "server.wipe_requested", "server:"+serverID, map[string]any{
		"kind": p.Kind, "source": source, "countdown": countdown,
	})
	return id, nil
}

func (h *Handler) wipeCancelRun(ctx context.Context, serverID, runID string) (bool, error) {
	tag, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.server_wipe_runs
		SET status = 'cancelled', finished_at = now()
		WHERE id = $1 AND server_id = $2 AND status = 'announcing'
	`, runID, serverID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}
