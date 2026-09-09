package notify

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DB interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type Result struct {
	NotificationID string
	Channels       []Channel
	Duplicate      bool
}

func Dispatch(ctx context.Context, db DB, r Recipient, e Event) (Result, error) {
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
			( user_id, type, title, body, meta, dedupe_key, action_label, action_href)
		VALUES ( $1, $2, $3, $4, $5::jsonb, $6, $7, $8)
		ON CONFLICT DO NOTHING
		RETURNING id::text
	`, r.UserID, string(e.Kind), e.Title, e.Body, metaJSON,
		e.DedupeKey, action.Label, action.Href).Scan(&id)

	if errors.Is(err, pgx.ErrNoRows) {
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
				( user_id, notification_id, kind, channel, target,
				 subject, body, action_label, action_href)
			VALUES ( $1, $2::uuid, $3, $4, $5, $6, $7, $8, $9)
		`, r.UserID, id, string(e.Kind), string(c), target,
			subject, body, action.Label, action.Href); err != nil {
			return Result{NotificationID: id}, err
		}
	}
	return Result{NotificationID: id, Channels: channels}, nil
}

func DispatchMany(ctx context.Context, db DB, rs []Recipient, e Event) (int, error) {
	sent := 0
	var firstErr error
	for _, r := range rs {
		res, err := Dispatch(ctx, db, r, e)
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

func LoadRecipient(ctx context.Context, db DB, userID string) (Recipient, error) {
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
		       ON c.user_id = u.id
		WHERE u.id = $1
	`, userID).Scan(&r.Email, &r.Prefs.Email, &r.Prefs.Telegram,
		&r.Prefs.Discord, &r.Prefs.TelegramChatID, &r.Prefs.DiscordWebhook)
	if err != nil {
		return Recipient{}, err
	}
	return r, nil
}

func LoadServerOwner(ctx context.Context, db DB, serverID string) (Recipient, error) {
	var userID string
	err := db.QueryRow(ctx, `
		SELECT user_id::text FROM core.servers
		WHERE id = $1 AND user_id IS NOT NULL
	`, serverID).Scan(&userID)
	if err != nil {
		return Recipient{}, err
	}
	return LoadRecipient(ctx, db, userID)
}

func orEmpty(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}
