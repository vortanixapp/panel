package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vortanix/vortanix/pkg/notify"
)

// Лента оповещений в панели.
//
// Прежняя версия отдавала последние 200 записей и считала всё по ним: счётчики
// разделов были счётчиками выданной страницы, а не разделов, двести первое
// уведомление было недостижимо навсегда, фильтровать приходилось на стороне
// браузера. Вид записи собирался из справочников, ключи которых с реальными
// типами не совпадали ни в одном случае, поэтому почти всё показывалось как
// «Система» с общей иконкой.

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

// tone переводит важность в то, что понимает интерфейс.
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

	// Фильтр по разделу переводится в список типов: в базе хранится тип события,
	// а раздел — свойство справочника, и держать его копию в строке значило бы
	// завести ещё одно место, которому предстоит разойтись с истиной.
	var kinds []string
	if group != "" {
		for _, k := range notify.Kinds() {
			if string(notify.DefFor(k).Group) == group {
				kinds = append(kinds, string(k))
			}
		}
		if len(kinds) == 0 {
			kinds = []string{"\x00"} // заведомо пустая выборка
		}
	}

	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT id::text, type, title, COALESCE(body, ''),
		       COALESCE(action_label, ''), COALESCE(action_href, ''), read_at, created_at
		FROM core.notifications
		WHERE tenant_id = $1 AND user_id = $2
		  AND ($3::boolean IS NOT TRUE OR read_at IS NULL)
		  AND ($4::text[] IS NULL OR type = ANY($4::text[]))
		ORDER BY created_at DESC
		LIMIT $5 OFFSET $6
	`, claims.TenantID, claims.UserID, unreadOnly, kinds, limit+1, offset)
	if err != nil {
		// Ответ обязан сохранять форму даже при сбое: интерфейс читает channels
		// напрямую, и ответ без этого ключа ронял страницу целиком.
		writeJSON(w, http.StatusOK, map[string]any{
			"notifications": []any{}, "unread": 0, "groups": []any{}, "has_more": false,
			"channels": h.loadNotificationChannels(ctx, claims.TenantID, claims.UserID),
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
		"unread":        h.notificationsUnread(ctx, claims.TenantID, claims.UserID),
		"groups":        h.notificationGroups(ctx, claims.TenantID, claims.UserID),
		"has_more":      hasMore,
		"channels":      h.loadNotificationChannels(ctx, claims.TenantID, claims.UserID),
	})
}

// notificationGroups считает разделы по всей таблице, а не по выданной странице.
//
// Прежний счётчик в чипе показывал, сколько записей этого раздела попало в
// текущие двести строк, что при любой заметной ленте означало неверное число.
func (h *Handler) notificationGroups(ctx context.Context, tenantID, userID string) []map[string]any {
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT type, COUNT(*), COUNT(*) FILTER (WHERE read_at IS NULL)
		FROM core.notifications
		WHERE tenant_id = $1 AND user_id = $2
		GROUP BY type
	`, tenantID, userID)
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

func (h *Handler) notificationsUnread(ctx context.Context, tenantID, userID string) int {
	var n int
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT COUNT(*) FROM core.notifications
		WHERE tenant_id = $1 AND user_id = $2 AND read_at IS NULL
	`, tenantID, userID).Scan(&n)
	return n
}

// NotificationsUnreadCount — число для бейджа колокольчика.
//
// Считается тем же запросом, что и число на странице. Раньше это были разные
// условия: здесь фильтровали только по user_id, без tenant_id, и цифры
// расходились.
func (h *Handler) NotificationsUnreadCount(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{
		"count": h.notificationsUnread(r.Context(), claims.TenantID, claims.UserID),
	})
}

func (h *Handler) NotificationsReadAll(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	// Условие по tenant_id обязательно: без него «Прочитать все» гасило записи
	// пользователя во всех базах, где он встречается.
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.notifications SET read_at = now()
		WHERE tenant_id = $1 AND user_id = $2 AND read_at IS NULL
	`, claims.TenantID, claims.UserID)
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
		WHERE id = $1::uuid AND tenant_id = $2 AND user_id = $3 AND read_at IS NULL
	`, chi.URLParam(r, "id"), claims.TenantID, claims.UserID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// NotificationDelete убирает одно оповещение.
//
// Удаления не было вовсе: ни ручки, ни кнопки. Единственным способом убрать
// запись с глаз было пометить её прочитанной, а лента при этом росла вечно.
func (h *Handler) NotificationDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		DELETE FROM core.notifications
		WHERE id = $1::uuid AND tenant_id = $2 AND user_id = $3
	`, chi.URLParam(r, "id"), claims.TenantID, claims.UserID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// NotificationsClearRead убирает всё прочитанное разом.
func (h *Handler) NotificationsClearRead(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tag, _ := h.dbOf(r.Context()).Exec(r.Context(), `
		DELETE FROM core.notifications
		WHERE tenant_id = $1 AND user_id = $2 AND read_at IS NOT NULL
	`, claims.TenantID, claims.UserID)
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok", "deleted": tag.RowsAffected(),
	})
}
