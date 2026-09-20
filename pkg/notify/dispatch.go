package notify

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/vortanixapp/panel/pkg/i18n"
)

const LiveChannel = "vx_notifications"

var (
	StaffSupport = []string{"support", "admin", "owner"}
	StaffAdmins  = []string{"admin", "owner"}
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
	if r.UserID == "" {
		return Result{}, errors.New("notify: не указан получатель")
	}

	l := i18n.For(ctx, db, r.Locale)
	title := strings.TrimSpace(l.Text(e.Title))
	if title == "" {
		return Result{}, errors.New("notify: у события пустой заголовок")
	}
	body := l.Paragraphs(append([]i18n.Msg{e.Body}, e.Extra...)...)
	label, href := "", ""
	if e.Action != nil {
		label = strings.TrimSpace(l.Text(e.Action.Label))
		href = strings.TrimSpace(e.Action.Href)
	}
	if label == "" || href == "" {
		label, href = "", ""
	}

	metaJSON, err := json.Marshal(orEmpty(e.Meta))
	if err != nil {
		return Result{}, err
	}

	var id string
	err = db.QueryRow(ctx, `
		INSERT INTO core.notifications
			( user_id, type, title, body, meta, dedupe_key, action_label, action_href)
		VALUES ( $1, $2, $3, $4, $5::jsonb, $6, $7, $8)
		ON CONFLICT DO NOTHING
		RETURNING id::text
	`, r.UserID, string(e.Kind), title, body, metaJSON,
		e.DedupeKey, label, href).Scan(&id)

	if errors.Is(err, pgx.ErrNoRows) {
		return Result{Duplicate: true}, nil
	}
	if err != nil {
		return Result{}, err
	}
	publish(ctx, db, r.UserID+":"+id)

	routes := routesFor(e, r)
	if len(routes) == 0 {
		return Result{NotificationID: id}, nil
	}

	subject, text := render(title, body)
	var holdUntil *time.Time
	if until, held := r.Quiet.HoldUntil(time.Now(), DefFor(e.Kind)); held {
		holdUntil = &until
	}
	channels := make([]Channel, 0, len(routes))
	for _, rt := range routes {
		if _, err := db.Exec(ctx, `
			INSERT INTO core.notification_deliveries
				( user_id, notification_id, kind, channel, target,
				 subject, body, action_label, action_href, next_attempt_at)
			VALUES ( $1, $2::uuid, $3, $4, $5, $6, $7, $8, $9, COALESCE($10::timestamptz, now()))
		`, r.UserID, id, string(e.Kind), string(rt.channel), rt.target,
			subject, text, label, href, holdUntil); err != nil {
			return Result{NotificationID: id, Channels: channels}, err
		}
		channels = append(channels, rt.channel)
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

func DispatchStaff(ctx context.Context, db DB, roles []string, exclude string, e Event) (int, error) {
	rs, err := LoadStaff(ctx, db, roles, exclude)
	if err != nil {
		return 0, err
	}
	return DispatchMany(ctx, db, rs, e)
}

func PublishSync(ctx context.Context, db DB, userID string) {
	publish(ctx, db, userID+":sync")
}

func publish(ctx context.Context, db DB, payload string) {
	_, _ = db.Exec(ctx, `SELECT pg_notify($1, $2)`, LiveChannel, payload)
}

func LoadRecipient(ctx context.Context, db DB, userID string) (Recipient, error) {
	r := Recipient{UserID: userID, Prefs: Prefs{Email: true}, StatusEmail: true}
	var routes []byte
	err := db.QueryRow(ctx, `
		SELECT u.email,
		       COALESCE(p.locale, ''),
		       COALESCE(c.email_enabled, true),
		       COALESCE(c.telegram_enabled, false),
		       COALESCE(c.discord_enabled, false),
		       COALESCE(c.telegram_chat_id, ''),
		       COALESCE(c.discord_webhook, ''),
		       COALESCE(c.routes, '{}'::jsonb),
		       COALESCE(c.quiet_enabled, false),
		       COALESCE(c.quiet_from, 1380),
		       COALESCE(c.quiet_to, 480),
		       COALESCE(c.quiet_critical, true),
		       COALESCE(NULLIF(p.timezone, ''), NULLIF(c.quiet_tz, ''), ''),
		       COALESCE((SELECT s.value #>> '{}' FROM core.tenant_settings s
		                 WHERE s.key = 'vtx_mail.server_status_notifications'), '1') <> '0'
		FROM core.users u
		LEFT JOIN core.user_profiles p
		       ON p.user_id = u.id
		LEFT JOIN core.user_notification_channels c
		       ON c.user_id = u.id
		WHERE u.id = $1
	`, userID).Scan(&r.Email, &r.Locale, &r.Prefs.Email, &r.Prefs.Telegram,
		&r.Prefs.Discord, &r.Prefs.TelegramChatID, &r.Prefs.DiscordWebhook, &routes,
		&r.Quiet.Enabled, &r.Quiet.From, &r.Quiet.To, &r.Quiet.Critical, &r.Quiet.TimeZone,
		&r.StatusEmail)
	if err != nil {
		return Recipient{}, err
	}
	r.Prefs.Routes = ParseRoutes(routes)
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

func LoadStaff(ctx context.Context, db DB, roles []string, exclude string) ([]Recipient, error) {
	rows, err := db.Query(ctx, `
		SELECT id::text FROM core.users
		WHERE role = ANY($1::text[]) AND status = 'active'
		  AND ($2 = '' OR id::text <> $2)
		ORDER BY created_at
	`, roles, exclude)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]Recipient, 0, len(ids))
	for _, id := range ids {
		r, err := LoadRecipient(ctx, db, id)
		if err != nil {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func orEmpty(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}
