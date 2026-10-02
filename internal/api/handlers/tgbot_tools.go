package handlers

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vortanixapp/panel/pkg/gameconsole"
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

const (
	botOutputLimit  = 3000
	botCommandLimit = 500
	botBackupsShown = 8
	botSparkPoints  = 24
)

var ansiPattern = regexp.MustCompile(`\x1B\[[0-?]*[ -/]*[@-~]`)

func tailRunes(s string, limit int) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	r := []rune(s)
	return "…" + string(r[len(r)-limit:])
}

func (u *botUI) outputBlock(title, out string) string {
	out = ansiPattern.ReplaceAllString(strings.TrimSpace(out), "")
	if out == "" {
		out = u.t("notify.bot.no_output")
	}
	return "<b>" + esc(title) + "</b>\n<pre>" + esc(tailRunes(out, botOutputLimit)) + "</pre>"
}

func (u *botUI) backToServer(serverID string) []notify.TelegramButton {
	return []notify.TelegramButton{{Text: "⬅️ " + u.t("notify.bot.btn.server"), Data: notify.Callback(notify.CBServer, serverID)}}
}

func (u *botUI) detach(timeout time.Duration) (*botUI, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return &botUI{h: u.h, ctx: ctx, bot: u.bot, user: u.user, l: u.l, chatID: u.chatID, msgID: u.msgID, tgID: u.tgID}, cancel
}

func (u *botUI) viewConsole(serverID string) {
	if _, ok := u.gate(serverID, "console_command"); !ok {
		return
	}
	s, err := u.h.botLoadServer(u.ctx, serverID)
	if err != nil {
		u.answer(u.t("notify.bot.srv_missing"), true)
		return
	}
	profile := gameconsole.For(s.GameID)
	var kb notify.TelegramKeyboard
	var row []notify.TelegramButton
	for i, c := range profile.Commands {
		label := c.Label
		if c.Danger {
			label = "⚠️ " + label
		}
		row = append(row, notify.TelegramButton{Text: label, Data: notify.Callback(notify.CBQuick, serverID, fmt.Sprint(i))})
		if len(row) == 2 {
			kb = append(kb, row)
			row = nil
		}
	}
	if len(row) > 0 {
		kb = append(kb, row)
	}
	kb = append(kb, []notify.TelegramButton{{Text: "✏️ " + u.t("notify.bot.btn.custom"), Data: notify.Callback(notify.CBCustom, serverID)}})
	kb = append(kb, u.backToServer(serverID))
	u.show(u.t("notify.bot.console_title", i18n.Params{"name": esc(s.Name)}), kb)
}

func (u *botUI) actionCustomCommand(serverID string) {
	if _, ok := u.gate(serverID, "console_command"); !ok {
		return
	}
	u.h.botSetState(u.ctx, u.tgID, botState{Kind: "console", ServerID: serverID})
	u.show(u.t("notify.bot.custom_prompt"), notify.TelegramKeyboard{
		{{Text: "✖️ " + u.t("notify.bot.btn.done"), Data: notify.Callback(notify.CBConsole, serverID)}},
	})
}

func (u *botUI) runConsole(serverID, command string) (string, bool) {
	if _, ok := u.gate(serverID, "console_command"); !ok {
		return "", false
	}
	if u.h.freezeActive(u.ctx) {
		u.answer(u.t("notify.bot.frozen"), true)
		return "", false
	}
	s, err := u.h.botLoadServer(u.ctx, serverID)
	if err != nil {
		u.answer(u.t("notify.bot.srv_missing"), true)
		return "", false
	}
	if !u.h.serverRunning(u.ctx, serverID) {
		u.answer(u.t("notify.bot.need_running"), true)
		return "", false
	}
	u.bot.SendChatAction(u.ctx, u.chatID, "typing")
	out, err := u.h.consoleSend(u.ctx, serverID, s.GameID, command)
	audit(u.ctx, u.h.dbOf(u.ctx), u.user.UserID, "server.console_telegram", "server:"+serverID, map[string]any{"command": shorten(command, 200)})
	if err != nil {
		_, text := explainAgentError("console_command", err)
		return text, true
	}
	return out, true
}

