package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vortanix/vortanix/internal/api/jobwake"
)

func (h *Handler) ListSupportTickets(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	// Последнее сообщение и их число берём подзапросами: список показывает
	// превью переписки, и без него карточка отвечает только на вопрос «о чём
	// тикет», но не «на чём он стоит».
	q := `SELECT t.id::text, t.subject, t.status, t.created_at::text, t.category, t.priority,
	             COALESCE(t.last_message_at, t.created_at)::text,
	             (SELECT COUNT(*) FROM core.support_messages m WHERE m.ticket_id = t.id),
	             COALESCE((SELECT LEFT(COALESCE(m.body, m.message), 200) FROM core.support_messages m
	                       WHERE m.ticket_id = t.id ORDER BY m.created_at DESC LIMIT 1), ''),
	             COALESCE((SELECT m.is_staff FROM core.support_messages m
	                       WHERE m.ticket_id = t.id ORDER BY m.created_at DESC LIMIT 1), false),
	             t.service_kind, COALESCE(t.service_id::text, ''), t.notify
	      FROM core.support_tickets t WHERE t.tenant_id = $1`
	args := []any{claims.TenantID}
	if !isStaffRole(claims.Role) {
		q += ` AND t.user_id = $2`
		args = append(args, claims.UserID)
	}
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), q+` ORDER BY COALESCE(t.last_message_at, t.created_at) DESC`, args...)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var id, sub, status, created, category, priority, lastAt, lastText string
			var serviceKind, serviceID string
			var messages int
			var lastFromStaff, notify bool
			if rows.Scan(&id, &sub, &status, &created, &category, &priority, &lastAt,
				&messages, &lastText, &lastFromStaff, &serviceKind, &serviceID, &notify) == nil {
				list = append(list, map[string]any{
					"id": id, "subject": sub, "status": status, "created_at": created,
					"category": category, "priority": priority, "last_message_at": lastAt,
					"messages": messages, "last_message": lastText,
					"last_from_staff": lastFromStaff, "notify": notify,
					"service_kind": serviceKind, "service_id": serviceID,
				})
			}
		}
	}
	// Названия услуг разрешаем одним запросом на весь список: по запросу на
	// строку это дало бы столько же обращений к базе, сколько тикетов.
	serviceIDs := []string{}
	for _, item := range list {
		if sid, _ := item["service_id"].(string); sid != "" {
			serviceIDs = append(serviceIDs, sid)
		}
	}
	if labels := h.serviceLabels(r.Context(), claims.TenantID, serviceIDs); len(labels) > 0 {
		for _, item := range list {
			sid, _ := item["service_id"].(string)
			if label := labels[sid]; label != "" {
				item["service"] = label
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"tickets": list})
}

