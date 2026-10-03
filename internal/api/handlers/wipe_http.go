package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/pkg/settingsreg"
	"github.com/vortanixapp/panel/pkg/wipe"
)

type wipePlanBody struct {
	Name            *string `json:"name"`
	Kind            *string `json:"kind"`
	Enabled         *bool   `json:"enabled"`
	ScheduleType    *string `json:"schedule_type"`
	Weekday         *int    `json:"weekday"`
	Nth             *int    `json:"nth"`
	Time            *string `json:"time"`
	Cron            *string `json:"cron"`
	RunAt           *string `json:"run_at"`
	Timezone        *string `json:"timezone"`
	BackupBefore    *bool   `json:"backup_before"`
	NewSeed         *bool   `json:"new_seed"`
	AnnounceMinutes *[]int  `json:"announce_minutes"`
	AnnounceText    *string `json:"announce_text"`
	NotifyOwner     *bool   `json:"notify_owner"`
	SkipNext        *bool   `json:"skip_next"`
}

func wipePlanJSON(p wipePlanRow) map[string]any {
	out := map[string]any{
		"id":               p.ID,
		"name":             p.Name,
		"kind":             p.Kind,
		"enabled":          p.Enabled,
		"schedule_type":    p.Schedule.Type,
		"weekday":          p.Schedule.Weekday,
		"nth":              p.Schedule.Nth,
		"time":             p.Schedule.TimeOfDay,
		"cron":             p.Schedule.Cron,
		"timezone":         p.Schedule.TZ,
		"backup_before":    p.BackupBefore,
		"new_seed":         p.NewSeed,
		"announce_minutes": ints(p.AnnounceMinutes),
		"announce_text":    p.AnnounceText,
		"notify_owner":     p.NotifyOwner,
		"skip_next":        p.SkipNext,
		"next_run_at":      p.NextRunAt,
		"last_run_at":      p.LastRunAt,
		"last_status":      p.LastStatus,
	}
	if !p.Schedule.RunAt.IsZero() {
		out["run_at"] = p.Schedule.RunAt
		if loc, err := time.LoadLocation(p.Schedule.TZ); err == nil {
			out["run_at_local"] = p.Schedule.RunAt.In(loc).Format("2006-01-02T15:04")
		}
	}
	return out
}

func (h *Handler) wipeLoadPlans(ctx context.Context, serverID string) []wipePlanRow {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT `+wipePlanColumns+`
		FROM core.server_wipe_plans p
		WHERE p.server_id = $1
		ORDER BY p.created_at, p.id
	`, serverID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []wipePlanRow
	for rows.Next() {
		if p, err := scanWipePlan(rows); err == nil {
			out = append(out, p)
		}
	}
	return out
}

func (h *Handler) ServerWipesInfo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, id, "wipes_read"); !ok {
		return
	}
	ctx := r.Context()
	_, gameID, _, _, err := h.wipeServerBasics(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	recipe, ok := wipe.RecipeFor(gameID)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"supported": false})
		return
	}

	plans := make([]map[string]any, 0)
	for _, p := range h.wipeLoadPlans(ctx, id) {
		plans = append(plans, wipePlanJSON(p))
	}

	runs := make([]map[string]any, 0)
	var active map[string]any
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT id::text, COALESCE(plan_id::text, ''), kind, source, status, stage, starts_at,
		       started_at, finished_at, error, details, created_at
		FROM core.server_wipe_runs
		WHERE server_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`, id, wipeHistoryLimit)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var runID, planID, kind, source, status, stage, errText string
			var startsAt, createdAt time.Time
			var startedAt, finishedAt *time.Time
			var details []byte
			if rows.Scan(&runID, &planID, &kind, &source, &status, &stage, &startsAt,
				&startedAt, &finishedAt, &errText, &details, &createdAt) != nil {
				continue
			}
			item := map[string]any{
				"id": runID, "plan_id": planID, "kind": kind, "source": source, "status": status,
				"stage": stage, "starts_at": startsAt, "started_at": startedAt, "finished_at": finishedAt,
				"error": errText, "details": json.RawMessage(details), "created_at": createdAt,
			}
			runs = append(runs, item)
			if active == nil && (status == "announcing" || status == "running") {
				active = item
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"supported":                true,
		"kinds":                    recipe.Kinds,
		"seed":                     recipe.Seed,
		"announce":                 recipe.AnnounceTool != "",
		"timezone":                 h.serverCronTimezone(ctx, id),
		"default_announce_minutes": wipe.DefaultAnnounceMinutes(),
		"max_plans":                intOf(settingsreg.ServersWipeMaxPlans),
		"plans":                    plans,
		"runs":                     runs,
		"active_run":               active,
	})
}

