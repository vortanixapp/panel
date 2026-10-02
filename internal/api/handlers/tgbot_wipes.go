package handlers

import (
	"fmt"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
	"github.com/vortanixapp/panel/pkg/wipe"
)

var botCountdowns = []int{0, 5, 15, 30}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func (u *botUI) scheduleText(p wipePlanRow) string {
	day := u.t(fmt.Sprintf("notify.bot.wd.%d", p.Schedule.Weekday))
	switch p.Schedule.Type {
	case wipe.ScheduleWeekly:
		return u.t("notify.bot.sched.weekly", i18n.Params{"day": day, "time": p.Schedule.TimeOfDay})
	case wipe.ScheduleMonthly:
		nth := u.t("notify.bot.sched.nth", i18n.Params{"n": p.Schedule.Nth})
		if p.Schedule.Nth >= 5 {
			nth = u.t("notify.bot.sched.last")
		}
		return u.t("notify.bot.sched.monthly", i18n.Params{"nth": nth, "day": day, "time": p.Schedule.TimeOfDay})
	case wipe.ScheduleCron:
		return "cron " + p.Schedule.Cron
	}
	return u.t("notify.bot.sched.once")
}

func (u *botUI) wipeServer(serverID, action string) (*botServer, wipe.Recipe, *serverAccess, bool) {
	access, ok := u.gate(serverID, action)
	if !ok {
		return nil, wipe.Recipe{}, nil, false
	}
	s, err := u.h.botLoadServer(u.ctx, serverID)
	if err != nil {
		u.answer(u.t("notify.bot.srv_missing"), true)
		return nil, wipe.Recipe{}, nil, false
	}
	recipe, supported := wipe.RecipeFor(s.GameID)
	if !supported {
		u.answer(u.t("notify.bot.wipes_unsupported"), true)
		return nil, wipe.Recipe{}, nil, false
	}
	return s, recipe, access, true
}

func (u *botUI) activeWipe(serverID string) (id, kind, status string, startsAt time.Time, ok bool) {
	err := u.h.dbOf(u.ctx).QueryRow(u.ctx, `
		SELECT id::text, kind, status, starts_at
		FROM core.server_wipe_runs
		WHERE server_id = $1 AND status IN ('announcing', 'running')
		ORDER BY created_at DESC LIMIT 1
	`, serverID).Scan(&id, &kind, &status, &startsAt)
	return id, kind, status, startsAt, err == nil
}

