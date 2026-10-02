package handlers

import (
	"fmt"

	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

func (u *botUI) viewBalance() {
	rows, err := u.h.dbOf(u.ctx).Query(u.ctx, `
		SELECT currency, balance::float8
		FROM core.wallets
		WHERE user_id = $1::uuid
		ORDER BY is_default DESC, currency
	`, u.user.UserID)
	kb := notify.TelegramKeyboard{}
	if link := u.h.botPanelURL("/billing"); link != "" {
		kb = append(kb, []notify.TelegramButton{{Text: "💳 " + u.t("notify.bot.btn.topup"), URL: link}})
	}
	kb = append(kb, u.backToMenu())
	if err != nil {
		u.show(u.t("notify.bot.error"), kb)
		return
	}
	defer rows.Close()
	text := u.t("notify.bot.balance_title") + "\n"
	found := false
	for rows.Next() {
		var currency string
		var balance float64
		if rows.Scan(&currency, &balance) != nil {
			continue
		}
		found = true
		text += fmt.Sprintf("\n💰 <b>%.2f %s</b>", balance, esc(currency))
	}
	if !found {
		text += "\n" + u.t("notify.bot.balance_none")
	}
	u.show(text, kb)
}

func (u *botUI) viewSettings() {
	var chat string
	_ = u.h.dbOf(u.ctx).QueryRow(u.ctx, `
		SELECT COALESCE(telegram_chat_id, '') FROM core.user_notification_channels WHERE user_id = $1::uuid
	`, u.user.UserID).Scan(&chat)
	kb := notify.TelegramKeyboard{}
	if link := u.h.botPanelURL("/settings?tab=notifications"); link != "" {
		kb = append(kb, []notify.TelegramButton{{Text: "🔔 " + u.t("notify.bot.btn.notify_settings"), URL: link}})
	}
	kb = append(kb,
		[]notify.TelegramButton{{Text: "🔌 " + u.t("notify.bot.btn.unlink"), Data: notify.Callback(notify.CBUnlink)}},
		u.backToMenu(),
	)
	u.show(u.t("notify.bot.settings_text", i18n.Params{"chat": esc(chat)}), kb)
}

func (u *botUI) viewUnlinkAsk() {
	u.show(u.t("notify.bot.unlink_ask"), notify.TelegramKeyboard{
		{
			{Text: "✅ " + u.t("notify.bot.btn.yes"), Data: notify.Callback(notify.CBUnlinkApply)},
			{Text: "✖️ " + u.t("notify.bot.btn.no"), Data: notify.Callback(notify.CBSettings)},
		},
	})
}

func (u *botUI) actionUnlink() {
	_, err := u.h.dbOf(u.ctx).Exec(u.ctx, `
		UPDATE core.user_notification_channels
		SET telegram_enabled = false, telegram_chat_id = '', telegram_user_id = NULL,
		    telegram_verified_at = NULL, telegram_control = false, updated_at = now()
		WHERE user_id = $1::uuid
	`, u.user.UserID)
	if err != nil {
		u.answer(u.t("notify.bot.error"), true)
		return
	}
	u.h.botClearState(u.ctx, u.tgID)
	audit(u.ctx, u.h.dbOf(u.ctx), u.user.UserID, "account.telegram_unlinked", "user:"+u.user.UserID, map[string]any{"source": "bot"})
	u.show(u.t("notify.bot.unlinked_done"), nil)
}
