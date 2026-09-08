package handlers

import (
	"context"
	"log"

	"github.com/vortanixapp/panel/pkg/notify"
)

// Оповещения из обработчиков.
//
// Раньше здесь была одна функция h.notify, делавшая INSERT в core.notifications
// и на этом заканчивавшаяся: уведомление внутри панели никогда не превращалось
// ни в письмо, ни в сообщение в Telegram или Discord. Единственное исключение
// было сделано вручную и точечно — в подметателе дуннинга рядом стояли два
// вызова, notify и mailDunning.
//
// Теперь решение о каналах принимает справочник событий и настройки получателя,
// а места, где событие происходит, об этом не думают.

func (h *Handler) notifyConfig() notify.Config {
	return notify.Config{
		SMTPHost:         h.mail.Host,
		SMTPPort:         h.mail.Port,
		SMTPUser:         h.mail.User,
		SMTPPass:         h.mail.Pass,
		MailFrom:         h.mail.From,
		TelegramBotToken: h.telegramBotToken,
	}
}

// notifyUser отправляет событие одному пользователю.
//
// Ошибку не возвращаем: оповещение — это следствие уже случившегося действия, и
// проваливать из-за него сам запрос нельзя. Но и глушить нельзя — в логе должно
// остаться, что клиенту не сообщили.
func (h *Handler) notifyUser(ctx context.Context, tenantID, userID string, e notify.Event) {
	db := h.dbOf(ctx)
	r, err := notify.LoadRecipient(ctx, db, tenantID, userID)
	if err != nil {
		log.Printf("оповещение %s: не найден получатель %s: %v", e.Kind, userID, err)
		return
	}
	if _, err := notify.Dispatch(ctx, db, tenantID, r, e); err != nil {
		log.Printf("оповещение %s пользователю %s: %v", e.Kind, userID, err)
	}
}

// notifyServerOwner — самый частый адресат: владелец сервера.
func (h *Handler) notifyServerOwner(ctx context.Context, tenantID, serverID string, e notify.Event) {
	db := h.dbOf(ctx)
	r, err := notify.LoadServerOwner(ctx, db, tenantID, serverID)
	if err != nil {
		log.Printf("оповещение %s: не найден владелец сервера %s: %v", e.Kind, serverID, err)
		return
	}
	if _, err := notify.Dispatch(ctx, db, tenantID, r, e); err != nil {
		log.Printf("оповещение %s по серверу %s: %v", e.Kind, serverID, err)
	}
}

// serverAction — ссылка на сервер для кнопки в уведомлении.
//
// Адрес абсолютный: то же уведомление уходит письмом и в Telegram, а там
// относительная ссылка вида /servers/… никуда не ведёт. Это одна из причин,
// по которой действие раньше не работало даже там, где его пытались положить.
func (h *Handler) serverAction(label, serverID, suffix string) *notify.Action {
	base := h.frontendURL
	if base == "" {
		return nil
	}
	return &notify.Action{Label: label, Href: base + "/servers/" + serverID + suffix}
}

func (h *Handler) panelAction(label, path string) *notify.Action {
	if h.frontendURL == "" {
		return nil
	}
	return &notify.Action{Label: label, Href: h.frontendURL + path}
}
