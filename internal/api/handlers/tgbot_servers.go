package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
	"github.com/vortanixapp/panel/pkg/wipe"
)

const (
	botPageSize  = 6
	botQueryWait = 6 * time.Second
	botNameLimit = 26
)

type botServer struct {
	ID       string
	Name     string
	GameID   string
	GameName string
	Status   string
	Runtime  string
	Prov     string
	NodeID   string
	IP       string
	Port     int
	Expires  *time.Time
	Location string
	Tariff   string
	Blocked  bool
	Limits   map[string]any
}

func (u *botUI) loc() *time.Location {
	var tz string
	_ = u.h.dbOf(u.ctx).QueryRow(u.ctx, `
		SELECT COALESCE(timezone, '') FROM core.user_profiles WHERE user_id = $1
	`, u.user.UserID).Scan(&tz)
	if loc, err := time.LoadLocation(tz); err == nil && tz != "" {
		return loc
	}
	return time.UTC
}

func (u *botUI) date(t time.Time) string {
	if u.l.Locale() == "ru" {
		return t.In(u.loc()).Format("02.01.2006")
	}
	return t.In(u.loc()).Format("2006-01-02")
}

func (u *botUI) dateTime(t time.Time) string {
	if u.l.Locale() == "ru" {
		return t.In(u.loc()).Format("02.01 15:04")
	}
	return t.In(u.loc()).Format("01-02 15:04")
}

func botServerState(status, runtime, prov string, live *string) string {
	effective := resolveEffectiveStatus(status, runtime, live)
	switch strings.ToLower(prov) {
	case "pending", "provisioning", "installing":
		if effective != "running" {
			return "installing"
		}
	case "failed":
		return "error"
	case "reinstalling", "updating":
		return "updating"
	}
	switch effective {
	case "running", "stopped", "starting", "stopping", "installing", "updating", "error", "suspended":
		return effective
	case "reinstalling":
		return "updating"
	case "offline", "":
		return "stopped"
	case "failed":
		return "error"
	}
	return "other"
}

func botStateIcon(state string) string {
	switch state {
	case "running":
		return "🟢"
	case "stopped":
		return "⚪"
	case "starting", "stopping", "installing", "updating":
		return "🟡"
	case "error":
		return "🔴"
	case "suspended":
		return "⏸"
	}
	return "⚫"
}

func (h *Handler) botLiveState(ctx context.Context, id, status, runtime, prov string) string {
	var live *string
	if cached, ok := h.cache.GetServerStatus(ctx, id); ok {
		live = &cached
	}
	return botServerState(status, runtime, prov, live)
}

func shorten(s string, limit int) string {
	if utf8.RuneCountInString(s) <= limit {
		return s
	}
	r := []rune(s)
	return string(r[:limit-1]) + "…"
}

func (u *botUI) menuKeyboard() notify.TelegramKeyboard {
	kb := notify.TelegramKeyboard{
		{{Text: u.t("notify.bot.btn.servers"), Data: notify.Callback(notify.CBServers, "0")}},
		{
			{Text: u.t("notify.bot.btn.balance"), Data: notify.Callback(notify.CBBalance)},
			{Text: u.t("notify.bot.btn.settings"), Data: notify.Callback(notify.CBSettings)},
		},
	}
	if link := u.h.botPanelURL(""); link != "" {
		kb = append(kb, []notify.TelegramButton{{Text: u.t("notify.bot.btn.panel"), URL: link}})
	}
	return kb
}

func (u *botUI) viewMenu() {
	u.show(u.t("notify.bot.menu", i18n.Params{"app": esc(u.h.botBrandName(u.ctx))}), u.menuKeyboard())
}

func (u *botUI) viewHelp() {
	u.send(u.t("notify.bot.help"), u.menuKeyboard())
}

func (u *botUI) backToMenu() []notify.TelegramButton {
	return []notify.TelegramButton{{Text: u.t("notify.bot.btn.menu"), Data: notify.Callback(notify.CBMenu)}}
}