func (h *Handler) wipeApplyPlanBody(base wipePlanRow, body wipePlanBody, recipe wipe.Recipe, defaultTZ string) (wipePlanRow, error) {
	p := base
	if body.Name != nil {
		name := strings.TrimSpace(*body.Name)
		if utf8.RuneCountInString(name) > 80 {
			return p, errors.New("название не длиннее 80 символов")
		}
		p.Name = name
	}
	if body.Kind != nil {
		p.Kind = strings.TrimSpace(*body.Kind)
	}
	if !recipe.HasKind(p.Kind) {
		return p, wipe.ErrUnknownKind
	}
	if body.Enabled != nil {
		p.Enabled = *body.Enabled
	}
	if body.ScheduleType != nil {
		p.Schedule.Type = strings.TrimSpace(*body.ScheduleType)
	}
	if body.Weekday != nil {
		p.Schedule.Weekday = *body.Weekday
	}
	if body.Nth != nil {
		p.Schedule.Nth = *body.Nth
	}
	if body.Time != nil {
		p.Schedule.TimeOfDay = strings.TrimSpace(*body.Time)
	}
	if body.Cron != nil {
		p.Schedule.Cron = strings.Join(strings.Fields(*body.Cron), " ")
	}
	if body.Timezone != nil {
		p.Schedule.TZ = strings.TrimSpace(*body.Timezone)
	}
	if p.Schedule.TZ == "" {
		p.Schedule.TZ = defaultTZ
	}
	if body.RunAt != nil {
		raw := strings.TrimSpace(*body.RunAt)
		if raw == "" {
			p.Schedule.RunAt = time.Time{}
		} else {
			t, err := parseWipeRunAt(raw, p.Schedule.TZ)
			if err != nil {
				return p, err
			}
			p.Schedule.RunAt = t
		}
	}
	if body.BackupBefore != nil {
		p.BackupBefore = *body.BackupBefore
	}
	if body.NewSeed != nil {
		p.NewSeed = *body.NewSeed
	}
	if !recipe.Seed {
		p.NewSeed = false
	}
	if body.AnnounceMinutes != nil {
		p.AnnounceMinutes = ints32(wipe.NormalizeMinutes(*body.AnnounceMinutes))
	}
	if body.AnnounceText != nil {
		text := strings.TrimSpace(*body.AnnounceText)
		if utf8.RuneCountInString(text) > 200 {
			return p, errors.New("текст объявления не длиннее 200 символов")
		}
		p.AnnounceText = text
	}
	if body.NotifyOwner != nil {
		p.NotifyOwner = *body.NotifyOwner
	}
	if body.SkipNext != nil {
		p.SkipNext = *body.SkipNext
	}
	if err := p.Schedule.Validate(); err != nil {
		return p, err
	}
	return p, nil
}

func parseWipeRunAt(raw, tz string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.Time{}, errors.New("неизвестный часовой пояс")
	}
	t, err := time.ParseInLocation("2006-01-02T15:04", raw, loc)
	if err != nil {
		return time.Time{}, errors.New("время вайпа указано неверно")
	}
	return t, nil
}

func scheduleChanged(a, b wipe.Schedule) bool {
	return a.Type != b.Type || a.Weekday != b.Weekday || a.Nth != b.Nth || a.TimeOfDay != b.TimeOfDay ||
		a.Cron != b.Cron || !a.RunAt.Equal(b.RunAt) || a.TZ != b.TZ
}

