package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

const (
	notificationsPageSize    = 30
	notificationsStreamCycle = 25 * time.Second
	notificationsStreamPing  = 10 * time.Second
)

const notificationColumns = `
	n.id::text, n.type, n.title, COALESCE(n.body, ''),
	COALESCE(n.action_label, ''), COALESCE(n.action_href, ''), n.read_at, n.created_at`

type notificationItem struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Group     string `json:"group"`
	Category  string `json:"category"`
	Icon      string `json:"icon"`
	Severity  string `json:"severity"`
	Tone      string `json:"tone"`
	Quiet     bool   `json:"quiet"`
	Action    string `json:"action"`
	Href      string `json:"href"`
	Unread    bool   `json:"unread"`
	ReadAt    string `json:"read_at"`
	CreatedAt string `json:"created_at"`

	createdAt time.Time
}

func toneOf(s notify.Severity) string {
	switch s {
	case notify.SeverityCritical:
		return "bad"
	case notify.SeverityWarning:
		return "warn"
	case notify.SeveritySuccess:
		return "ok"
	default:
		return "info"
	}
}

func scanNotification(row pgx.Row, l i18n.Localizer) (notificationItem, error) {
	var n notificationItem
	var readAt *time.Time
	if err := row.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &n.Action, &n.Href, &readAt, &n.createdAt); err != nil {
		return n, err
	}
	def := notify.DefFor(notify.Kind(n.Type))
	n.Group = string(def.Group)
	n.Category = l.Text(notify.GroupLabel(def.Group))
	n.Icon = def.Icon
	n.Severity = string(def.Severity)
	n.Tone = toneOf(def.Severity)
	n.Quiet = def.Quiet
	n.CreatedAt = n.createdAt.UTC().Format(time.RFC3339)
	n.Unread = readAt == nil
	if readAt != nil {
		n.ReadAt = readAt.UTC().Format(time.RFC3339)
	}
	return n, nil
}

func notificationKinds(group string) []string {
	if group == "" {
		return nil
	}
	kinds := []string{}
	for _, k := range notify.KindsOf(notify.Group(group)) {
		kinds = append(kinds, string(k))
	}
	if len(kinds) == 0 {
		return []string{"\x00"}
	}
	return kinds
}

func notificationCursor(n notificationItem) string {
	return n.createdAt.UTC().Format(time.RFC3339Nano) + "_" + n.ID
}

func parseNotificationCursor(raw string) (any, any) {
	at, id, ok := strings.Cut(strings.TrimSpace(raw), "_")
	if !ok {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, at)
	if err != nil {
		return nil, nil
	}
	if _, err := uuid.Parse(id); err != nil {
		return nil, nil
	}
	return t, id
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	q := r.URL.Query()
	l := i18n.ForUser(ctx, h.dbOf(ctx), claims.UserID)

	limit := notificationsPageSize
	if v, err := strconv.Atoi(q.Get("limit")); err == nil && v > 0 && v <= 100 {
		limit = v
	}
	kinds := notificationKinds(strings.TrimSpace(q.Get("group")))
	unreadOnly := q.Get("unread") == "1"
	search := []rune(strings.TrimSpace(q.Get("q")))
	if len(search) > 100 {
		search = search[:100]
	}
	beforeAt, beforeID := parseNotificationCursor(q.Get("before"))

	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT `+notificationColumns+`
		FROM core.notifications n
		WHERE n.user_id = $1
		  AND ($2::boolean IS NOT TRUE OR n.read_at IS NULL)
		  AND ($3::text[] IS NULL OR n.type = ANY($3::text[]))
		  AND ($4 = '' OR n.title ILIKE '%' || $4 || '%' OR n.body ILIKE '%' || $4 || '%')
		  AND ($5::timestamptz IS NULL OR (n.created_at, n.id) < ($5::timestamptz, $6::uuid))
		ORDER BY n.created_at DESC, n.id DESC
		LIMIT $7
	`, claims.UserID, unreadOnly, kinds, likeEscaper.Replace(string(search)), beforeAt, beforeID, limit+1)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	list := []notificationItem{}
	for rows.Next() {
		n, err := scanNotification(rows, l)
		if err != nil {
			continue
		}
		list = append(list, n)
	}
	rows.Close()
	if rows.Err() != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	next := ""
	if len(list) > limit {
		list = list[:limit]
		next = notificationCursor(list[len(list)-1])
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"notifications": list,
		"next_cursor":   next,
		"unread":        h.notificationsUnread(ctx, h.readerOf(ctx), claims.UserID),
		"groups":        h.notificationGroups(ctx, claims.UserID, l),
	})
}

func (h *Handler) notificationGroups(ctx context.Context, userID string, l i18n.Localizer) []map[string]any {
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT type, COUNT(*), COUNT(*) FILTER (WHERE read_at IS NULL)
		FROM core.notifications
		WHERE user_id = $1
		GROUP BY type
	`, userID)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	total := map[notify.Group]int{}
	unread := map[notify.Group]int{}
	for rows.Next() {
		var kind string
		var n, u int
		if rows.Scan(&kind, &n, &u) != nil {
			continue
		}
		g := notify.DefFor(notify.Kind(kind)).Group
		total[g] += n
		unread[g] += u
	}

	out := []map[string]any{}
	for _, g := range notify.Groups() {
		if total[g] == 0 {
			continue
		}
		out = append(out, map[string]any{
			"id": string(g), "label": l.Text(notify.GroupLabel(g)),
			"count": total[g], "unread": unread[g],
		})
	}
	return out
}