type botListItem struct {
	ID, Name, Game, Status, Runtime, Prov string
}

func (h *Handler) botListServers(ctx context.Context, userID string) ([]botListItem, error) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT s.id::text, s.name, COALESCE(g.name, s.game_id, ''), COALESCE(s.status, ''),
		       COALESCE(s.runtime_status, ''), COALESCE(s.provisioning_status, 'pending')
		FROM core.servers s
		LEFT JOIN core.games g ON g.slug = s.game_id
		WHERE s.user_id = $1::uuid
		   OR EXISTS (SELECT 1 FROM core.server_friends f WHERE f.server_id = s.id AND f.user_id = $1::uuid)
		ORDER BY lower(s.name), s.id
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []botListItem
	for rows.Next() {
		var it botListItem
		if err := rows.Scan(&it.ID, &it.Name, &it.Game, &it.Status, &it.Runtime, &it.Prov); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (u *botUI) viewServers(page int) {
	items, err := u.h.botListServers(u.ctx, u.user.UserID)
	if err != nil {
		u.show(u.t("notify.bot.error"), notify.TelegramKeyboard{u.backToMenu()})
		return
	}
	if len(items) == 0 {
		kb := notify.TelegramKeyboard{}
		if link := u.h.botPanelURL("/rent-server"); link != "" {
			kb = append(kb, []notify.TelegramButton{{Text: u.t("notify.bot.btn.rent"), URL: link}})
		}
		kb = append(kb, u.backToMenu())
		u.show(u.t("notify.bot.no_servers"), kb)
		return
	}
	pages := (len(items) + botPageSize - 1) / botPageSize
	page = min(max(page, 0), pages-1)
	from := page * botPageSize
	to := min(from+botPageSize, len(items))

	var kb notify.TelegramKeyboard
	for _, it := range items[from:to] {
		state := u.h.botLiveState(u.ctx, it.ID, it.Status, it.Runtime, it.Prov)
		label := botStateIcon(state) + " " + shorten(it.Name, botNameLimit)
		if it.Game != "" {
			label += " · " + shorten(it.Game, 14)
		}
		kb = append(kb, []notify.TelegramButton{{Text: label, Data: notify.Callback(notify.CBServer, it.ID)}})
	}
	if pages > 1 {
		nav := []notify.TelegramButton{}
		if page > 0 {
			nav = append(nav, notify.TelegramButton{Text: "◀️", Data: notify.Callback(notify.CBServers, fmt.Sprint(page-1))})
		}
		nav = append(nav, notify.TelegramButton{Text: fmt.Sprintf("%d / %d", page+1, pages), Data: notify.Callback(notify.CBServers, fmt.Sprint(page))})
		if page < pages-1 {
			nav = append(nav, notify.TelegramButton{Text: "▶️", Data: notify.Callback(notify.CBServers, fmt.Sprint(page+1))})
		}
		kb = append(kb, nav)
	}
	kb = append(kb, u.backToMenu())
	u.show(u.t("notify.bot.servers_title", i18n.Params{"count": len(items)}), kb)
}

func (u *botUI) gate(serverID, action string) (*serverAccess, bool) {
	if !uuidPattern.MatchString(serverID) {
		u.answer(u.t("notify.bot.srv_missing"), true)
		return nil, false
	}
	access, err := u.h.resolveServerAccess(u.ctx, u.user.UserID, u.user.Role, serverID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			u.answer(u.t("notify.bot.srv_missing"), true)
		} else {
			u.answer(u.t("notify.bot.error"), true)
		}
		return nil, false
	}
	if action != "" {
		if status, _ := accessDenial(access, action); status != 0 {
			u.answer(u.t("notify.bot.denied"), true)
			return nil, false
		}
	}
	return access, true
}

