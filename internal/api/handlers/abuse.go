package handlers

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

var (
	abuseSources          = []string{"rkn", "court", "police", "copyright", "abuse", "other"}
	abuseDefaultDeadlines = map[string]time.Duration{
		"rkn":       24 * time.Hour,
		"court":     24 * time.Hour,
		"police":    24 * time.Hour,
		"copyright": 24 * time.Hour,
		"abuse":     72 * time.Hour,
		"other":     72 * time.Hour,
	}
)

type abuseCase struct {
	ID          string     `json:"id"`
	Number      int64      `json:"number"`
	Source      string     `json:"source"`
	Reference   string     `json:"reference"`
	Subject     string     `json:"subject"`
	Description string     `json:"description"`
	Target      string     `json:"target"`
	Status      string     `json:"status"`
	Resolution  string     `json:"resolution"`
	ReceivedAt  time.Time  `json:"received_at"`
	DeadlineAt  *time.Time `json:"deadline_at"`
	ClosedAt    *time.Time `json:"closed_at"`
	ServerID    string     `json:"server_id"`
	ServerName  string     `json:"server_name"`
	UserID      string     `json:"user_id"`
	UserEmail   string     `json:"user_email"`
	CreatedAt   time.Time  `json:"created_at"`
	Overdue     bool       `json:"overdue"`
}

type abuseEvent struct {
	Action string    `json:"action"`
	Note   string    `json:"note"`
	Actor  string    `json:"actor"`
	At     time.Time `json:"at"`
}

const abuseCaseSelect = `
	SELECT c.id::text, c.number, c.source, c.reference, c.subject, c.description, c.target, c.status, c.resolution,
	       c.received_at, c.deadline_at, c.closed_at, COALESCE(c.server_id::text, ''),
	       COALESCE(s.name, c.server_name), COALESCE(c.user_id::text, ''), COALESCE(u.email, ''), c.created_at
	FROM core.abuse_cases c
	LEFT JOIN core.servers s ON s.id = c.server_id
	LEFT JOIN core.users u ON u.id = c.user_id`

func scanAbuseCase(row pgx.Row) (abuseCase, error) {
	var c abuseCase
	err := row.Scan(&c.ID, &c.Number, &c.Source, &c.Reference, &c.Subject, &c.Description, &c.Target, &c.Status,
		&c.Resolution, &c.ReceivedAt, &c.DeadlineAt, &c.ClosedAt, &c.ServerID, &c.ServerName, &c.UserID, &c.UserEmail, &c.CreatedAt)
	c.Overdue = c.Status == "new" && c.DeadlineAt != nil && time.Now().After(*c.DeadlineAt)
	return c, err
}

func abuseClosed(status string) bool {
	return status == "resolved" || status == "rejected"
}