func (h *Handler) wipePlanRecipe(w http.ResponseWriter, r *http.Request, serverID string) (wipe.Recipe, bool) {
	_, gameID, _, _, err := h.wipeServerBasics(r.Context(), serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return wipe.Recipe{}, false
	}
	recipe, ok := wipe.RecipeFor(gameID)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, errWipeNoPlan.Error())
		return wipe.Recipe{}, false
	}
	return recipe, true
}

func (h *Handler) ServerWipePlanCreate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims, ok := h.authorizeServerTab(w, r, id, "wipes_write")
	if !ok {
		return
	}
	var body wipePlanBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	recipe, ok := h.wipePlanRecipe(w, r, id)
	if !ok {
		return
	}
	ctx := r.Context()

	var count int
	_ = h.dbOf(ctx).QueryRow(ctx, `SELECT count(*) FROM core.server_wipe_plans WHERE server_id = $1`, id).Scan(&count)
	if count >= intOf(settingsreg.ServersWipeMaxPlans) {
		writeError(w, http.StatusUnprocessableEntity, "Достигнут предел планов вайпа для сервера")
		return
	}

	base := wipePlanRow{
		ServerID:        id,
		Enabled:         true,
		BackupBefore:    true,
		NewSeed:         true,
		NotifyOwner:     true,
		AnnounceMinutes: ints32(wipe.DefaultAnnounceMinutes()),
		Schedule:        wipe.Schedule{Type: wipe.ScheduleWeekly, Weekday: 4, Nth: 1, TimeOfDay: "19:00"},
	}
	plan, err := h.wipeApplyPlanBody(base, body, recipe, h.serverCronTimezone(ctx, id))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	var nextArg any
	if plan.Enabled {
		next, ok := plan.Schedule.Next(time.Now())
		if !ok {
			writeError(w, http.StatusUnprocessableEntity, "Не удалось вычислить время ближайшего вайпа")
			return
		}
		nextArg = next
	}

	var planID string
	err = h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.server_wipe_plans
			(server_id, name, kind, enabled, schedule_type, weekday, nth, time_of_day, cron_expr, run_at,
			 timezone, backup_before, new_seed, announce_minutes, announce_text, notify_owner, next_run_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id::text
	`, id, plan.Name, plan.Kind, plan.Enabled, plan.Schedule.Type, plan.Schedule.Weekday, plan.Schedule.Nth,
		plan.Schedule.TimeOfDay, plan.Schedule.Cron, nullableTime(plan.Schedule.RunAt), plan.Schedule.TZ,
		plan.BackupBefore, plan.NewSeed, plan.AnnounceMinutes, plan.AnnounceText, plan.NotifyOwner,
		nextArg, claims.UserID).Scan(&planID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "server.wipe_plan_create", "server:"+id, map[string]any{"plan_id": planID, "kind": plan.Kind})
	writeJSON(w, http.StatusCreated, map[string]any{"id": planID})
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}

func (h *Handler) wipeLoadPlan(r *http.Request, serverID, planID string) (wipePlanRow, error) {
	return scanWipePlan(h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT `+wipePlanColumns+`
		FROM core.server_wipe_plans p
		WHERE p.id::text = $1 AND p.server_id = $2
	`, planID, serverID))
}