func (u *botUI) actionQuick(serverID string, idx int, confirmed bool) {
	if _, ok := u.gate(serverID, "console_command"); !ok {
		return
	}
	s, err := u.h.botLoadServer(u.ctx, serverID)
	if err != nil {
		u.answer(u.t("notify.bot.srv_missing"), true)
		return
	}
	profile := gameconsole.For(s.GameID)
	if idx < 0 || idx >= len(profile.Commands) {
		return
	}
	cmd := profile.Commands[idx]
	if len(cmd.Args) > 0 {
		u.h.botSetState(u.ctx, u.tgID, botState{Kind: "cmdargs", ServerID: serverID, Index: idx, Args: map[string]string{}})
		u.show(u.argPrompt(cmd, 0), notify.TelegramKeyboard{
			{{Text: "✖️ " + u.t("notify.bot.btn.no"), Data: notify.Callback(notify.CBConsole, serverID)}},
		})
		return
	}
	if cmd.Danger && !confirmed {
		u.show(u.t("notify.bot.confirm_command", i18n.Params{"label": esc(cmd.Label), "name": esc(s.Name)}), notify.TelegramKeyboard{
			{
				{Text: "✅ " + u.t("notify.bot.btn.yes"), Data: notify.Callback(notify.CBQuick, serverID, fmt.Sprint(idx), "y")},
				{Text: "✖️ " + u.t("notify.bot.btn.no"), Data: notify.Callback(notify.CBConsole, serverID)},
			},
		})
		return
	}
	line, err := cmd.Render(nil)
	if err != nil {
		u.answer(err.Error(), true)
		return
	}
	out, ok := u.runConsole(serverID, line)
	if !ok {
		return
	}
	u.show(u.outputBlock(cmd.Label, out), notify.TelegramKeyboard{
		{{Text: "⌨️ " + u.t("notify.bot.btn.console"), Data: notify.Callback(notify.CBConsole, serverID)}},
		u.backToServer(serverID),
	})
}

func (u *botUI) argPrompt(cmd gameconsole.Command, step int) string {
	arg := cmd.Args[step]
	text := u.t("notify.bot.arg_prompt", i18n.Params{"command": esc(cmd.Label), "arg": esc(arg.Label)})
	if arg.Optional {
		text += "\n" + u.t("notify.bot.arg_optional")
	}
	return text
}

func (u *botUI) handleStateInput(st botState, text string) {
	if utf8.RuneCountInString(text) > botCommandLimit || strings.ContainsAny(text, "\r\n") {
		u.send(u.t("notify.bot.too_long"), nil)
		return
	}
	switch st.Kind {
	case "console":
		out, ok := u.runConsole(st.ServerID, text)
		if !ok {
			u.h.botClearState(u.ctx, u.tgID)
			u.send(u.t("notify.bot.denied"), notify.TelegramKeyboard{u.backToServer(st.ServerID)})
			return
		}
		u.h.botSetState(u.ctx, u.tgID, st)
		u.send(u.outputBlock("> "+shorten(text, 60), out), notify.TelegramKeyboard{
			{{Text: "✖️ " + u.t("notify.bot.btn.done"), Data: notify.Callback(notify.CBConsole, st.ServerID)}},
		})
	case "cmdargs":
		s, err := u.h.botLoadServer(u.ctx, st.ServerID)
		if err != nil {
			u.h.botClearState(u.ctx, u.tgID)
			return
		}
		profile := gameconsole.For(s.GameID)
		if st.Index < 0 || st.Index >= len(profile.Commands) {
			u.h.botClearState(u.ctx, u.tgID)
			return
		}
		cmd := profile.Commands[st.Index]
		if st.Step >= len(cmd.Args) {
			u.h.botClearState(u.ctx, u.tgID)
			return
		}
		if st.Args == nil {
			st.Args = map[string]string{}
		}
		value := text
		if value == "-" && cmd.Args[st.Step].Optional {
			value = ""
		}
		st.Args[cmd.Args[st.Step].Name] = value
		st.Step++
		if st.Step < len(cmd.Args) {
			u.h.botSetState(u.ctx, u.tgID, st)
			u.send(u.argPrompt(cmd, st.Step), nil)
			return
		}
		u.h.botClearState(u.ctx, u.tgID)
		line, err := cmd.Render(st.Args)
		if err != nil {
			u.send(esc(err.Error()), notify.TelegramKeyboard{u.backToServer(st.ServerID)})
			return
		}
		out, ok := u.runConsole(st.ServerID, line)
		if !ok {
			u.send(u.t("notify.bot.denied"), notify.TelegramKeyboard{u.backToServer(st.ServerID)})
			return
		}
		u.send(u.outputBlock(cmd.Label, out), notify.TelegramKeyboard{
			{{Text: "⌨️ " + u.t("notify.bot.btn.console"), Data: notify.Callback(notify.CBConsole, st.ServerID)}},
			u.backToServer(st.ServerID),
		})
	default:
		u.h.botClearState(u.ctx, u.tgID)
	}
}