func (h *Handler) notificationsUnread(ctx context.Context, db notify.DB, userID string) int {
	var n int
	_ = db.QueryRow(ctx, `
		SELECT COUNT(*) FROM core.notifications
		WHERE user_id = $1 AND read_at IS NULL
	`, userID).Scan(&n)
	return n
}

func (h *Handler) NotificationsUnreadCount(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{
		"count": h.notificationsUnread(r.Context(), h.readerOf(r.Context()), claims.UserID),
	})
}

func (h *Handler) notificationChanged(w http.ResponseWriter, r *http.Request, userID string, affected int64, extra map[string]any) {
	ctx := r.Context()
	db := h.dbOf(ctx)
	if affected > 0 {
		notify.PublishSync(ctx, db, userID)
	}
	out := map[string]any{"status": "ok", "affected": affected, "unread": h.notificationsUnread(ctx, db, userID)}
	for k, v := range extra {
		out[k] = v
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) NotificationsReadAll(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	tag, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.notifications SET read_at = now()
		WHERE user_id = $1 AND read_at IS NULL
		  AND ($2::text[] IS NULL OR type = ANY($2::text[]))
	`, claims.UserID, notificationKinds(strings.TrimSpace(r.URL.Query().Get("group"))))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	h.notificationChanged(w, r, claims.UserID, tag.RowsAffected(), nil)
}

func (h *Handler) NotificationRead(w http.ResponseWriter, r *http.Request) {
	h.setNotificationRead(w, r, true)
}

func (h *Handler) NotificationUnread(w http.ResponseWriter, r *http.Request) {
	h.setNotificationRead(w, r, false)
}

func (h *Handler) setNotificationRead(w http.ResponseWriter, r *http.Request, read bool) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	query := `UPDATE core.notifications SET read_at = now()
		WHERE id = $1::uuid AND user_id = $2 AND read_at IS NULL`
	if !read {
		query = `UPDATE core.notifications SET read_at = NULL
		WHERE id = $1::uuid AND user_id = $2 AND read_at IS NOT NULL`
	}
	ctx := r.Context()
	tag, err := h.dbOf(ctx).Exec(ctx, query, id, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	h.notificationChanged(w, r, claims.UserID, tag.RowsAffected(), nil)
}

func (h *Handler) NotificationDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	ctx := r.Context()
	tag, err := h.dbOf(ctx).Exec(ctx, `
		DELETE FROM core.notifications
		WHERE id = $1::uuid AND user_id = $2
	`, id, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	h.notificationChanged(w, r, claims.UserID, tag.RowsAffected(), map[string]any{"status": "deleted"})
}

func (h *Handler) NotificationsClearRead(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	tag, err := h.dbOf(ctx).Exec(ctx, `
		DELETE FROM core.notifications
		WHERE user_id = $1 AND read_at IS NOT NULL
	`, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	h.notificationChanged(w, r, claims.UserID, tag.RowsAffected(), map[string]any{"deleted": tag.RowsAffected()})
}

func (h *Handler) NotificationsStream(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok || h.live == nil {
		writeError(w, http.StatusServiceUnavailable, "streaming unavailable")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

	ctx := r.Context()
	events, cancel := h.live.subscribe(claims.UserID)
	defer cancel()

	send := func(event string, payload any) bool {
		b, err := json.Marshal(payload)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	db := h.dbOf(ctx)
	if !send("hello", map[string]int{"unread": h.notificationsUnread(ctx, db, claims.UserID)}) {
		return
	}

	cycle := time.NewTimer(notificationsStreamCycle)
	defer cycle.Stop()
	ping := time.NewTicker(notificationsStreamPing)
	defer ping.Stop()

	var l *i18n.Localizer
	for {
		select {
		case <-ctx.Done():
			return
		case <-cycle.C:
			return
		case <-ping.C:
			if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case msg := <-events:
			unread := h.notificationsUnread(ctx, db, claims.UserID)
			if msg != liveSync {
				if l == nil {
					loc := i18n.ForUser(ctx, db, claims.UserID)
					l = &loc
				}
				item, err := scanNotification(db.QueryRow(ctx, `
					SELECT `+notificationColumns+`
					FROM core.notifications n
					WHERE n.id = $1::uuid AND n.user_id = $2
				`, msg, claims.UserID), *l)
				if err == nil {
					if !send("notification", map[string]any{"item": item, "unread": unread}) {
						return
					}
					continue
				}
			}
			if !send("sync", map[string]int{"unread": unread}) {
				return
			}
		}
	}
}
