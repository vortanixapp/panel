package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vortanixapp/panel/pkg/notify"
)

const notificationsPageSize = 30

type notificationItem struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Group     string `json:"group"`
	Category  string `json:"category"`
	Icon      string `json:"icon"`
	Tone      string `json:"tone"`
	Action    string `json:"action"`
	Href      string `json:"href"`
	Unread    bool   `json:"unread"`
	ReadAt    string `json:"read_at"`
	CreatedAt string `json:"created_at"`
}

func toneOf(s notify.Severity) string {
	switch s {
	case notify.SeverityCritical:
		return "bad"
	case notify.SeverityWarning:
		return "warn"
	default:
		return "info"
	}
}

func (h *Handler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	q := r.URL.Query()

	limit := notificationsPageSize
	if v, err := strconv.Atoi(q.Get("limit")); err == nil && v > 0 && v <= 100 {
		limit = v
	}
	offset := 0
	if v, err := strconv.Atoi(q.Get("offset")); err == nil && v > 0 {
		offset = v
	}
	group := strings.TrimSpace(q.Get("group"))
	unreadOnly := q.Get("unread") == "1"

	var kinds []string
	if group != "" {
		for _, k := range notify.Kinds() {
			if string(notify.DefFor(k).Group) == group {
				kinds = append(kinds, string(k))
			}
		}
		if len(kinds) == 0 {
			kinds = []string{"\x00"}
		}
	}

	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT id::text, type, title, COALESCE(body, ''),
		       COALESCE(action_label, ''), COALESCE(action_href, ''), read_at, created_at
		FROM core.notifications
		WHERE user_id = $1
		  AND ($2::boolean IS NOT TRUE OR read_at IS NULL)
		  AND ($3::text[] IS NULL OR type = ANY($3::text[]))
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`, claims.UserID, unreadOnly, kinds, limit+1, offset)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"notifications": []any{}, "unread": 0, "groups": []any{}, "has_more": false,
			"channels": h.loadNotificationChannels(ctx, claims.UserID),
			"error":    "Не удалось загрузить оповещения",
		})
		return
	}
	defer rows.Close()

	list := []notificationItem{}
	for rows.Next() {
		var n notificationItem
		var readAt *time.Time
		var created time.Time
		if rows.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &n.Action, &n.Href, &readAt, &created) != nil {
			continue
		}
		def := notify.DefFor(notify.Kind(n.Type))
		n.Group = string(def.Group)
		n.Category = notify.GroupTitle(def.Group)
		n.Icon = def.Icon
		n.Tone = toneOf(def.Severity)
		n.CreatedAt = created.UTC().Format(time.RFC3339)
		n.Unread = readAt == nil
		if readAt != nil {
			n.ReadAt = readAt.UTC().Format(time.RFC3339)
		}
		list = append(list, n)
	}

	hasMore := len(list) > limit
	if hasMore {
		list = list[:limit]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"notifications": list,
		"unread":        h.notificationsUnread(ctx, claims.UserID),
		"groups":        h.notificationGroups(ctx, claims.UserID),
		"has_more":      hasMore,
		"channels":      h.loadNotificationChannels(ctx, claims.UserID),
	})
}

func (h *Handler) notificationGroups(ctx context.Context, userID string) []map[string]any {
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
			"id": string(g), "label": notify.GroupTitle(g),
			"count": total[g], "unread": unread[g],
		})
	}
	return out
}

func (h *Handler) notificationsUnread(ctx context.Context, userID string) int {
	var n int
	_ = h.readerOf(ctx).QueryRow(ctx, `
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
		"count": h.notificationsUnread(r.Context(), claims.UserID),
	})
}

func (h *Handler) NotificationsReadAll(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.notifications SET read_at = now()
		WHERE user_id = $1 AND read_at IS NULL
	`, claims.UserID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) NotificationRead(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.notifications SET read_at = now()
		WHERE id = $1::uuid AND user_id = $2 AND read_at IS NULL
	`, chi.URLParam(r, "id"), claims.UserID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) NotificationDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		DELETE FROM core.notifications
		WHERE id = $1::uuid AND user_id = $2
	`, chi.URLParam(r, "id"), claims.UserID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) NotificationsClearRead(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tag, _ := h.dbOf(r.Context()).Exec(r.Context(), `
		DELETE FROM core.notifications
		WHERE user_id = $1 AND read_at IS NOT NULL
	`, claims.UserID)
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok", "deleted": tag.RowsAffected(),
	})
}
