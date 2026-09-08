package notify

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DB — то, что нужно от подключения. Интерфейс, а не *pgxpool.Pool, чтобы
// оповещение можно было создать внутри уже открытой транзакции вызывающего:
// иначе запись в панели появлялась бы раньше, чем событие, о котором она
// сообщает, окончательно зафиксировано.
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Result — что получилось из одной отправки.
type Result struct {
	NotificationID string
	Channels       []Channel // каналы, поставленные в очередь
	Duplicate      bool      // событие отброшено по ключу повтора
}

// Dispatch создаёт уведомление в панели и ставит доставку во внешние каналы.
//
// Запись внутри панели появляется всегда — это единственное место, где клиент
// увидит событие наверняка. Внешние каналы добавляются сверх неё и только те,
// которые получатель включил И для которых указал адрес.
func Dispatch(ctx context.Context, db DB, tenantID string, r Recipient, e Event) (Result, error) {
	if strings.TrimSpace(e.Title) == "" {
		return Result{}, errors.New("notify: у события пустой заголовок")
	}
	if r.UserID == "" {
		return Result{}, errors.New("notify: не указан получатель")
	}

	metaJSON, err := json.Marshal(orEmpty(e.Meta))
	if err != nil {
		return Result{}, err
	}

	action := e.Action
	if !action.valid() {
		action = &Action{}
	}

	var id string
	err = db.QueryRow(ctx, `
		INSERT INTO core.notifications
			(tenant_id, user_id, type, title, body, meta, dedupe_key, action_label, action_href)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9)
		ON CONFLICT DO NOTHING
		RETURNING id::text
	`, tenantID, r.UserID, string(e.Kind), e.Title, e.Body, metaJSON,
		e.DedupeKey, action.Label, action.Href).Scan(&id)

	if errors.Is(err, pgx.ErrNoRows) {
		// Сработал ключ повтора: событие уже приходило этому получателю.
		return Result{Duplicate: true}, nil
	}
	if err != nil {
		return Result{}, err
	}

	channels := channelsFor(e, r)
	if len(channels) == 0 {
		return Result{NotificationID: id}, nil
	}

	subject, body := render(e)
	for _, c := range channels {
		target, ok := r.Target(c)
		if !ok {
			continue
		}
		if _, err := db.Exec(ctx, `
			INSERT INTO core.notification_deliveries
				(tenant_id, user_id, notification_id, kind, channel, target,
				 subject, body, action_label, action_href)
			VALUES ($1, $2, $3::uuid, $4, $5, $6, $7, $8, $9, $10)
		`, tenantID, r.UserID, id, string(e.Kind), string(c), target,
			subject, body, action.Label, action.Href); err != nil {
			// Уведомление в панели уже создано — терять его из-за сбоя постановки
			// в очередь незачем. Возвращаем ошибку, но с идентификатором, чтобы
			// вызывающий видел, что половина работы сделана.
			return Result{NotificationID: id}, err
		}
	}
	return Result{NotificationID: id, Channels: channels}, nil
}

// DispatchMany рассылает одно событие нескольким получателям.
//
// Отдельная функция, потому что массовая рассылка не должна прерываться на
// первом сбое: остановка локации касается сотен владельцев, и молчание для всех
// из-за одного плохого адреса хуже, чем частичная доставка.
func DispatchMany(ctx context.Context, db DB, tenantID string, rs []Recipient, e Event) (int, error) {
	sent := 0
	var firstErr error
	for _, r := range rs {
		res, err := Dispatch(ctx, db, tenantID, r, e)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if !res.Duplicate {
			sent++
		}
	}
	return sent, firstErr
}

// LoadRecipient собирает получателя с его настройками каналов.
//
// Отсутствие строки в core.user_notification_channels — не ошибка, а обычное
// состояние: настройки создаются при первом сохранении. Тогда действуют
// умолчания, те же, что стоят в схеме.
func LoadRecipient(ctx context.Context, db DB, tenantID, userID string) (Recipient, error) {
	r := Recipient{UserID: userID, Prefs: Prefs{Email: true}}
	err := db.QueryRow(ctx, `
		SELECT u.email,
		       COALESCE(c.email_enabled, true),
		       COALESCE(c.telegram_enabled, false),
		       COALESCE(c.discord_enabled, false),
		       COALESCE(c.telegram_chat_id, ''),
		       COALESCE(c.discord_webhook, '')
		FROM core.users u
		LEFT JOIN core.user_notification_channels c
		       ON c.user_id = u.id AND c.tenant_id = u.tenant_id
		WHERE u.id = $1 AND u.tenant_id = $2
	`, userID, tenantID).Scan(&r.Email, &r.Prefs.Email, &r.Prefs.Telegram,
		&r.Prefs.Discord, &r.Prefs.TelegramChatID, &r.Prefs.DiscordWebhook)
	if err != nil {
		return Recipient{}, err
	}
	return r, nil
}

// LoadServerOwner находит владельца сервера — самый частый адресат.
func LoadServerOwner(ctx context.Context, db DB, tenantID, serverID string) (Recipient, error) {
	var userID string
	err := db.QueryRow(ctx, `
		SELECT user_id::text FROM core.servers
		WHERE id = $1 AND tenant_id = $2 AND user_id IS NOT NULL
	`, serverID, tenantID).Scan(&userID)
	if err != nil {
		return Recipient{}, err
	}
	return LoadRecipient(ctx, db, tenantID, userID)
}

func orEmpty(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}