func (u *botUI) viewLogs(serverID string) {
	if _, ok := u.gate(serverID, "logs"); !ok {
		return
	}
	u.bot.SendChatAction(u.ctx, u.chatID, "typing")
	lines, err := u.h.serverLogLines(u.ctx, serverID, 40)
	kb := notify.TelegramKeyboard{
		{{Text: "🔃 " + u.t("notify.bot.btn.refresh"), Data: notify.Callback(notify.CBLogs, serverID)}},
		u.backToServer(serverID),
	}
	if err != nil {
		_, text := explainAgentError("logs", err)
		u.show(esc(text), kb)
		return
	}
	u.show(u.outputBlock(u.t("notify.bot.logs_title"), strings.Join(lines, "\n")), kb)
}

func (u *botUI) viewPlayers(serverID string) {
	if _, ok := u.gate(serverID, "metrics"); !ok {
		return
	}
	s, err := u.h.botLoadServer(u.ctx, serverID)
	if err != nil {
		u.answer(u.t("notify.bot.srv_missing"), true)
		return
	}
	kb := notify.TelegramKeyboard{
		{{Text: "🔃 " + u.t("notify.bot.btn.refresh"), Data: notify.Callback(notify.CBPlayers, serverID)}},
		u.backToServer(serverID),
	}
	if !u.h.serverRunning(u.ctx, serverID) {
		u.show(u.t("notify.bot.need_running"), kb)
		return
	}
	live := u.h.botGameQuery(u.ctx, s)
	if !live.OK {
		u.show(u.t("notify.bot.no_data"), kb)
		return
	}
	max := "—"
	if live.Max > 0 {
		max = fmt.Sprint(live.Max)
	}
	text := u.t("notify.bot.players_title", i18n.Params{"name": esc(s.Name), "online": live.Online, "max": max})
	if len(live.Players) == 0 {
		text += "\n\n" + u.t("notify.bot.no_players")
	} else {
		shown := live.Players
		if len(shown) > 50 {
			shown = shown[:50]
		}
		var b strings.Builder
		for _, name := range shown {
			b.WriteString("• " + esc(name) + "\n")
		}
		text += "\n\n" + b.String()
	}
	u.show(text, kb)
}

func sparkline(values []float64, ceiling float64) string {
	const bars = "▁▂▃▄▅▆▇█"
	runes := []rune(bars)
	var b strings.Builder
	for _, v := range values {
		ratio := v / ceiling
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		b.WriteRune(runes[int(ratio*float64(len(runes)-1)+0.5)])
	}
	return b.String()
}

