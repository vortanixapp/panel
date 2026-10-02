package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"html"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

const (
	botPollSeconds   = 25
	botStateTTL      = 10 * time.Minute
	botActionsPerMin = 40
	botOffsetKey     = "telegram.bot.offset"
	botLinkDoneTTL   = 5 * time.Minute
)

var (
	botUserLocks sync.Map
	uuidPattern  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	linkCodeExpr = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)
)

type botAccount struct {
	UserID  string
	Role    string
	Control bool
}

type botState struct {
	Kind     string            `json:"k"`
	ServerID string            `json:"s"`
	Index    int               `json:"i"`
	Step     int               `json:"p"`
	Args     map[string]string `json:"a"`
}

func telegramLinkDoneKey(code string) string {
	return "tgbot:linked:" + code
}

func botStateKey(tgID int64) string {
	return "tgbot:state:" + strconv.FormatInt(tgID, 10)
}

func (h *Handler) StartTelegramBot(ctx context.Context) {
	go h.leaderLoop(ctx, lockTelegramBot, h.telegramBotRun)
}

func tokenFingerprint(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:6])
}

func (h *Handler) botLoadOffset(ctx context.Context, token string) int64 {
	var raw []byte
	if h.dbOf(ctx).QueryRow(ctx, `SELECT value FROM core.tenant_settings WHERE key = $1`, botOffsetKey).Scan(&raw) != nil {
		return 0
	}
	var stored struct {
		Token  string `json:"token"`
		Offset int64  `json:"offset"`
	}
	if json.Unmarshal(raw, &stored) != nil || stored.Token != tokenFingerprint(token) {
		return 0
	}
	return stored.Offset
}

func (h *Handler) botSaveOffset(ctx context.Context, token string, offset int64) {
	raw, err := json.Marshal(map[string]any{"token": tokenFingerprint(token), "offset": offset})
	if err != nil {
		return
	}
	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.tenant_settings (key, value) VALUES ($1, $2::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, botOffsetKey, string(raw))
}

func (h *Handler) telegramBotRun(ctx context.Context) {
	lastToken := ""
	for ctx.Err() == nil {
		token := h.notifyTelegramToken(ctx)
		if token == "" {
			lastToken = ""
			sleepCtx(ctx, 30*time.Second)
			continue
		}
		bot := notify.NewTelegramBot(token)
		if token != lastToken {
			h.botSetup(ctx, bot)
			lastToken = token
		}

		updates, err := bot.GetUpdates(ctx, h.botLoadOffset(ctx, token), botPollSeconds)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			switch {
			case notify.IsTelegramWebhookConflict(err):
				log.Printf("telegram: у бота настроен вебхук, он будет удалён, чтобы бот мог принимать сообщения")
				_ = bot.DeleteWebhook(ctx)
			case notify.IsTelegramUnauthorized(err):
				log.Printf("telegram: токен бота отклонён, проверка через 2 минуты")
				lastToken = ""
				sleepCtx(ctx, 2*time.Minute)
			case notify.IsTelegramConflict(err):
				sleepCtx(ctx, 10*time.Second)
			default:
				log.Printf("telegram: получение сообщений: %v", err)
				sleepCtx(ctx, 5*time.Second)
			}
			continue
		}
		if len(updates) == 0 {
			continue
		}
		var last int64
		for _, u := range updates {
			if u.UpdateID > last {
				last = u.UpdateID
			}
		}
		h.botSaveOffset(ctx, token, last+1)
		for _, u := range updates {
			go h.botSafeHandle(ctx, bot, u)
		}
	}
}

func (h *Handler) botSetup(ctx context.Context, bot notify.TelegramBot) {
	_ = bot.DeleteWebhook(ctx)
	for _, lang := range []string{"", "ru"} {
		loc := lang
		if loc == "" {
			loc = "en"
		}
		l := i18n.For(ctx, h.dbOf(ctx), loc)
		cmds := []notify.TelegramCommand{
			{Command: "menu", Description: l.T("notify.bot.cmd.menu")},
			{Command: "servers", Description: l.T("notify.bot.cmd.servers")},
			{Command: "balance", Description: l.T("notify.bot.cmd.balance")},
			{Command: "cancel", Description: l.T("notify.bot.cmd.cancel")},
			{Command: "help", Description: l.T("notify.bot.cmd.help")},
		}
		if err := bot.SetCommands(ctx, lang, cmds); err != nil {
			log.Printf("telegram: список команд бота не обновлён: %v", err)
		}
	}
}

func (h *Handler) botUserLock(tgID int64) *sync.Mutex {
	m, _ := botUserLocks.LoadOrStore(tgID, &sync.Mutex{})
	return m.(*sync.Mutex)
}