func (h *Handler) AdminAbuseCases(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = "open"
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	rows, err := h.dbOf(ctx).Query(ctx, abuseCaseSelect+`
		WHERE ($1 = 'all' OR ($1 = 'open' AND c.status NOT IN ('resolved', 'rejected')) OR c.status = $1)
		  AND ($2 = '' OR c.subject ILIKE '%' || $2 || '%' OR c.reference ILIKE '%' || $2 || '%'
		       OR c.target ILIKE '%' || $2 || '%' OR COALESCE(u.email, '') ILIKE '%' || $2 || '%')
		ORDER BY (c.status IN ('resolved', 'rejected')), c.deadline_at NULLS LAST, c.created_at DESC
		LIMIT 300
	`, status, q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	cases := []abuseCase{}
	for rows.Next() {
		c, err := scanAbuseCase(rows)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		cases = append(cases, c)
	}
	var open, overdue int
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE status NOT IN ('resolved', 'rejected')),
		       COUNT(*) FILTER (WHERE status = 'new' AND deadline_at < now())
		FROM core.abuse_cases
	`).Scan(&open, &overdue)
	writeJSON(w, http.StatusOK, map[string]any{"cases": cases, "open": open, "overdue": overdue})
}

func (h *Handler) AdminAbuseCaseCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Source        string `json:"source"`
		Reference     string `json:"reference"`
		Subject       string `json:"subject"`
		Description   string `json:"description"`
		Target        string `json:"target"`
		ServerID      string `json:"server_id"`
		UserEmail     string `json:"user_email"`
		ReceivedAt    string `json:"received_at"`
		DeadlineHours int    `json:"deadline_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	subject := strings.TrimSpace(body.Subject)
	if !slices.Contains(abuseSources, body.Source) {
		writeError(w, http.StatusBadRequest, "Укажите, от кого поступило обращение")
		return
	}
	if subject == "" {
		writeError(w, http.StatusBadRequest, "Кратко опишите суть обращения")
		return
	}
	ctx := r.Context()
	db := h.dbOf(ctx)
	profile := h.accountingProfile(ctx)

	received := time.Now()
	if raw := strings.TrimSpace(body.ReceivedAt); raw != "" {
		t, err := time.ParseInLocation("2006-01-02T15:04", raw, profile.Location)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Дата поступления указана неверно")
			return
		}
		received = t
	}
	deadline := received.Add(abuseDefaultDeadlines[body.Source])
	if body.DeadlineHours > 0 {
		deadline = received.Add(time.Duration(body.DeadlineHours) * time.Hour)
	}

	serverID := strings.TrimSpace(body.ServerID)
	var userID, serverName string
	if serverID != "" {
		if err := db.QueryRow(ctx, `
			SELECT COALESCE(user_id::text, ''), name FROM core.servers WHERE id = $1
		`, serverID).Scan(&userID, &serverName); err != nil {
			writeError(w, http.StatusBadRequest, "Сервер не найден")
			return
		}
	} else if email := strings.TrimSpace(body.UserEmail); email != "" {
		if err := db.QueryRow(ctx, `
			SELECT id::text FROM core.users WHERE lower(email) = lower($1) AND deleted_at IS NULL
		`, email).Scan(&userID); err != nil {
			writeError(w, http.StatusBadRequest, "Клиент с таким email не найден")
			return
		}
	}

	var id string
	if err := db.QueryRow(ctx, `
		INSERT INTO core.abuse_cases (source, reference, subject, description, target, received_at, deadline_at,
		                              server_id, server_name, user_id, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, '')::uuid, $9, NULLIF($10, '')::uuid, $11)
		RETURNING id::text
	`, body.Source, strings.TrimSpace(body.Reference), subject, strings.TrimSpace(body.Description),
		strings.TrimSpace(body.Target), received, deadline, serverID, serverName, userID, claims.UserID).Scan(&id); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить обращение")
		return
	}
	h.addAbuseEvent(r, id, "created", subject, claims.UserID)
	audit(ctx, db, claims.UserID, "abuse.create", "abuse:"+id, map[string]any{"source": body.Source, "server_id": serverID})
	h.writeAbuseCase(w, r, id)
}