func (u *botUI) viewMetrics(serverID string) {
	if _, ok := u.gate(serverID, "metrics"); !ok {
		return
	}
	s, err := u.h.botLoadServer(u.ctx, serverID)
	if err != nil {
		u.answer(u.t("notify.bot.srv_missing"), true)
		return
	}
	kb := notify.TelegramKeyboard{
		{{Text: "🔃 " + u.t("notify.bot.btn.refresh"), Data: notify.Callback(notify.CBMetrics, serverID)}},
		u.backToServer(serverID),
	}
	rows, err := u.h.dbOf(u.ctx).Query(u.ctx, `
		SELECT cpu_pct, mem_used_mb, mem_limit_mb
		FROM core.server_metric_points
		WHERE server_id = $1 AND ts >= now() - interval '1 hour'
		ORDER BY ts
	`, serverID)
	if err != nil {
		u.show(u.t("notify.bot.error"), kb)
		return
	}
	defer rows.Close()
	var cpu, mem []float64
	var memUsed, memLimit int
	for rows.Next() {
		var c float64
		var used, limit int
		if rows.Scan(&c, &used, &limit) != nil {
			continue
		}
		cpu = append(cpu, c)
		if limit > 0 {
			mem = append(mem, float64(used)*100/float64(limit))
		}
		memUsed, memLimit = used, limit
	}
	if len(cpu) == 0 {
		u.show(u.t("notify.bot.no_data"), kb)
		return
	}
	var sum, peak float64
	for _, v := range cpu {
		sum += v
		peak = max(peak, v)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", u.t("notify.bot.metrics_title", i18n.Params{"name": esc(s.Name)}))
	fmt.Fprintf(&b, "<b>CPU</b>: %.0f%% · %s %.0f%% · %s %.0f%%\n<code>%s</code>\n\n",
		cpu[len(cpu)-1], u.t("notify.bot.avg"), sum/float64(len(cpu)), u.t("notify.bot.peak"), peak,
		sparkline(downsample(cpu, botSparkPoints), 100))
	if memLimit > 0 {
		fmt.Fprintf(&b, "<b>RAM</b>: %d%% (%d / %d MB)\n<code>%s</code>\n",
			memUsed*100/memLimit, memUsed, memLimit, sparkline(downsample(mem, botSparkPoints), 100))
	}
	fmt.Fprintf(&b, "\n%s", u.t("notify.bot.metrics_hour"))
	u.show(b.String(), kb)
}

func downsample(values []float64, points int) []float64 {
	if len(values) <= points {
		return values
	}
	out := make([]float64, 0, points)
	step := float64(len(values)) / float64(points)
	for i := 0; i < points; i++ {
		from := int(float64(i) * step)
		to := int(float64(i+1) * step)
		if to <= from {
			to = from + 1
		}
		var sum float64
		for _, v := range values[from:to] {
			sum += v
		}
		out = append(out, sum/float64(to-from))
	}
	return out
}

type botBackupList struct {
	ServerID string   `json:"s"`
	Names    []string `json:"n"`
}

func botBackupKey(tgID int64) string {
	return fmt.Sprintf("tgbot:backups:%d", tgID)
}

func (u *botUI) viewBackups(serverID string) {
	if _, ok := u.gate(serverID, "backup_create"); !ok {
		return
	}
	s, err := u.h.botLoadServer(u.ctx, serverID)
	if err != nil {
		u.answer(u.t("notify.bot.srv_missing"), true)
		return
	}
	u.show(u.t("notify.bot.backups_title", i18n.Params{"name": esc(s.Name)}), notify.TelegramKeyboard{
		{{Text: "💾 " + u.t("notify.bot.btn.backup_make"), Data: notify.Callback(notify.CBBackupMake, serverID)}},
		{{Text: "📂 " + u.t("notify.bot.btn.backup_list"), Data: notify.Callback(notify.CBBackupList, serverID)}},
		u.backToServer(serverID),
	})
}

func (u *botUI) actionBackupMake(serverID string) {
	if _, ok := u.gate(serverID, "backup_create"); !ok {
		return
	}
	if u.h.freezeActive(u.ctx) {
		u.answer(u.t("notify.bot.frozen"), true)
		return
	}
	if !u.h.serverRunning(u.ctx, serverID) {
		u.answer(u.t("notify.bot.need_running"), true)
		return
	}
	u.show("⏳ "+u.t("notify.bot.backup_working"), nil)
	bg, cancel := u.detach(backupWait + 2*time.Minute)
	go func() {
		defer cancel()
		name := "tg-" + time.Now().Format("20060102-1504")
		_, filename, size, err := bg.h.createServerBackup(bg.ctx, serverID, name, "telegram")
		kb := notify.TelegramKeyboard{
			{{Text: "📂 " + bg.t("notify.bot.btn.backup_list"), Data: notify.Callback(notify.CBBackupList, serverID)}},
			bg.backToServer(serverID),
		}
		audit(bg.ctx, bg.h.dbOf(bg.ctx), bg.user.UserID, "server.backup_telegram", "server:"+serverID, map[string]any{"ok": err == nil})
		if err != nil {
			_, text := explainAgentError("backup_create", err)
			bg.show("❌ "+bg.t("notify.bot.backup_failed", i18n.Params{"reason": esc(text)}), kb)
			return
		}
		bg.show("✅ "+bg.t("notify.bot.backup_done", i18n.Params{"file": esc(filename), "size": humanBytes(size)}), kb)
	}()
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	value := float64(n)
	for _, suffix := range []string{"KB", "MB", "GB", "TB"} {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.1f %s", value, suffix)
		}
	}
	return fmt.Sprintf("%.1f PB", value/unit)
}