func (h *Handler) botLoadServer(ctx context.Context, serverID string) (*botServer, error) {
	var s botServer
	var limits []byte
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT s.id::text, s.name, COALESCE(s.game_id, ''), COALESCE(g.name, s.game_id, ''),
		       COALESCE(s.status, ''), COALESCE(s.runtime_status, ''), COALESCE(s.provisioning_status, 'pending'),
		       COALESCE(s.node_id::text, ''), COALESCE(s.ip_address, ''), COALESCE(s.primary_port, 0),
		       s.expires_at, COALESCE(n.name, ''), COALESCE(t.name, ''), COALESCE(s.is_blocked, false),
		       COALESCE(s.limits, '{}'::jsonb)
		FROM core.servers s
		LEFT JOIN core.games g ON g.slug = s.game_id
		LEFT JOIN core.nodes n ON n.id = s.node_id
		LEFT JOIN core.tariffs t ON t.id = s.tariff_id
		WHERE s.id = $1
	`, serverID).Scan(&s.ID, &s.Name, &s.GameID, &s.GameName, &s.Status, &s.Runtime, &s.Prov,
		&s.NodeID, &s.IP, &s.Port, &s.Expires, &s.Location, &s.Tariff, &s.Blocked, &limits)
	if err != nil {
		return nil, err
	}
	s.Limits = map[string]any{}
	_ = json.Unmarshal(limits, &s.Limits)
	return &s, nil
}

type botLive struct {
	Online  int
	Max     int
	Players []string
	OK      bool
}

func (h *Handler) botGameQuery(ctx context.Context, s *botServer) botLive {
	if s.NodeID == "" {
		return botLive{}
	}
	qctx, cancel := context.WithTimeout(ctx, botQueryWait)
	defer cancel()
	res, err := h.agentCommand(qctx, s.NodeID, s.ID, "game_query", map[string]any{
		"game_id": s.GameID,
		"limits":  s.Limits,
		"port":    s.Port,
	})
	if err != nil || res == nil {
		return botLive{}
	}
	live := botLive{OK: true}
	if v, ok := res["online_players"].(float64); ok {
		live.Online = int(v)
	}
	if v, ok := res["max_players"].(float64); ok {
		live.Max = int(v)
	}
	if list, ok := res["players_online"].([]any); ok {
		for _, item := range list {
			switch p := item.(type) {
			case string:
				live.Players = append(live.Players, p)
			case map[string]any:
				if name, _ := p["name"].(string); name != "" {
					live.Players = append(live.Players, name)
				}
			}
		}
		if live.Online == 0 {
			live.Online = len(live.Players)
		}
	}
	return live
}

type botMetric struct {
	CPU      float64
	MemUsed  int
	MemLimit int
	At       time.Time
}

func (h *Handler) botLastMetric(ctx context.Context, serverID string) (botMetric, bool) {
	var m botMetric
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT cpu_pct, mem_used_mb, mem_limit_mb, ts
		FROM core.server_metric_points
		WHERE server_id = $1 AND ts >= now() - interval '15 minutes'
		ORDER BY ts DESC LIMIT 1
	`, serverID).Scan(&m.CPU, &m.MemUsed, &m.MemLimit, &m.At)
	return m, err == nil
}