func (h *Handler) CreateSupportTicket(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Subject     string `json:"subject"`
		Body        string `json:"body"`
		Category    string `json:"category"`
		Priority    string `json:"priority"`
		ServiceKind string `json:"service_kind"`
		ServiceID   string `json:"service_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	// Форма спрашивает отдел и срочность — до этой правки оба поля молча
	// терялись, и очередь разбиралась глазами по теме письма.
	category := strings.TrimSpace(body.Category)
	if category == "" {
		category = "other"
	}
	// Приоритет ограничен схемой, и чужое значение уронило бы вставку целиком.
	priority := strings.TrimSpace(body.Priority)
	switch priority {
	case "low", "normal", "high", "urgent":
	default:
		priority = "normal"
	}

	// Услугу принимаем только вместе с её видом: идентификатор без вида не
	// сказать в какой таблице искать, а вид без идентификатора ничего не
	// значит.
	serviceKind := normalizeServiceKind(body.ServiceKind)
	serviceID := strings.TrimSpace(body.ServiceID)
	if serviceKind == "" || serviceID == "" {
		serviceKind, serviceID = "", ""
	}

	var id string
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.support_tickets
		    (tenant_id, user_id, subject, category, priority, service_kind, service_id)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::uuid) RETURNING id::text
	`, claims.TenantID, claims.UserID, body.Subject, category, priority,
		serviceKind, serviceID).Scan(&id)
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.support_messages (tenant_id, ticket_id, user_id, message, body)
		SELECT tenant_id, $1::uuid, $2::uuid, $3, $3 FROM core.support_tickets WHERE id = $1::uuid
	`, id, claims.UserID, body.Body)
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) SupportUnreadCount(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var n int
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `SELECT COUNT(*) FROM core.support_tickets WHERE user_id = $1 AND status = 'open'`, claims.UserID).Scan(&n)
	writeJSON(w, http.StatusOK, map[string]int{"count": n})
}

func (h *Handler) GetSupportTicket(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	ctx := r.Context()
	var ticketUserID string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT user_id::text FROM core.support_tickets WHERE id = $1 AND tenant_id = $2
	`, id, claims.TenantID).Scan(&ticketUserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if !isStaffRole(claims.Role) && ticketUserID != claims.UserID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	rows, _ := h.dbOf(ctx).Query(ctx, `
		SELECT id::text, COALESCE(body, message), is_staff, created_at::text, meta
		FROM core.support_messages WHERE ticket_id = $1 ORDER BY created_at
	`, id)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	msgs := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var mid, body, created string
			var staff bool
			var metaRaw []byte
			if rows.Scan(&mid, &body, &staff, &created, &metaRaw) == nil {
				msg := map[string]any{"id": mid, "body": body, "is_staff": staff, "created_at": created}
				// Вложения из таблицы, старое одиночное — из meta сообщения:
				// файлы, залитые до появления таблицы, никуда не делись.
				list := h.supportAttachmentsFor(ctx, claims.TenantID, id, mid)
				if att := supportAttachmentInfo(metaRaw, id, mid); att != nil {
					list = append(list, att)
				}
				if len(list) > 0 {
					msg["attachments"] = list
					msg["attachment"] = list[0]
				}
				msgs = append(msgs, msg)
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ticket_id": id, "messages": msgs})
}

func (h *Handler) ReplySupportTicket(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Body string `json:"body"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	ctx := r.Context()
	ticketID := chi.URLParam(r, "id")
	staff := isStaffRole(claims.Role)

	// Обращение искалось по одному id: клиент писал в чужую переписку, а если
	// он сотрудник — его сообщение ещё и уходило владельцу как ответ поддержки.
	var ticketUserID string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT user_id::text FROM core.support_tickets WHERE id = $1 AND tenant_id = $2
	`, ticketID, claims.TenantID).Scan(&ticketUserID); err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if !staff && ticketUserID != claims.UserID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if strings.TrimSpace(body.Body) == "" {
		writeError(w, http.StatusBadRequest, "нужен текст сообщения")
		return
	}

	// Ошибка вставки глушилась, а ответ всё равно был успешным: человек видел
	// «отправлено» и ждал ответа на сообщение, которого нет.
	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.support_messages (tenant_id, ticket_id, user_id, message, body, is_staff)
		VALUES ($1, $2::uuid, $3::uuid, $4, $4, $5)
	`, claims.TenantID, ticketID, claims.UserID, body.Body, staff); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить сообщение")
		return
	}
	if staff {
		h.notifySupportReply(ctx, claims.TenantID, ticketID, body.Body)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Закрытие искало тикет по одному id, без арендатора и автора: зная id, любой
// вошедший пользователь закрывал чужое обращение.
func (h *Handler) CloseSupportTicket(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	ticketID := chi.URLParam(r, "id")
	var ticketUserID string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT user_id::text FROM core.support_tickets WHERE id = $1 AND tenant_id = $2
	`, ticketID, claims.TenantID).Scan(&ticketUserID); err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if !isStaffRole(claims.Role) && ticketUserID != claims.UserID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if _, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.support_tickets SET status = 'closed', closed_at = now()
		WHERE id = $1 AND tenant_id = $2
	`, ticketID, claims.TenantID); err != nil {
		writeError(w, http.StatusInternalServerError, "close failed")
		return
	}
	h.notifySupportStatus(ctx, claims.TenantID, ticketID, "closed")
	writeJSON(w, http.StatusOK, map[string]string{"status": "closed"})
}

func (h *Handler) ListNews(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	// Скрытые новости клиенту не показываем: колонка active была, но в отборе
	// не участвовала — выключенная новость всё равно висела в списке.
	// Непрочитанное считаем сравнением с отметкой пользователя, а не отдельной
	// таблицей связей: лента показывает «новое с прошлого захода», и для этого
	// одной даты достаточно.
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT n.id::text, n.slug, n.title, n.excerpt, n.published_at::text,
		       n.tag, n.pinned,
		       n.published_at > COALESCE(u.news_read_at, 'epoch'::timestamptz) AS unread
		FROM core.news n
		CROSS JOIN LATERAL (
		    SELECT news_read_at FROM core.users WHERE id = $2::uuid
		) u
		WHERE n.tenant_id = $1 AND n.active = true AND n.published_at IS NOT NULL
		ORDER BY n.pinned DESC, n.published_at DESC
		LIMIT 50
	`, claims.TenantID, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось получить новости")
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	ids := []string{}
	for rows.Next() {
		var id, slug, title, tag string
		var excerpt *string
		var pub string
		var pinned, unread bool
		if rows.Scan(&id, &slug, &title, &excerpt, &pub, &tag, &pinned, &unread) != nil {
			continue
		}
		ids = append(ids, id)
		list = append(list, map[string]any{
			"id": id, "slug": slug, "title": title, "excerpt": excerpt,
			"published_at": pub, "tag": tag, "pinned": pinned, "unread": unread,
		})
	}
	images := h.newsImages(r, claims.TenantID, ids)
	for _, item := range list {
		id, _ := item["id"].(string)
		if imgs := images[id]; len(imgs) > 0 {
			item["image"] = imgs[0]["url"]
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"news": list})
}