func (h *Handler) addAbuseEvent(r *http.Request, caseID, action, note, actorID string) {
	ctx := r.Context()
	_, _ = h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.abuse_case_events (case_id, action, note, actor_id) VALUES ($1, $2, $3, NULLIF($4, '')::uuid)
	`, caseID, action, note, actorID)
}

func (h *Handler) AdminAbuseCase(w http.ResponseWriter, r *http.Request) {
	h.writeAbuseCase(w, r, chi.URLParam(r, "id"))
}

func (h *Handler) writeAbuseCase(w http.ResponseWriter, r *http.Request, id string) {
	ctx := r.Context()
	db := h.dbOf(ctx)
	c, err := scanAbuseCase(db.QueryRow(ctx, abuseCaseSelect+` WHERE c.id = $1`, id))
	if err != nil {
		writeError(w, http.StatusNotFound, "Обращение не найдено")
		return
	}
	events := []abuseEvent{}
	rows, err := db.Query(ctx, `
		SELECT e.action, e.note, COALESCE(u.email, ''), e.created_at
		FROM core.abuse_case_events e
		LEFT JOIN core.users u ON u.id = e.actor_id
		WHERE e.case_id = $1
		ORDER BY e.created_at
	`, id)
	if err == nil {
		for rows.Next() {
			var e abuseEvent
			if rows.Scan(&e.Action, &e.Note, &e.Actor, &e.At) == nil {
				events = append(events, e)
			}
		}
		rows.Close()
	}
	writeJSON(w, http.StatusOK, map[string]any{"case": c, "events": events})
}

func (h *Handler) AdminAbuseCaseAction(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	var body struct {
		Action  string `json:"action"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ctx := r.Context()
	db := h.dbOf(ctx)
	c, err := scanAbuseCase(db.QueryRow(ctx, abuseCaseSelect+` WHERE c.id = $1`, id))
	if err != nil {
		writeError(w, http.StatusNotFound, "Обращение не найдено")
		return
	}
	message := strings.TrimSpace(body.Message)
	if abuseClosed(c.Status) && body.Action != "note" && body.Action != "lift" {
		writeError(w, http.StatusConflict, "Обращение закрыто")
		return
	}
	number := strconv.FormatInt(c.Number, 10)
	status := c.Status
	closing := false

	switch body.Action {
	case "notify":
		if c.UserID == "" {
			writeError(w, http.StatusConflict, "У обращения не указан клиент")
			return
		}
		if message == "" {
			writeError(w, http.StatusBadRequest, "Напишите текст уведомления для клиента")
			return
		}
		h.notifyUser(ctx, c.UserID, notify.Event{
			Kind:      notify.KindAbuseNotice,
			Title:     i18n.Key("notify.abuse.title", i18n.Params{"number": number}),
			Body:      i18n.Key("notify.abuse.body", i18n.Params{"message": message}),
			Meta:      map[string]any{"abuse_case_id": c.ID},
			DedupeKey: "abuse.notice:" + c.ID + ":" + strconv.FormatInt(time.Now().UnixNano(), 10),
		})
		if status == "new" {
			status = "notified"
		}
	case "restrict", "lift":
		if c.ServerID == "" {
			writeError(w, http.StatusConflict, "У обращения не указан сервер")
			return
		}
		blocked := body.Action == "restrict"
		reason := ""
		if blocked {
			reason = firstNonEmpty(message, "обращение № "+number)
		}
		ownerID, name, err := h.applyServerBlock(ctx, c.ServerID, blocked, reason)
		if err != nil {
			writeError(w, http.StatusConflict, "Не удалось изменить блокировку сервера")
			return
		}
		if ownerID != "" {
			event := notify.Event{
				Kind:      notify.KindServerBlocked,
				Title:     i18n.Key("notify.server_blocked.title"),
				Body:      i18n.Key("notify.server_blocked.body", i18n.Params{"name": name, "reason": i18n.Raw(reason)}),
				Meta:      map[string]any{"server_id": c.ServerID, "abuse_case_id": c.ID},
				DedupeKey: "server.blocked:" + c.ServerID + ":" + strconv.FormatInt(time.Now().Unix(), 10),
			}
			if !blocked {
				event = notify.Event{
					Kind:      notify.KindServerUnblocked,
					Title:     i18n.Key("notify.server_unblocked.title"),
					Body:      i18n.Key("notify.server_unblocked.body", i18n.Params{"name": name}),
					Meta:      map[string]any{"server_id": c.ServerID, "abuse_case_id": c.ID},
					DedupeKey: "server.unblocked:" + c.ServerID + ":" + strconv.FormatInt(time.Now().Unix(), 10),
				}
			}
			h.notifyUser(ctx, ownerID, event)
		}
		if blocked {
			status = "restricted"
		} else if status == "restricted" {
			status = "notified"
		}
	case "resolve", "reject":
		if message == "" {
			writeError(w, http.StatusBadRequest, "Опишите результат рассмотрения")
			return
		}
		status = "resolved"
		if body.Action == "reject" {
			status = "rejected"
		}
		closing = true
	case "note":
		if message == "" {
			writeError(w, http.StatusBadRequest, "Напишите заметку")
			return
		}
	default:
		writeError(w, http.StatusBadRequest, "Неизвестное действие")
		return
	}

	if _, err := db.Exec(ctx, `
		UPDATE core.abuse_cases
		SET status = $2, updated_at = now(),
		    resolution = CASE WHEN $3 THEN $4 ELSE resolution END,
		    closed_at = CASE WHEN $3 THEN now() ELSE closed_at END
		WHERE id = $1
	`, c.ID, status, closing, message); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	h.addAbuseEvent(r, c.ID, body.Action, message, claims.UserID)
	audit(ctx, db, claims.UserID, "abuse."+body.Action, "abuse:"+c.ID, map[string]any{"number": c.Number, "note": message})
	h.writeAbuseCase(w, r, c.ID)
}