func (u *botUI) viewBackupList(serverID string) {
	if _, ok := u.gate(serverID, "backup_create"); !ok {
		return
	}
	names, err := u.h.listServerBackups(u.ctx, serverID)
	kb := notify.TelegramKeyboard{}
	if err != nil {
		_, text := explainAgentError("files_list", err)
		u.show(esc(text), notify.TelegramKeyboard{u.backToServer(serverID)})
		return
	}
	if len(names) > botBackupsShown {
		names = names[:botBackupsShown]
	}
	_ = u.h.cache.SetJSON(u.ctx, botBackupKey(u.tgID), botBackupList{ServerID: serverID, Names: names}, botStateTTL)
	var b strings.Builder
	b.WriteString(u.t("notify.bot.backup_list_title") + "\n\n")
	if len(names) == 0 {
		b.WriteString(u.t("notify.bot.backup_empty"))
	}
	for i, name := range names {
		b.WriteString(fmt.Sprintf("%d. <code>%s</code>\n", i+1, esc(name)))
		kb = append(kb, []notify.TelegramButton{{
			Text: fmt.Sprintf("↩️ %d. %s", i+1, shorten(name, 30)),
			Data: notify.Callback(notify.CBBackupAsk, serverID, fmt.Sprint(i)),
		}})
	}
	kb = append(kb, []notify.TelegramButton{{Text: "💾 " + u.t("notify.bot.btn.backup_make"), Data: notify.Callback(notify.CBBackupMake, serverID)}})
	kb = append(kb, u.backToServer(serverID))
	u.show(b.String(), kb)
}

func (u *botUI) backupNameAt(serverID string, idx int) (string, bool) {
	var list botBackupList
	found, err := u.h.cache.GetJSON(u.ctx, botBackupKey(u.tgID), &list)
	if err != nil || !found || list.ServerID != serverID || idx < 0 || idx >= len(list.Names) {
		u.answer(u.t("notify.bot.list_stale"), true)
		return "", false
	}
	return list.Names[idx], true
}

func (u *botUI) viewBackupAsk(serverID string, idx int) {
	if _, ok := u.gate(serverID, "backup_restore"); !ok {
		return
	}
	name, ok := u.backupNameAt(serverID, idx)
	if !ok {
		return
	}
	u.show(u.t("notify.bot.backup_restore_ask", i18n.Params{"file": esc(name)}), notify.TelegramKeyboard{
		{
			{Text: "✅ " + u.t("notify.bot.btn.yes"), Data: notify.Callback(notify.CBBackupApply, serverID, fmt.Sprint(idx))},
			{Text: "✖️ " + u.t("notify.bot.btn.no"), Data: notify.Callback(notify.CBBackupList, serverID)},
		},
	})
}

func (u *botUI) actionBackupRestore(serverID string, idx int) {
	if _, ok := u.gate(serverID, "backup_restore"); !ok {
		return
	}
	if u.h.freezeActive(u.ctx) {
		u.answer(u.t("notify.bot.frozen"), true)
		return
	}
	name, ok := u.backupNameAt(serverID, idx)
	if !ok {
		return
	}
	u.show("⏳ "+u.t("notify.bot.restore_working"), nil)
	bg, cancel := u.detach(restoreWait + 2*time.Minute)
	go func() {
		defer cancel()
		err := bg.h.restoreServerBackup(bg.ctx, serverID, name)
		audit(bg.ctx, bg.h.dbOf(bg.ctx), bg.user.UserID, "server.restore_telegram", "server:"+serverID, map[string]any{"file": name, "ok": err == nil})
		if err != nil {
			_, text := explainAgentError("backup_restore", err)
			bg.show("❌ "+bg.t("notify.bot.restore_failed", i18n.Params{"reason": esc(text)}), notify.TelegramKeyboard{bg.backToServer(serverID)})
			return
		}
		bg.show("✅ "+bg.t("notify.bot.restore_done", i18n.Params{"file": esc(name)}), notify.TelegramKeyboard{bg.backToServer(serverID)})
	}()
}