func (u *botUI) viewWipes(serverID string) {
	s, _, access, ok := u.wipeServer(serverID, "wipes_read")
	if !ok {
		return
	}
	canWrite := false
	if status, _ := accessDenial(access, "wipes_write"); status == 0 {
		canWrite = true
	}
	plans := u.h.wipeLoadPlans(u.ctx, serverID)

	var b strings.Builder
	b.WriteString(u.t("notify.bot.wipes_title", i18n.Params{"name": esc(s.Name)}) + "\n")

	var kb notify.TelegramKeyboard
	runID, runKind, runStatus, runStarts, active := u.activeWipe(serverID)
	if active {
		fmt.Fprintf(&b, "\n🔔 %s\n", u.t("notify.bot.wipe_active", i18n.Params{
			"kind":   u.t("notify.wipe_kind." + runKind),
			"time":   u.dateTime(runStarts),
			"status": u.t("notify.bot.wipe_status." + runStatus),
		}))
		if canWrite && runStatus == "announcing" {
			kb = append(kb, []notify.TelegramButton{{
				Text: "❌ " + u.t("notify.bot.btn.wipe_cancel"),
				Data: notify.Callback(notify.CBWipeCancel, serverID, shortID(runID)),
			}})
		}
	}

	if len(plans) == 0 {
		b.WriteString("\n" + u.t("notify.bot.wipes_empty") + "\n")
	}
	if len(plans) > 5 {
		plans = plans[:5]
	}
	for i, p := range plans {
		title := p.Name
		if title == "" {
			title = u.t("notify.wipe_kind." + p.Kind)
		}
		state := "⏸"
		if p.Enabled {
			state = "✅"
		}
		fmt.Fprintf(&b, "\n%d. %s <b>%s</b>\n   %s\n", i+1, state, esc(title), esc(u.scheduleText(p)))
		if p.Enabled && p.NextRunAt != nil {
			line := u.t("notify.bot.wipe_next", i18n.Params{"time": u.dateTime(*p.NextRunAt)})
			if p.SkipNext {
				line += " · " + u.t("notify.bot.wipe_skipped")
			}
			fmt.Fprintf(&b, "   %s\n", line)
		}
		if canWrite {
			toggle := "⏸ "
			if !p.Enabled {
				toggle = "▶️ "
			}
			row := []notify.TelegramButton{{
				Text: fmt.Sprintf("%s%d", toggle, i+1),
				Data: notify.Callback(notify.CBWipeToggle, serverID, shortID(p.ID)),
			}}
			if p.Enabled {
				skip := "⏭ "
				if p.SkipNext {
					skip = "↩️ "
				}
				row = append(row, notify.TelegramButton{
					Text: fmt.Sprintf("%s%d", skip, i+1),
					Data: notify.Callback(notify.CBWipeSkip, serverID, shortID(p.ID)),
				})
			}
			kb = append(kb, row)
		}
	}
	if canWrite && !active {
		kb = append(kb, []notify.TelegramButton{{Text: "🧨 " + u.t("notify.bot.btn.wipe_now"), Data: notify.Callback(notify.CBWipeNow, serverID)}})
	}
	if link := u.h.botPanelURL("/servers/" + serverID + "/wipes"); link != "" {
		kb = append(kb, []notify.TelegramButton{{Text: "🌐 " + u.t("notify.bot.btn.open_panel"), URL: link}})
	}
	kb = append(kb, u.backToServer(serverID))
	u.show(b.String(), kb)
}

func (u *botUI) viewWipeNow(serverID, kindArg string) {
	s, recipe, _, ok := u.wipeServer(serverID, "wipes_write")
	if !ok {
		return
	}
	if kindArg == "" {
		var kb notify.TelegramKeyboard
		for i, kind := range recipe.Kinds {
			kb = append(kb, []notify.TelegramButton{{
				Text: u.t("notify.wipe_kind." + kind),
				Data: notify.Callback(notify.CBWipeNow, serverID, fmt.Sprint(i)),
			}})
		}
		kb = append(kb, []notify.TelegramButton{{Text: "⬅️ " + u.t("notify.bot.btn.wipes"), Data: notify.Callback(notify.CBWipe, serverID)}})
		u.show(u.t("notify.bot.wipe_pick_kind", i18n.Params{"name": esc(s.Name)}), kb)
		return
	}
	var idx int
	if _, err := fmt.Sscanf(kindArg, "%d", &idx); err != nil || idx < 0 || idx >= len(recipe.Kinds) {
		return
	}
	var kb notify.TelegramKeyboard
	for _, minutes := range botCountdowns {
		label := u.t("notify.bot.btn.wipe_immediately")
		if minutes > 0 {
			label = u.t("notify.bot.btn.wipe_in", i18n.Params{"minutes": minutes})
		}
		kb = append(kb, []notify.TelegramButton{{
			Text: "🧨 " + label,
			Data: notify.Callback(notify.CBWipeRun, serverID, fmt.Sprint(idx), fmt.Sprint(minutes)),
		}})
	}
	kb = append(kb, []notify.TelegramButton{{Text: "⬅️ " + u.t("notify.bot.btn.back"), Data: notify.Callback(notify.CBWipeNow, serverID)}})
	u.show(u.t("notify.bot.wipe_confirm", i18n.Params{
		"name": esc(s.Name),
		"kind": u.t("notify.wipe_kind." + recipe.Kinds[idx]),
		"seed": map[bool]string{true: u.t("notify.bot.wipe_seed"), false: ""}[recipe.Seed],
	}), kb)
}