func (h *Handler) botSafeHandle(ctx context.Context, bot notify.TelegramBot, u notify.TelegramUpdate) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("telegram: сбой обработки сообщения: %v", r)
		}
	}()
	var tgID int64
	switch {
	case u.Message != nil && u.Message.From != nil:
		tgID = u.Message.From.ID
	case u.CallbackQuery != nil:
		tgID = u.CallbackQuery.From.ID
	default:
		return
	}
	lock := h.botUserLock(tgID)
	lock.Lock()
	defer lock.Unlock()
	handleCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if u.Message != nil {
		h.botMessage(handleCtx, bot, u.Message)
		return
	}
	h.botCallback(handleCtx, bot, u.CallbackQuery)
}

type botUI struct {
	h      *Handler
	ctx    context.Context
	bot    notify.TelegramBot
	user   *botAccount
	l      i18n.Localizer
	chatID int64
	msgID  int64
	cbID   string
	tgID   int64
	once   sync.Once
}

func (u *botUI) t(key string, params ...i18n.Params) string {
	return u.l.T(key, params...)
}

func (u *botUI) answer(text string, alert bool) {
	u.once.Do(func() {
		if u.cbID != "" {
			_ = u.bot.AnswerCallback(u.ctx, u.cbID, text, alert)
		}
	})
}

func (u *botUI) show(text string, kb notify.TelegramKeyboard) {
	u.answer("", false)
	if u.msgID != 0 {
		err := u.bot.EditMessage(u.ctx, u.chatID, u.msgID, text, kb)
		if err == nil {
			return
		}
		var te *notify.TelegramError
		if errors.As(err, &te) && te.Code != 400 {
			return
		}
	}
	u.send(text, kb)
}

func (u *botUI) send(text string, kb notify.TelegramKeyboard) int64 {
	id, err := u.bot.SendMessage(u.ctx, u.chatID, text, kb)
	if err != nil {
		log.Printf("telegram: сообщение не отправлено: %v", err)
		return 0
	}
	u.msgID = id
	return id
}

func esc(s string) string {
	return html.EscapeString(s)
}

func (h *Handler) botBrandName(ctx context.Context) string {
	if name := strings.TrimSpace(appearanceFromSettings(h.loadTenantSettingStrings(ctx)).Name); name != "" {
		return name
	}
	return "Vortanix"
}

func (h *Handler) botPanelURL(path string) string {
	base := strings.TrimRight(strings.TrimSpace(h.frontendURL), "/")
	if base == "" || !strings.HasPrefix(base, "https://") {
		return ""
	}
	return base + path
}