func (u *botUI) viewServer(serverID, note string) {
	access, ok := u.gate(serverID, "")
	if !ok {
		return
	}
	s, err := u.h.botLoadServer(u.ctx, serverID)
	if err != nil {
		u.answer(u.t("notify.bot.srv_missing"), true)
		return
	}
	allowed := func(action string) bool {
		status, _ := accessDenial(access, action)
		return status == 0
	}
	state := u.h.botLiveState(u.ctx, s.ID, s.Status, s.Runtime, s.Prov)

	var b strings.Builder
	fmt.Fprintf(&b, "%s <b>%s</b>\n", botStateIcon(state), esc(s.Name))
	meta := s.GameName
	if s.Location != "" {
		meta += " · " + s.Location
	}
	if meta != "" {
		fmt.Fprintf(&b, "%s\n", esc(meta))
	}
	fmt.Fprintf(&b, "%s\n", u.t("notify.bot.st."+state))
	if s.Blocked {
		fmt.Fprintf(&b, "🔒 %s\n", u.t("notify.bot.blocked"))
	}
	if s.IP != "" && s.Port > 0 {
		fmt.Fprintf(&b, "\n📍 <code>%s:%d</code>\n", esc(s.IP), s.Port)
	}
	if state == "running" {
		live := u.h.botGameQuery(u.ctx, s)
		if live.OK {
			max := "—"
			if live.Max > 0 {
				max = fmt.Sprint(live.Max)
			}
			fmt.Fprintf(&b, "👥 %s: %d / %s\n", u.t("notify.bot.players"), live.Online, max)
		}
		if m, ok := u.h.botLastMetric(u.ctx, s.ID); ok {
			ram := ""
			if m.MemLimit > 0 {
				ram = fmt.Sprintf(" · RAM %d%% (%d/%d MB)", m.MemUsed*100/m.MemLimit, m.MemUsed, m.MemLimit)
			}
			fmt.Fprintf(&b, "⚙️ CPU %.0f%%%s\n", m.CPU, ram)
		}
	}
	if s.Expires != nil {
		days := int(time.Until(*s.Expires).Hours() / 24)
		left := u.t("notify.bot.days_left", i18n.Params{"days": days})
		if days < 0 {
			left = u.t("notify.bot.expired")
		}
		fmt.Fprintf(&b, "📅 %s %s (%s)\n", u.t("notify.bot.until"), u.date(*s.Expires), left)
	}
	if note != "" {
		fmt.Fprintf(&b, "\n%s\n", note)
	}

	var kb notify.TelegramKeyboard
	var power []notify.TelegramButton
	switch state {
	case "running":
		if allowed("power_restart") {
			power = append(power, notify.TelegramButton{Text: "🔄 " + u.t("notify.bot.btn.restart"), Data: notify.Callback(notify.CBPower, s.ID, "r")})
		}
		if allowed("power_stop") {
			power = append(power, notify.TelegramButton{Text: "⏹ " + u.t("notify.bot.btn.stop"), Data: notify.Callback(notify.CBPower, s.ID, "t")})
		}
	case "stopped", "error", "other":
		if allowed("power_start") {
			power = append(power, notify.TelegramButton{Text: "▶️ " + u.t("notify.bot.btn.start"), Data: notify.Callback(notify.CBPower, s.ID, "s")})
		}
		if state == "error" && allowed("power_restart") {
			power = append(power, notify.TelegramButton{Text: "🔄 " + u.t("notify.bot.btn.restart"), Data: notify.Callback(notify.CBPower, s.ID, "r")})
		}
	case "starting", "stopping", "installing", "updating":
		if allowed("power_kill") {
			power = append(power, notify.TelegramButton{Text: "☠️ " + u.t("notify.bot.btn.kill"), Data: notify.Callback(notify.CBPower, s.ID, "k")})
		}
	}
	if len(power) > 0 {
		kb = append(kb, power)
	}
	var tools []notify.TelegramButton
	if allowed("console_command") {
		tools = append(tools, notify.TelegramButton{Text: "⌨️ " + u.t("notify.bot.btn.console"), Data: notify.Callback(notify.CBConsole, s.ID)})
	}
	if allowed("logs") {
		tools = append(tools, notify.TelegramButton{Text: "📋 " + u.t("notify.bot.btn.logs"), Data: notify.Callback(notify.CBLogs, s.ID)})
	}
	if len(tools) > 0 {
		kb = append(kb, tools)
	}
	var more []notify.TelegramButton
	if allowed("metrics") {
		more = append(more,
			notify.TelegramButton{Text: "👥 " + u.t("notify.bot.btn.players"), Data: notify.Callback(notify.CBPlayers, s.ID)},
			notify.TelegramButton{Text: "📈 " + u.t("notify.bot.btn.metrics"), Data: notify.Callback(notify.CBMetrics, s.ID)},
		)
	}
	if len(more) > 0 {
		kb = append(kb, more)
	}
	var ops []notify.TelegramButton
	if allowed("backup_create") {
		ops = append(ops, notify.TelegramButton{Text: "💾 " + u.t("notify.bot.btn.backups"), Data: notify.Callback(notify.CBBackup, s.ID)})
	}
	if wipe.Supported(s.GameID) && allowed("wipes_read") {
		ops = append(ops, notify.TelegramButton{Text: "🧹 " + u.t("notify.bot.btn.wipes"), Data: notify.Callback(notify.CBWipe, s.ID)})
	}
	if len(ops) > 0 {
		kb = append(kb, ops)
	}
	nav := []notify.TelegramButton{
		{Text: "⬅️ " + u.t("notify.bot.btn.list"), Data: notify.Callback(notify.CBServers, "0")},
		{Text: "🔃 " + u.t("notify.bot.btn.refresh"), Data: notify.Callback(notify.CBServer, s.ID)},
	}
	kb = append(kb, nav)
	if link := u.h.botPanelURL("/servers/" + s.ID); link != "" {
		kb = append(kb, []notify.TelegramButton{{Text: "🌐 " + u.t("notify.bot.btn.open_panel"), URL: link}})
	}
	u.show(b.String(), kb)
}