func (u *botUI) actionWipeRun(serverID string, kindIdx, minutes int) {
	_, recipe, _, ok := u.wipeServer(serverID, "wipes_write")
	if !ok {
		return
	}
	if kindIdx < 0 || kindIdx >= len(recipe.Kinds) {
		return
	}
	allowedCountdown := false
	for _, m := range botCountdowns {
		if m == minutes {
			allowedCountdown = true
		}
	}
	if !allowedCountdown {
		return
	}
	if u.h.freezeActive(u.ctx) {
		u.answer(u.t("notify.bot.frozen"), true)
		return
	}
	_, err := u.h.wipeCreateRun(u.ctx, serverID, "telegram", u.user.UserID, wipeRunParams{
		Kind:             recipe.Kinds[kindIdx],
		BackupBefore:     true,
		NewSeed:          true,
		NotifyOwner:      true,
		CountdownMinutes: minutes,
	})
	switch {
	case err == nil:
		u.answer(u.t("notify.bot.wipe_started"), false)
		u.viewWipes(serverID)
	case err == errWipeBusy:
		u.answer(u.t("notify.bot.wipe_busy"), true)
	default:
		u.answer(u.t("notify.bot.error"), true)
	}
}

func (u *botUI) planByShort(serverID, short string) (wipePlanRow, bool) {
	for _, p := range u.h.wipeLoadPlans(u.ctx, serverID) {
		if strings.HasPrefix(p.ID, short) && len(short) >= 6 {
			return p, true
		}
	}
	u.answer(u.t("notify.bot.list_stale"), true)
	return wipePlanRow{}, false
}

func (u *botUI) actionWipeToggle(serverID, short string, skip bool) {
	if _, _, _, ok := u.wipeServer(serverID, "wipes_write"); !ok {
		return
	}
	p, ok := u.planByShort(serverID, short)
	if !ok {
		return
	}
	if skip {
		_, _ = u.h.dbOf(u.ctx).Exec(u.ctx, `
			UPDATE core.server_wipe_plans SET skip_next = NOT skip_next, updated_at = now() WHERE id = $1
		`, p.ID)
	} else if p.Enabled {
		_, _ = u.h.dbOf(u.ctx).Exec(u.ctx, `
			UPDATE core.server_wipe_plans SET enabled = false, skip_next = false, updated_at = now() WHERE id = $1
		`, p.ID)
	} else {
		next, found := p.Schedule.Next(time.Now())
		if !found {
			u.answer(u.t("notify.bot.wipe_no_next"), true)
			return
		}
		_, _ = u.h.dbOf(u.ctx).Exec(u.ctx, `
			UPDATE core.server_wipe_plans SET enabled = true, next_run_at = $2, updated_at = now() WHERE id = $1
		`, p.ID, next)
	}
	audit(u.ctx, u.h.dbOf(u.ctx), u.user.UserID, "server.wipe_plan_telegram", "server:"+serverID, map[string]any{"plan_id": p.ID, "skip": skip})
	u.viewWipes(serverID)
}

func (u *botUI) actionWipeCancel(serverID, short string) {
	if _, _, _, ok := u.wipeServer(serverID, "wipes_write"); !ok {
		return
	}
	id, _, status, _, active := u.activeWipe(serverID)
	if !active || status != "announcing" || !strings.HasPrefix(id, short) {
		u.answer(u.t("notify.bot.wipe_cannot_cancel"), true)
		u.viewWipes(serverID)
		return
	}
	cancelled, err := u.h.wipeCancelRun(u.ctx, serverID, id)
	if err != nil || !cancelled {
		u.answer(u.t("notify.bot.wipe_cannot_cancel"), true)
	} else {
		u.answer(u.t("notify.bot.wipe_cancelled"), false)
	}
	u.viewWipes(serverID)
}