func (h *Handler) botAuth(ctx context.Context, tgID, chatID int64) (*botAccount, bool, error) {
	var acc botAccount
	var status string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT c.user_id::text, u.role, u.status, c.telegram_control
		FROM core.user_notification_channels c
		JOIN core.users u ON u.id = c.user_id
		WHERE c.telegram_user_id = $1
		  AND c.telegram_verified_at IS NOT NULL
		  AND c.telegram_chat_id = $2
	`, tgID, strconv.FormatInt(chatID, 10)).Scan(&acc.UserID, &acc.Role, &status, &acc.Control)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if status != "active" {
		return nil, false, nil
	}
	return &acc, true, nil
}

func parseBotCommand(text string) (string, string) {
	if !strings.HasPrefix(text, "/") {
		return "", text
	}
	head, rest, _ := strings.Cut(text, " ")
	head = strings.TrimPrefix(head, "/")
	if at := strings.Index(head, "@"); at >= 0 {
		head = head[:at]
	}
	return strings.ToLower(head), strings.TrimSpace(rest)
}

func (h *Handler) botPrepare(ctx context.Context, bot notify.TelegramBot, chat notify.TelegramChatInfo, from notify.TelegramUser, cbID string, msgID int64) (*botUI, bool) {
	ui := &botUI{h: h, ctx: ctx, bot: bot, chatID: chat.ID, msgID: msgID, cbID: cbID, tgID: from.ID}
	acc, linked, err := h.botAuth(ctx, from.ID, chat.ID)
	if err != nil {
		log.Printf("telegram: проверка аккаунта: %v", err)
		ui.l = i18n.For(ctx, h.dbOf(ctx), from.LanguageCode)
		ui.answer(ui.t("notify.bot.error"), true)
		return ui, false
	}
	if !linked {
		ui.l = i18n.For(ctx, h.dbOf(ctx), from.LanguageCode)
		ui.showUnlinked()
		return ui, false
	}
	ui.user = acc
	ui.l = i18n.ForUser(ctx, h.dbOf(ctx), acc.UserID)
	if !acc.Control {
		ui.showControlOff()
		return ui, false
	}
	if !h.allowAttempt(ctx, "tgbot:rate:"+strconv.FormatInt(from.ID, 10), botActionsPerMin, time.Minute) {
		ui.answer(ui.t("notify.bot.rate"), true)
		return ui, false
	}
	return ui, true
}

func (u *botUI) showUnlinked() {
	var kb notify.TelegramKeyboard
	if link := u.h.botPanelURL("/settings?tab=notifications"); link != "" {
		kb = notify.TelegramKeyboard{{{Text: u.t("notify.bot.open_settings"), URL: link}}}
	}
	text := u.t("notify.bot.unlinked", i18n.Params{"app": esc(u.h.botBrandName(u.ctx))})
	if u.cbID != "" {
		u.answer(u.t("notify.bot.unlinked_toast"), true)
	}
	u.send(text, kb)
}

func (u *botUI) showControlOff() {
	var kb notify.TelegramKeyboard
	if link := u.h.botPanelURL("/settings?tab=notifications"); link != "" {
		kb = notify.TelegramKeyboard{{{Text: u.t("notify.bot.open_settings"), URL: link}}}
	}
	if u.cbID != "" {
		u.answer(u.t("notify.bot.control_off_toast"), true)
	}
	u.send(u.t("notify.bot.control_off"), kb)
}

func (h *Handler) botMessage(ctx context.Context, bot notify.TelegramBot, m *notify.TelegramMessage) {
	if m.From == nil || m.From.IsBot {
		return
	}
	text := strings.TrimSpace(m.Text)
	if text == "" {
		return
	}
	cmd, arg := parseBotCommand(text)

	if cmd == "start" && linkCodeExpr.MatchString(arg) {
		h.botLink(ctx, bot, m, arg)
		return
	}
	if !m.Chat.Private() {
		return
	}

	ui, ok := h.botPrepare(ctx, bot, m.Chat, *m.From, "", 0)
	if !ok {
		return
	}

	switch cmd {
	case "start", "menu":
		h.botClearState(ctx, ui.tgID)
		ui.viewMenu()
	case "servers":
		h.botClearState(ctx, ui.tgID)
		ui.viewServers(0)
	case "balance":
		ui.viewBalance()
	case "cancel":
		h.botClearState(ctx, ui.tgID)
		ui.send(ui.t("notify.bot.cancelled"), ui.menuKeyboard())
	case "help":
		ui.viewHelp()
	case "":
		if st, ok := h.botGetState(ctx, ui.tgID); ok {
			ui.handleStateInput(st, text)
			return
		}
		ui.send(ui.t("notify.bot.hint"), ui.menuKeyboard())
	default:
		ui.send(ui.t("notify.bot.unknown"), ui.menuKeyboard())
	}
}

func (h *Handler) botCallback(ctx context.Context, bot notify.TelegramBot, cb *notify.TelegramCallback) {
	if cb.Message == nil || !cb.Message.Chat.Private() {
		_ = bot.AnswerCallback(ctx, cb.ID, "", false)
		return
	}
	ui, ok := h.botPrepare(ctx, bot, cb.Message.Chat, cb.From, cb.ID, cb.Message.MessageID)
	if !ok {
		ui.answer("", false)
		return
	}
	defer ui.answer("", false)

	verb, args := notify.ParseCallback(cb.Data)
	arg := func(i int) string {
		if i < len(args) {
			return args[i]
		}
		return ""
	}
	switch verb {
	case notify.CBMenu:
		h.botClearState(ctx, ui.tgID)
		ui.viewMenu()
	case notify.CBServers:
		page, _ := strconv.Atoi(arg(0))
		ui.viewServers(page)
	case notify.CBServer:
		h.botClearState(ctx, ui.tgID)
		ui.viewServer(arg(0), "")
	case notify.CBPower:
		ui.actionPower(arg(0), arg(1), arg(2) == "y")
	case notify.CBConsole:
		ui.viewConsole(arg(0))
	case notify.CBQuick:
		idx, _ := strconv.Atoi(arg(1))
		ui.actionQuick(arg(0), idx, arg(2) == "y")
	case notify.CBCustom:
		ui.actionCustomCommand(arg(0))
	case notify.CBLogs:
		ui.viewLogs(arg(0))
	case notify.CBPlayers:
		ui.viewPlayers(arg(0))
	case notify.CBMetrics:
		ui.viewMetrics(arg(0))
	case notify.CBBackup:
		ui.viewBackups(arg(0))
	case notify.CBBackupMake:
		ui.actionBackupMake(arg(0))
	case notify.CBBackupList:
		ui.viewBackupList(arg(0))
	case notify.CBBackupAsk:
		idx, _ := strconv.Atoi(arg(1))
		ui.viewBackupAsk(arg(0), idx)
	case notify.CBBackupApply:
		idx, _ := strconv.Atoi(arg(1))
		ui.actionBackupRestore(arg(0), idx)
	case notify.CBWipe:
		ui.viewWipes(arg(0))
	case notify.CBWipeNow:
		ui.viewWipeNow(arg(0), arg(1))
	case notify.CBWipeRun:
		idx, _ := strconv.Atoi(arg(1))
		minutes, _ := strconv.Atoi(arg(2))
		ui.actionWipeRun(arg(0), idx, minutes)
	case notify.CBWipeToggle:
		ui.actionWipeToggle(arg(0), arg(1), false)
	case notify.CBWipeSkip:
		ui.actionWipeToggle(arg(0), arg(1), true)
	case notify.CBWipeCancel:
		ui.actionWipeCancel(arg(0), arg(1))
	case notify.CBBalance:
		ui.viewBalance()
	case notify.CBSettings:
		ui.viewSettings()
	case notify.CBUnlink:
		ui.viewUnlinkAsk()
	case notify.CBUnlinkApply:
		ui.actionUnlink()
	case notify.CBCancel:
		h.botClearState(ctx, ui.tgID)
		ui.viewMenu()
	}
}

func (h *Handler) botGetState(ctx context.Context, tgID int64) (botState, bool) {
	var st botState
	found, err := h.cache.GetJSON(ctx, botStateKey(tgID), &st)
	if err != nil || !found {
		return botState{}, false
	}
	return st, true
}

func (h *Handler) botSetState(ctx context.Context, tgID int64, st botState) {
	_ = h.cache.SetJSON(ctx, botStateKey(tgID), st, botStateTTL)
}

func (h *Handler) botClearState(ctx context.Context, tgID int64) {
	_ = h.cache.Delete(ctx, botStateKey(tgID))
}

func (h *Handler) botLink(ctx context.Context, bot notify.TelegramBot, m *notify.TelegramMessage, code string) {
	l := i18n.For(ctx, h.dbOf(ctx), m.From.LanguageCode)
	chatID := m.Chat.ID

	reply := func(text string, kb notify.TelegramKeyboard) {
		if _, err := bot.SendMessage(ctx, chatID, text, kb); err != nil {
			log.Printf("telegram: ответ на подключение не отправлен: %v", err)
		}
	}

	var link telegramLink
	found, err := h.cache.GetJSON(ctx, telegramLinkKey(code), &link)
	if err != nil {
		reply(l.T("notify.bot.error"), nil)
		return
	}
	if !found || link.UserID == "" {
		reply(l.T("notify.bot.link_expired"), nil)
		return
	}

	private := m.Chat.Private()
	var tgUser any
	var verified any
	if private {
		tgUser = m.From.ID
		verified = time.Now()
	}

	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		reply(l.T("notify.bot.error"), nil)
		return
	}
	defer tx.Rollback(ctx)
	if private {
		if _, err := tx.Exec(ctx, `
			UPDATE core.user_notification_channels
			SET telegram_user_id = NULL, telegram_verified_at = NULL, telegram_control = false, updated_at = now()
			WHERE telegram_user_id = $1 AND user_id <> $2::uuid
		`, m.From.ID, link.UserID); err != nil {
			reply(l.T("notify.bot.error"), nil)
			return
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO core.user_notification_channels
			(user_id, telegram_enabled, telegram_chat_id, telegram_user_id, telegram_verified_at, telegram_control, updated_at)
		VALUES ($1::uuid, true, $2, $3, $4, $5, now())
		ON CONFLICT (user_id) DO UPDATE SET
			telegram_enabled     = true,
			telegram_chat_id     = EXCLUDED.telegram_chat_id,
			telegram_user_id     = EXCLUDED.telegram_user_id,
			telegram_verified_at = EXCLUDED.telegram_verified_at,
			telegram_control     = EXCLUDED.telegram_control,
			updated_at           = now()
	`, link.UserID, strconv.FormatInt(chatID, 10), tgUser, verified, private); err != nil {
		reply(l.T("notify.bot.error"), nil)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		reply(l.T("notify.bot.error"), nil)
		return
	}

	_ = h.cache.Delete(ctx, telegramLinkKey(code))
	_ = h.cache.SetJSON(ctx, telegramLinkDoneKey(code), map[string]string{
		"user_id": link.UserID,
		"chat":    m.Chat.DisplayName(),
	}, botLinkDoneTTL)
	audit(ctx, h.dbOf(ctx), link.UserID, "account.telegram_linked", "user:"+link.UserID, map[string]any{"private": private})

	ul := i18n.ForUser(ctx, h.dbOf(ctx), link.UserID)
	app := esc(h.botBrandName(ctx))
	if !private {
		reply(ul.T("notify.bot.linked_group", i18n.Params{"app": app}), nil)
		return
	}
	ui := &botUI{h: h, ctx: ctx, bot: bot, chatID: chatID, l: ul, tgID: m.From.ID,
		user: &botAccount{UserID: link.UserID, Control: true}}
	ui.send(ul.T("notify.bot.linked", i18n.Params{"app": app}), ui.menuKeyboard())
}