func (h *Handler) ServerWipePlanUpdate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims, ok := h.authorizeServerTab(w, r, id, "wipes_write")
	if !ok {
		return
	}
	var body wipePlanBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	recipe, ok := h.wipePlanRecipe(w, r, id)
	if !ok {
		return
	}
	ctx := r.Context()
	current, err := h.wipeLoadPlan(r, id, chi.URLParam(r, "planId"))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "plan not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	plan, err := h.wipeApplyPlanBody(current, body, recipe, h.serverCronTimezone(ctx, id))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	next := plan.NextRunAt
	recompute := scheduleChanged(current.Schedule, plan.Schedule) || (plan.Enabled && !current.Enabled)
	if recompute && plan.Enabled {
		t, ok := plan.Schedule.Next(time.Now())
		if !ok {
			writeError(w, http.StatusUnprocessableEntity, "Не удалось вычислить время ближайшего вайпа")
			return
		}
		next = &t
	}
	if plan.Enabled && next == nil {
		t, ok := plan.Schedule.Next(time.Now())
		if !ok {
			writeError(w, http.StatusUnprocessableEntity, "Не удалось вычислить время ближайшего вайпа")
			return
		}
		next = &t
	}
	skip := plan.SkipNext && plan.Enabled

	_, err = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.server_wipe_plans SET
			name = $3, kind = $4, enabled = $5, schedule_type = $6, weekday = $7, nth = $8,
			time_of_day = $9, cron_expr = $10, run_at = $11, timezone = $12, backup_before = $13,
			new_seed = $14, announce_minutes = $15, announce_text = $16, notify_owner = $17,
			skip_next = $18, next_run_at = $19, updated_at = now()
		WHERE id::text = $1 AND server_id = $2
	`, current.ID, id, plan.Name, plan.Kind, plan.Enabled, plan.Schedule.Type, plan.Schedule.Weekday,
		plan.Schedule.Nth, plan.Schedule.TimeOfDay, plan.Schedule.Cron, nullableTime(plan.Schedule.RunAt),
		plan.Schedule.TZ, plan.BackupBefore, plan.NewSeed, plan.AnnounceMinutes, plan.AnnounceText,
		plan.NotifyOwner, skip, next)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "server.wipe_plan_update", "server:"+id, map[string]any{"plan_id": current.ID})
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *Handler) ServerWipePlanDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims, ok := h.authorizeServerTab(w, r, id, "wipes_write")
	if !ok {
		return
	}
	planID := chi.URLParam(r, "planId")
	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		DELETE FROM core.server_wipe_plans WHERE id::text = $1 AND server_id = $2
	`, planID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "plan not found")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "server.wipe_plan_delete", "server:"+id, map[string]any{"plan_id": planID})
	writeJSON(w, http.StatusOK, map[string]any{"status": "deleted"})
}

func (h *Handler) ServerWipeRun(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims, ok := h.authorizeServerTab(w, r, id, "wipes_write")
	if !ok {
		return
	}
	var body struct {
		Kind             string `json:"kind"`
		BackupBefore     *bool  `json:"backup_before"`
		NewSeed          *bool  `json:"new_seed"`
		CountdownMinutes int    `json:"countdown_minutes"`
		AnnounceText     string `json:"announce_text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if utf8.RuneCountInString(body.AnnounceText) > 200 {
		writeError(w, http.StatusUnprocessableEntity, "текст объявления не длиннее 200 символов")
		return
	}
	if h.freezeActive(r.Context()) {
		writeError(w, http.StatusLocked, "Панель в режиме только чтение")
		return
	}
	params := wipeRunParams{
		Kind:             strings.TrimSpace(body.Kind),
		BackupBefore:     body.BackupBefore == nil || *body.BackupBefore,
		NewSeed:          body.NewSeed == nil || *body.NewSeed,
		NotifyOwner:      true,
		CountdownMinutes: body.CountdownMinutes,
		AnnounceText:     body.AnnounceText,
	}
	runID, err := h.wipeCreateRun(r.Context(), id, "manual", claims.UserID, params)
	switch {
	case errors.Is(err, errWipeBusy):
		writeError(w, http.StatusConflict, err.Error())
		return
	case errors.Is(err, errWipeNoPlan), errors.Is(err, wipe.ErrUnknownKind):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	case err != nil:
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"run_id": runID})
}

func (h *Handler) ServerWipeRunCancel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims, ok := h.authorizeServerTab(w, r, id, "wipes_write")
	if !ok {
		return
	}
	cancelled, err := h.wipeCancelRun(r.Context(), id, chi.URLParam(r, "runId"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if !cancelled {
		writeError(w, http.StatusConflict, "Вайп уже начался или завершён и не может быть отменён")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "server.wipe_cancelled", "server:"+id, nil)
	writeJSON(w, http.StatusOK, map[string]any{"status": "cancelled"})
}