func (h *Handler) GetNews(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	slug := chi.URLParam(r, "slug")
	var id, title, body, tag string
	var excerpt *string
	var pub *string
	var pinned bool
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT id::text, title, excerpt, body, published_at::text, tag, pinned
		FROM core.news WHERE tenant_id = $1 AND slug = $2 AND active = true
	`, claims.TenantID, slug).Scan(&id, &title, &excerpt, &body, &pub, &tag, &pinned)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	item := map[string]any{
		"id": id, "slug": slug, "title": title, "excerpt": excerpt, "body": body,
		"tag": tag, "pinned": pinned,
	}
	imgs := h.newsImages(r, claims.TenantID, []string{id})[id]
	if imgs == nil {
		imgs = []map[string]string{}
	}
	item["images"] = imgs
	if len(imgs) > 0 {
		item["image"] = imgs[0]["url"]
	}
	if pub != nil {
		item["published_at"] = *pub
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) ListNewsAdmin(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, slug, title, excerpt, body, published_at::text, active, tag, pinned
		FROM core.news
		WHERE tenant_id = $1 ORDER BY pinned DESC, COALESCE(published_at, created_at) DESC LIMIT 50
	`, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось получить новости: "+err.Error())
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	ids := []string{}
	for rows.Next() {
		var id, slug, title, tag string
		var excerpt, bodyStr, pub *string
		var active, pinned bool
		if rows.Scan(&id, &slug, &title, &excerpt, &bodyStr, &pub, &active, &tag, &pinned) != nil {
			continue
		}
		item := map[string]any{
			"id": id, "slug": slug, "title": title, "excerpt": excerpt, "body": bodyStr,
			"active": active, "tag": tag, "pinned": pinned,
		}
		if pub != nil {
			item["published_at"] = *pub
		}
		ids = append(ids, id)
		list = append(list, item)
	}

	images := h.newsImages(r, claims.TenantID, ids)
	for _, item := range list {
		id, _ := item["id"].(string)
		imgs := images[id]
		if imgs == nil {
			imgs = []map[string]string{}
		}
		item["images"] = imgs
		if len(imgs) > 0 {
			item["image"] = imgs[0]["url"]
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"news": list})
}