var botPowerActions = map[string]string{"s": "start", "t": "stop", "r": "restart", "k": "kill"}

func (u *botUI) actionPower(serverID, code string, confirmed bool) {
	action, known := botPowerActions[code]
	if !known {
		return
	}
	if _, ok := u.gate(serverID, "power_"+action); !ok {
		return
	}
	if u.h.freezeActive(u.ctx) {
		u.answer(u.t("notify.bot.frozen"), true)
		return
	}
	s, err := u.h.botLoadServer(u.ctx, serverID)
	if err != nil {
		u.answer(u.t("notify.bot.srv_missing"), true)
		return
	}
	if (action == "stop" || action == "kill") && !confirmed {
		u.show(u.t("notify.bot.confirm_"+action, i18n.Params{"name": esc(s.Name)}), notify.TelegramKeyboard{
			{
				{Text: "✅ " + u.t("notify.bot.btn.yes"), Data: notify.Callback(notify.CBPower, serverID, code, "y")},
				{Text: "✖️ " + u.t("notify.bot.btn.no"), Data: notify.Callback(notify.CBServer, serverID)},
			},
		})
		return
	}

	_, _, err = u.h.sendServerPower(u.ctx, u.user.UserID, serverID, action)
	if err != nil {
		var key string
		switch {
		case errors.Is(err, errServerNotFound):
			key = "notify.bot.srv_missing"
		case errors.Is(err, errServerExpired):
			key = "notify.bot.err_expired"
		case errors.Is(err, errAgentUnreachable):
			key = "notify.bot.err_agent"
		case errors.Is(err, errNoFreePort):
			key = "notify.bot.err_port"
		default:
			key = "notify.bot.error"
		}
		u.answer(u.t(key), true)
		return
	}
	audit(u.ctx, u.h.dbOf(u.ctx), u.user.UserID, "server.power_telegram", "server:"+serverID, map[string]any{"action": action})
	u.answer(u.t("notify.bot.sent"), false)
	u.viewServer(serverID, "⏳ "+u.t("notify.bot.working_"+action))
	u.watchPower(serverID, action)
}

func (u *botUI) watchPower(serverID, action string) {
	want := "running"
	if action == "stop" || action == "kill" {
		want = "stopped"
	}
	watcher := &botUI{h: u.h, bot: u.bot, user: u.user, l: u.l, chatID: u.chatID, msgID: u.msgID, tgID: u.tgID}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		watcher.ctx = ctx
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			s, err := watcher.h.botLoadServer(ctx, serverID)
			if err != nil {
				return
			}
			state := watcher.h.botLiveState(ctx, s.ID, s.Status, s.Runtime, s.Prov)
			if state == want || state == "error" {
				watcher.viewServer(serverID, "")
				return
			}
		}
	}()
}