func (h *Handler) CreateNews(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body newsBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	title := strings.TrimSpace(derefOrEmpty(body.Title))
	excerpt := derefOrEmpty(body.Excerpt)
	slug := newsSlugOrFallback(derefOrEmpty(body.Slug), title, time.Now())

	if err := validateNews(title, excerpt, slug); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	publishedAt, hasPublished, err := parseNewsDate(body.PublishedAt)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// Занятый slug — это ошибка, а не повод переписать чужую запись. Раньше
	// здесь делался UPDATE, и создание новости с уже существующим адресом
	// молча затирало старую.
	var taken string
	_ = h.dbOf(r.Context()).QueryRow(r.Context(),
		`SELECT id::text FROM core.news WHERE tenant_id = $1 AND slug = $2`, claims.TenantID, slug).Scan(&taken)
	if taken != "" {
		writeError(w, http.StatusConflict, "новость с таким адресом уже есть")
		return
	}

	var id string
	err = h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.news (tenant_id, slug, title, excerpt, body, published_at, active, tag, pinned)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5,
		        CASE WHEN $6::bool THEN $7::timestamptz ELSE now() END,
		        COALESCE($8, true), COALESCE(NULLIF($9, ''), 'update'), COALESCE($10, false))
		RETURNING id::text
	`, claims.TenantID, slug, title, excerpt, body.Body, hasPublished, publishedAt, body.Active,
		strings.TrimSpace(derefOrEmpty(body.Tag)), body.Pinned).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось создать новость: "+err.Error())
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "news.create", "news:"+id, nil)
	writeJSON(w, http.StatusCreated, map[string]string{"id": id, "slug": slug})
}

func (h *Handler) MonitoringPublicLive(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	ctx := r.Context()

	tenantID, ok := h.resolveMonitoringPublicTenant(w, ctx, r, serverID)
	if !ok {
		return
	}
	if !h.publicMonitoringAllowed(w, ctx, tenantID, serverID) {
		return
	}
	if _, _, _, _, err := h.loadMonitoringServer(ctx, tenantID, serverID); err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	metrics, _ := h.cache.GetMetrics(ctx, serverID, 1)
	var latest map[string]any
	if len(metrics) > 0 {
		latest = metrics[len(metrics)-1]
	}
	online := 0
	maxPlayers := 0
	if latest != nil {
		if v, ok := latest["players_online"]; ok {
			online = intFromAny(v)
		} else if v, ok := latest["online"]; ok {
			online = intFromAny(v)
		}
		if v, ok := latest["max_players"]; ok {
			maxPlayers = intFromAny(v)
		} else if v, ok := latest["max"]; ok {
			maxPlayers = intFromAny(v)
		}
	}
	if live, ok := h.cache.GetServerStatus(ctx, serverID); ok && live == "running" && online == 0 {
		online = 1
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"item": map[string]any{"online": online, "max": maxPlayers, "metrics": latest},
	})
}

func (h *Handler) MonitoringPublicStats(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	ctx := r.Context()

	tenantID, ok := h.resolveMonitoringPublicTenant(w, ctx, r, serverID)
	if !ok {
		return
	}
	if !h.publicMonitoringAllowed(w, ctx, tenantID, serverID) {
		return
	}
	if _, _, _, _, err := h.loadMonitoringServer(ctx, tenantID, serverID); err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	days := 1
	if v := r.URL.Query().Get("days"); v == "7" {
		days = 7
	}
	hours := days * 24
	points, err := getMetricsFromPG(ctx, h.dbOf(ctx), serverID, hours, 500)
	if err != nil || len(points) == 0 {
		points, _ = h.cache.GetMetrics(ctx, serverID, 120)
	}

	labels := []string{}
	onlineArr := []any{}
	cpuArr := []any{}
	ramArr := []any{}
	for _, p := range points {
		if ts, ok := metricTS(p); ok {
			labels = append(labels, time.Unix(ts, 0).UTC().Format("15:04"))
		} else {
			labels = append(labels, "")
		}
		onlineArr = append(onlineArr, p["players_online"])
		cpuArr = append(cpuArr, p["cpu_pct"])
		if memUsed, ok := p["mem_used_mb"]; ok {
			if memLimit, ok2 := p["mem_limit_mb"]; ok2 {
				limit := floatFromAny(memLimit)
				if limit > 0 {
					ramArr = append(ramArr, floatFromAny(memUsed)/limit*100)
					continue
				}
			}
		}
		ramArr = append(ramArr, nil)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"days": days,
		"series": map[string]any{
			"labels":  labels,
			"online":  onlineArr,
			"max":     []any{},
			"cpu":     cpuArr,
			"ram":     ramArr,
			"disk":    []any{},
			"max_cap": nil,
		},
	})
}

// publicMonitoringAllowed проверяет, что владелец сервера включил публичную
// страницу. Проверка стояла только в одной ручке из четырёх: остальные отдавали
// онлайн, историю нагрузки и баннер по одному id, что бы владелец ни настроил.
func (h *Handler) publicMonitoringAllowed(w http.ResponseWriter, ctx context.Context, tenantID, serverID string) bool {
	if !h.loadMonitoringSettings(ctx, tenantID, serverID).PublicEnabled {
		writeError(w, http.StatusNotFound, "public page disabled")
		return false
	}
	return true
}

func (h *Handler) resolveMonitoringPublicTenant(w http.ResponseWriter, ctx context.Context, r *http.Request, serverID string) (string, bool) {
	tenantID, resolved := h.resolvePublicTenantID(ctx, r)
	if !resolved {
		if fallback, ok := lookupServerTenant(ctx, h.dbOf(ctx), serverID); ok {
			return fallback, true
		}
		writeError(w, http.StatusBadRequest, "tenant not resolved")
		return "", false
	}
	if serverTenant, ok := lookupServerTenant(ctx, h.dbOf(ctx), serverID); ok && serverTenant != tenantID {
		writeError(w, http.StatusNotFound, "server not found")
		return "", false
	}
	return tenantID, true
}

func (h *Handler) loadMonitoringServer(ctx context.Context, tenantID, serverID string) (name, status, gameID, gameName string, err error) {
	err = h.dbOf(ctx).QueryRow(ctx, `
		SELECT s.name, s.status, s.game_id, COALESCE(g.name, '')
		FROM core.servers s
		LEFT JOIN core.games g ON g.tenant_id = s.tenant_id AND g.slug = s.game_id
		WHERE s.id = $1 AND s.tenant_id = $2
	`, serverID, tenantID).Scan(&name, &status, &gameID, &gameName)
	return
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}

func floatFromAny(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

func (h *Handler) ListMailings(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	rows, _ := h.dbOf(r.Context()).Query(r.Context(), `SELECT id::text, subject, status FROM core.mailings WHERE tenant_id = $1`, claims.TenantID)
	defer func() {
		if rows != nil {
			rows.Close()
		}
	}()
	list := []map[string]any{}
	if rows != nil {
		for rows.Next() {
			var id, sub, status string
			if rows.Scan(&id, &sub, &status) == nil {
				list = append(list, map[string]any{"id": id, "subject": sub, "status": status})
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"mailings": list})
}

func (h *Handler) CreateMailing(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	subject, _ := body["subject"].(string)
	title, _ := body["title"].(string)
	bodyText, _ := body["body"].(string)
	if title == "" {
		title = subject
	}

	if id, ok := body["id"].(string); ok && id != "" {
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
			UPDATE core.mailings SET title=$3, subject=$4, body=$5 WHERE id=$1::uuid AND tenant_id=$2
		`, id, claims.TenantID, title, subject, bodyText)
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
		return
	}
	var id string
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.mailings (tenant_id, title, subject, body) VALUES ($1, COALESCE($2, $3), $3, $4) RETURNING id::text
	`, claims.TenantID, title, subject, bodyText).Scan(&id)
	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *Handler) SendMailing(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	mailingID := chi.URLParam(r, "id")
	_, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.mailings SET status = 'queued', scheduled_at = now() WHERE id = $1 AND tenant_id = $2
	`, mailingID, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	payload, _ := json.Marshal(map[string]string{"mailing_id": mailingID})
	_, _ = h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.jobs (tenant_id, type, status, payload) VALUES ($1, 'send_mailing', 'pending', $2::jsonb)
	`, claims.TenantID, payload)
	jobwake.Notify("send_mailing")
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}

func (h *Handler) MonitoringPublicBanner(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	size := chi.URLParam(r, "size")
	ctx := r.Context()
	tenantID, ok := h.resolveMonitoringPublicTenant(w, ctx, r, serverID)
	if !ok {
		return
	}
	if !h.publicMonitoringAllowed(w, ctx, tenantID, serverID) {
		return
	}
	name, status, _, _, err := h.loadMonitoringServer(ctx, tenantID, serverID)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=60")
	_, _ = w.Write([]byte(monitoringBannerSVG(name, status, size)))
}

func monitoringBannerSVG(name, status, size string) string {
	width, height := 560, 95
	if size == "small" {
		width, height = 320, 64
	}
	color := "#22c55e"
	if status != "running" {
		color = "#ef4444"
	}
	return `<svg xmlns="http://www.w3.org/2000/svg" width="` + bannerItoa(width) + `" height="` + bannerItoa(height) + `">
  <rect width="100%" height="100%" rx="8" fill="#0f172a"/>
  <circle cx="24" cy="` + bannerItoa(height/2) + `" r="8" fill="` + color + `"/>
  <text x="44" y="` + bannerItoa(height/2+6) + `" fill="#f8fafc" font-family="sans-serif" font-size="16">` + escapeXML(name) + `</text>
</svg>`
}

func bannerItoa(n int) string {
	if n <= 0 {
		return "0"
	}
	buf := make([]byte, 0, 12)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}
