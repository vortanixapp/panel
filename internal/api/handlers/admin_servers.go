package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/internal/api/pricing"
	"github.com/vortanixapp/panel/pkg/gamecatalog"
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

func timeOrNilRFC(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

func (h *Handler) AdminServerCard(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		return
	}
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	var (
		name, nodeID, nodeName, nodeFqdn, nodeStatus       string
		containerID, containerName, provStatus, provError  string
		blockedReason, billingSource, tariffID, tariffName string
		ownerID, ownerEmail, ownerStatus                   string
		limitsRaw, configRaw, portsRaw                     []byte
		isBlocked, isTrial, autoRenew, autoStart           bool
		steamUpdatable                                     bool
		dunningStage, rentalPeriodDays                     int
		blockedAt, suspendedAt, expiresAt, dunningFor      *time.Time
		ownerCreatedAt                                     *time.Time
		createdAt                                          time.Time
	)

	err := h.readerOf(ctx).QueryRow(ctx, `
		SELECT s.name,
		       COALESCE(s.node_id::text, ''), COALESCE(n.name, ''), COALESCE(n.fqdn, ''), COALESCE(n.status, ''),
		       COALESCE(s.container_id, ''), COALESCE(s.container_name, ''),
		       COALESCE(s.provisioning_status, 'pending'), COALESCE(s.provisioning_error, ''),
		       COALESCE(s.limits, '{}'::jsonb), COALESCE(s.config, '{}'::jsonb), COALESCE(s.ports, '[]'::jsonb),
		       COALESCE(s.is_blocked, false), COALESCE(s.blocked_reason, ''), s.blocked_at, s.suspended_at,
		       s.expires_at, s.created_at,
		       COALESCE(s.auto_renew, false), COALESCE(s.auto_start, true),
		       COALESCE(s.is_trial, false), COALESCE(s.steam_updatable, false),
		       COALESCE(s.dunning_stage, 0), s.dunning_for,
		       COALESCE(s.rental_period_days, 30), COALESCE(s.billing_source, 'panel'),
		       COALESCE(s.tariff_id::text, ''), COALESCE(t.name, ''),
		       COALESCE(s.user_id::text, ''), COALESCE(u.email, ''), COALESCE(u.status, ''), u.created_at
		FROM core.servers s
		LEFT JOIN core.nodes n ON n.id = s.node_id
		LEFT JOIN core.tariffs t ON t.id = s.tariff_id
		LEFT JOIN core.users u ON u.id = s.user_id
		WHERE s.id = $1::uuid
	`, id).Scan(
		&name,
		&nodeID, &nodeName, &nodeFqdn, &nodeStatus,
		&containerID, &containerName, &provStatus, &provError,
		&limitsRaw, &configRaw, &portsRaw,
		&isBlocked, &blockedReason, &blockedAt, &suspendedAt,
		&expiresAt, &createdAt,
		&autoRenew, &autoStart,
		&isTrial, &steamUpdatable,
		&dunningStage, &dunningFor,
		&rentalPeriodDays, &billingSource,
		&tariffID, &tariffName,
		&ownerID, &ownerEmail, &ownerStatus, &ownerCreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "server not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	limits := map[string]any{}
	config := map[string]any{}
	ports := []any{}
	_ = json.Unmarshal(limitsRaw, &limits)
	_ = json.Unmarshal(configRaw, &config)
	_ = json.Unmarshal(portsRaw, &ports)

	var owner map[string]any
	if ownerID != "" {
		balances := []map[string]any{}
		rows, wErr := h.readerOf(ctx).Query(ctx, `
			SELECT currency, balance::float8 FROM core.wallets
			WHERE user_id = $1::uuid ORDER BY is_default DESC, currency
		`, ownerID)
		if wErr == nil {
			for rows.Next() {
				var currency string
				var balance float64
				if rows.Scan(&currency, &balance) == nil {
					balances = append(balances, map[string]any{"currency": currency, "balance": balance})
				}
			}
			rows.Close()
		}
		var serverCount int
		_ = h.readerOf(ctx).QueryRow(ctx, `
			SELECT COUNT(*) FROM core.servers WHERE user_id = $1::uuid
		`, ownerID).Scan(&serverCount)
		owner = map[string]any{
			"id":           ownerID,
			"email":        ownerEmail,
			"status":       ownerStatus,
			"created_at":   timeOrNilRFC(ownerCreatedAt),
			"balances":     balances,
			"server_count": serverCount,
		}
	}

	var friends, backups, abuseCases, tickets, extraPorts int
	_ = h.readerOf(ctx).QueryRow(ctx, `
		SELECT (SELECT COUNT(*) FROM core.server_friends WHERE server_id = $1::uuid),
		       (SELECT COUNT(*) FROM core.server_backups WHERE server_id = $1::uuid),
		       (SELECT COUNT(*) FROM core.abuse_cases WHERE server_id = $1::uuid),
		       (SELECT COUNT(*) FROM core.support_tickets
		         WHERE service_kind = 'server' AND service_id = $1::uuid),
		       (SELECT COUNT(*) FROM core.server_ports WHERE server_id = $1::uuid)
	`, id).Scan(&friends, &backups, &abuseCases, &tickets, &extraPorts)

	writeJSON(w, http.StatusOK, map[string]any{
		"id":                  id,
		"name":                name,
		"created_at":          createdAt.Format(time.RFC3339),
		"node":                map[string]any{"id": nodeID, "name": nodeName, "fqdn": nodeFqdn, "status": nodeStatus},
		"container_id":        nilIfEmpty(containerID),
		"container_name":      nilIfEmpty(containerName),
		"provisioning_status": provStatus,
		"provisioning_error":  nilIfEmpty(provError),
		"limits":              limits,
		"config":              config,
		"ports":               ports,
		"is_blocked":          isBlocked,
		"blocked_reason":      nilIfEmpty(blockedReason),
		"blocked_at":          timeOrNilRFC(blockedAt),
		"suspended_at":        timeOrNilRFC(suspendedAt),
		"expires_at":          timeOrNilRFC(expiresAt),
		"auto_renew":          autoRenew,
		"auto_start":          autoStart,
		"is_trial":            isTrial,
		"steam_updatable":     steamUpdatable,
		"dunning_stage":       dunningStage,
		"dunning_for":         timeOrNilRFC(dunningFor),
		"rental_period_days":  rentalPeriodDays,
		"billing_source":      billingSource,
		"tariff":              map[string]any{"id": nilIfEmpty(tariffID), "name": nilIfEmpty(tariffName)},
		"owner":               owner,
		"counts": map[string]any{
			"friends":     friends,
			"backups":     backups,
			"abuse_cases": abuseCases,
			"tickets":     tickets,
			"ports":       extraPorts,
		},
	})
}

func (h *Handler) AdminServerAudit(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		return
	}
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	limit := 100
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	offset := 0
	if v := strings.TrimSpace(r.URL.Query().Get("offset")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			offset = n
		}
	}

	q := `
		SELECT a.id::text, a.action, COALESCE(a.meta, '{}'::jsonb), a.created_at,
		       COALESCE(a.user_id::text, ''), COALESCE(u.email, '')
		FROM core.audit_logs a
		LEFT JOIN core.users u ON u.id = a.user_id
		WHERE (a.resource = $1 OR a.resource = $2)
	`
	args := []any{"server:" + id, id}
	if action := strings.TrimSpace(r.URL.Query().Get("action")); action != "" {
		args = append(args, action)
		q += fmt.Sprintf(" AND a.action = $%d", len(args))
	}
	if search := strings.TrimSpace(r.URL.Query().Get("search")); search != "" {
		args = append(args, search)
		n := len(args)
		q += fmt.Sprintf(
			" AND (a.action ILIKE '%%' || $%d || '%%' OR a.meta::text ILIKE '%%' || $%d || '%%'"+
				" OR u.email ILIKE '%%' || $%d || '%%')", n, n, n)
	}
	q += fmt.Sprintf(" ORDER BY a.created_at DESC LIMIT %d OFFSET %d", limit, offset)

	rows, err := h.readerOf(ctx).Query(ctx, q, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var entryID, action, actorID, actorEmail string
		var metaRaw []byte
		var createdAt time.Time
		if rows.Scan(&entryID, &action, &metaRaw, &createdAt, &actorID, &actorEmail) != nil {
			continue
		}
		meta := map[string]any{}
		_ = json.Unmarshal(metaRaw, &meta)
		list = append(list, map[string]any{
			"id":          entryID,
			"action":      action,
			"meta":        meta,
			"created_at":  createdAt.Format(time.RFC3339),
			"actor_id":    nilIfEmpty(actorID),
			"actor_email": nilIfEmpty(actorEmail),
		})
	}

	actions := []string{}
	actionRows, actErr := h.readerOf(ctx).Query(ctx, `
		SELECT DISTINCT action FROM core.audit_logs
		WHERE resource = $1 OR resource = $2
		ORDER BY action
	`, "server:"+id, id)
	if actErr == nil {
		for actionRows.Next() {
			var a string
			if actionRows.Scan(&a) == nil {
				actions = append(actions, a)
			}
		}
		actionRows.Close()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"entries": list, "actions": actions, "limit": limit, "offset": offset,
	})
}

func (h *Handler) adminServerOwner(ctx context.Context, id string) (name, ownerID string, err error) {
	err = h.dbOf(ctx).QueryRow(ctx, `
		SELECT name, COALESCE(user_id::text, '') FROM core.servers WHERE id = $1::uuid
	`, id).Scan(&name, &ownerID)
	return name, ownerID, err
}

func (h *Handler) AdminServerSetOwner(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	var body struct {
		UserID string `json:"user_id"`
		Email  string `json:"email"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	body.UserID = strings.TrimSpace(body.UserID)
	body.Email = strings.TrimSpace(body.Email)
	if body.UserID == "" && body.Email == "" {
		writeError(w, http.StatusBadRequest, "Укажите пользователя")
		return
	}

	var newOwnerID, newOwnerEmail, newOwnerStatus string
	lookup := `SELECT id::text, email, status FROM core.users WHERE deleted_at IS NULL AND id = $1::uuid`
	arg := body.UserID
	if body.UserID == "" {
		lookup = `SELECT id::text, email, status FROM core.users WHERE deleted_at IS NULL AND lower(email) = lower($1)`
		arg = body.Email
	}
	if err := h.dbOf(ctx).QueryRow(ctx, lookup, arg).Scan(&newOwnerID, &newOwnerEmail, &newOwnerStatus); err != nil {
		writeError(w, http.StatusNotFound, "Пользователь не найден")
		return
	}
	if newOwnerStatus != "active" {
		writeError(w, http.StatusConflict, "Аккаунт получателя отключён")
		return
	}

	name, oldOwnerID, err := h.adminServerOwner(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}
	if oldOwnerID == newOwnerID {
		writeError(w, http.StatusConflict, "Сервер уже закреплён за этим пользователем")
		return
	}

	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE core.servers SET user_id = $2::uuid WHERE id = $1::uuid
	`, id, newOwnerID); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сменить владельца")
		return
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM core.server_friends WHERE server_id = $1::uuid AND user_id = $2::uuid
	`, id, newOwnerID); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сменить владельца")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сменить владельца")
		return
	}
	_ = h.cache.InvalidateTenantServers(ctx)

	h.notifyUser(ctx, newOwnerID, notify.Event{
		Kind:      notify.KindServerOwner,
		Title:     i18n.Key("notify.server_owner_changed.title"),
		Body:      i18n.Key("notify.server_owner_changed.body", i18n.Params{"name": name}),
		Action:    h.serverAction("notify.action.open_server", id, ""),
		Meta:      map[string]any{"server_id": id},
		DedupeKey: "server.owner_changed:" + id + ":" + newOwnerID,
	})
	if oldOwnerID != "" {
		h.notifyUser(ctx, oldOwnerID, notify.Event{
			Kind:      notify.KindServerOwner,
			Title:     i18n.Key("notify.server_owner_removed.title"),
			Body:      i18n.Key("notify.server_owner_removed.body", i18n.Params{"name": name}),
			Meta:      map[string]any{"server_id": id},
			DedupeKey: "server.owner_removed:" + id + ":" + oldOwnerID,
		})
	}

	audit(ctx, h.dbOf(ctx), claims.UserID, "server.owner_change", "server:"+id,
		map[string]any{"from": oldOwnerID, "to": newOwnerID, "email": newOwnerEmail})
	h.auditAlert(ctx, claims.UserID, claims.Email, "server.owner_change", "server:"+id)

	writeJSON(w, http.StatusOK, map[string]any{"user_id": newOwnerID, "email": newOwnerEmail})
}

func (h *Handler) AdminServerSetExpiry(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	ctx := r.Context()

	var body struct {
		Days      int    `json:"days"`
		ExpiresAt string `json:"expires_at"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	body.ExpiresAt = strings.TrimSpace(body.ExpiresAt)
	if body.Days == 0 && body.ExpiresAt == "" {
		writeError(w, http.StatusBadRequest, "Укажите количество дней или дату")
		return
	}
	if body.Days < -3650 || body.Days > 3650 {
		writeError(w, http.StatusBadRequest, "Недопустимое количество дней")
		return
	}

	name, ownerID, err := h.adminServerOwner(ctx, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	var newExpiry *time.Time
	if body.ExpiresAt != "" {
		parsed, parseErr := time.Parse(time.RFC3339, body.ExpiresAt)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "Неверный формат даты")
			return
		}
		err = h.dbOf(ctx).QueryRow(ctx, `
			UPDATE core.servers
			SET expires_at = $2,
			    suspended_at = CASE WHEN $2 > now() THEN NULL ELSE suspended_at END,
			    dunning_stage = CASE WHEN $2 > now() THEN 0 ELSE dunning_stage END,
			    dunning_for = CASE WHEN $2 > now() THEN NULL ELSE dunning_for END
			WHERE id = $1::uuid
			RETURNING expires_at
		`, id, parsed).Scan(&newExpiry)
	} else {
		err = h.dbOf(ctx).QueryRow(ctx, `
			UPDATE core.servers
			SET expires_at = GREATEST(COALESCE(expires_at, now()), now()) + make_interval(days => $2),
			    suspended_at = CASE WHEN $2 > 0 THEN NULL ELSE suspended_at END,
			    dunning_stage = CASE WHEN $2 > 0 THEN 0 ELSE dunning_stage END,
			    dunning_for = CASE WHEN $2 > 0 THEN NULL ELSE dunning_for END
			WHERE id = $1::uuid
			RETURNING expires_at
		`, id, body.Days).Scan(&newExpiry)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось изменить срок аренды")
		return
	}
	_ = h.cache.InvalidateTenantServers(ctx)

	if ownerID != "" && newExpiry != nil {
		date := newExpiry.Format("02.01.2006")
		event := notify.Event{
			Kind:      notify.KindServerExtended,
			Title:     i18n.Key("notify.server_expiry_changed.title"),
			Body:      i18n.Key("notify.server_expiry_changed.body", i18n.Params{"name": name, "date": date}),
			Action:    h.serverAction("notify.action.open_server", id, ""),
			Meta:      map[string]any{"server_id": id},
			DedupeKey: "server.expiry:" + id + ":" + date,
		}
		if body.ExpiresAt == "" && body.Days > 0 {
			event.Title = i18n.Key("notify.server_days_granted.title")
			event.Body = i18n.Key("notify.server_days_granted.body",
				i18n.Params{"name": name, "days": body.Days, "date": date})
		}
		h.notifyUser(ctx, ownerID, event)
	}

	audit(ctx, h.dbOf(ctx), claims.UserID, "server.expiry_set", "server:"+id,
		map[string]any{"days": body.Days, "expires_at": body.ExpiresAt})

	writeJSON(w, http.StatusOK, map[string]any{"expires_at": timeOrNilRFC(newExpiry)})
}

func (h *Handler) AdminServerNotes(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		return
	}
	id := chi.URLParam(r, "id")
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT n.id::text, n.body, n.created_at, COALESCE(u.email, '')
		FROM core.server_notes n
		LEFT JOIN core.users u ON u.id = n.author_id
		WHERE n.server_id = $1::uuid
		ORDER BY n.created_at DESC
		LIMIT 200
	`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	list := []map[string]any{}
	for rows.Next() {
		var noteID, body, authorEmail string
		var createdAt time.Time
		if rows.Scan(&noteID, &body, &createdAt, &authorEmail) != nil {
			continue
		}
		list = append(list, map[string]any{
			"id":         noteID,
			"body":       body,
			"created_at": createdAt.Format(time.RFC3339),
			"author":     nilIfEmpty(authorEmail),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"notes": list})
}

func (h *Handler) AdminServerNoteCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	var body struct {
		Body string `json:"body"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	body.Body = strings.TrimSpace(body.Body)
	if body.Body == "" {
		writeError(w, http.StatusBadRequest, "Заметка не может быть пустой")
		return
	}
	if len([]rune(body.Body)) > 4000 {
		writeError(w, http.StatusBadRequest, "Заметка слишком длинная")
		return
	}

	ctx := r.Context()
	var noteID string
	var createdAt time.Time
	err := h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.server_notes (server_id, author_id, body)
		VALUES ($1::uuid, $2::uuid, $3)
		RETURNING id::text, created_at
	`, id, claims.UserID, body.Body).Scan(&noteID, &createdAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить заметку")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":         noteID,
		"body":       body.Body,
		"created_at": createdAt.Format(time.RFC3339),
		"author":     claims.Email,
	})
}

func (h *Handler) AdminServerNoteDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	noteID := chi.URLParam(r, "noteId")
	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		DELETE FROM core.server_notes WHERE id = $1::uuid AND server_id = $2::uuid
	`, noteID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Заметка не найдена")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "server.note_delete", "server:"+id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) AdminServersBulk(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body struct {
		IDs    []string `json:"ids"`
		Action string   `json:"action"`
		Reason string   `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	switch body.Action {
	case "start", "stop", "restart", "kill", "block", "unblock":
	default:
		writeError(w, http.StatusBadRequest, "Неизвестное действие")
		return
	}
	if len(body.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "Не выбрано ни одного сервера")
		return
	}
	if len(body.IDs) > 200 {
		writeError(w, http.StatusBadRequest, "За один раз можно обработать не больше 200 серверов")
		return
	}

	ctx := r.Context()
	done := 0
	failures := []map[string]string{}
	for _, id := range body.IDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		var err error
		switch body.Action {
		case "block", "unblock":
			_, _, err = h.applyServerBlock(ctx, id, body.Action == "block", body.Reason)
		default:
			_, _, err = h.sendServerPower(ctx, claims.UserID, id, body.Action)
		}
		if err != nil {
			failures = append(failures, map[string]string{"id": id, "error": err.Error()})
			continue
		}
		done++
	}

	audit(ctx, h.dbOf(ctx), claims.UserID, "server.bulk", "server:*",
		map[string]any{"action": body.Action, "count": done, "failed": len(failures)})

	writeJSON(w, http.StatusOK, map[string]any{"done": done, "failures": failures})
}

func (h *Handler) adminResolveUser(ctx context.Context, userID, email string) (string, string, error) {
	userID = strings.TrimSpace(userID)
	email = strings.TrimSpace(email)
	if userID == "" && email == "" {
		return "", "", errors.New("укажите пользователя")
	}
	lookup := `SELECT id::text, email FROM core.users WHERE deleted_at IS NULL AND status = 'active' AND id = $1::uuid`
	arg := userID
	if userID == "" {
		lookup = `SELECT id::text, email FROM core.users WHERE deleted_at IS NULL AND status = 'active' AND lower(email) = lower($1)`
		arg = email
	}
	var id, mail string
	if err := h.dbOf(ctx).QueryRow(ctx, lookup, arg).Scan(&id, &mail); err != nil {
		return "", "", errors.New("пользователь не найден")
	}
	return id, mail, nil
}

func (h *Handler) AdminServerCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()

	var body struct {
		UserID        string `json:"user_id"`
		Email         string `json:"email"`
		NodeID        string `json:"node_id"`
		LocationID    string `json:"location_id"`
		GameID        string `json:"game_id"`
		GameVersionID string `json:"game_version_id"`
		TariffID      string `json:"tariff_id"`
		Name          string `json:"name"`
		Period        int    `json:"period"`
		Slots         int    `json:"slots"`
		CPUCores      int    `json:"cpu_cores"`
		RAMGb         int    `json:"ram_gb"`
		DiskGb        int    `json:"disk_gb"`
		Antiddos      bool   `json:"antiddos_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	nodeID := strings.TrimSpace(body.NodeID)
	if nodeID == "" {
		nodeID = strings.TrimSpace(body.LocationID)
	}
	if nodeID == "" || body.Name == "" {
		writeError(w, http.StatusBadRequest, "Укажите название и локацию")
		return
	}

	ownerID, ownerEmail, err := h.adminResolveUser(ctx, body.UserID, body.Email)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	gameID, gameOK := resolveGameSlug(ctx, h, body.GameID)
	if !gameOK {
		writeError(w, http.StatusBadRequest, "Игра не найдена в каталоге")
		return
	}
	if h.nodeInMaintenance(ctx, nodeID) {
		writeError(w, http.StatusConflict, "На этой локации идут технические работы")
		return
	}
	if reason := h.nodeCapacityReason(ctx, nodeID, gameID); reason != "" {
		writeError(w, http.StatusConflict, reason)
		return
	}

	periodDays := 30
	if body.Period > 0 {
		periodDays = body.Period
	}

	limits := gamecatalog.DefaultLimits(gameID)
	if body.TariffID != "" {
		tariffJSON, loadErr := h.loadTariffLegacyJSON(ctx, body.TariffID)
		if loadErr != nil {
			writeError(w, http.StatusBadRequest, "Тариф не найден")
			return
		}
		order := pricing.Resolve(tariffJSON, pricing.RentOrder{
			Slots: body.Slots, CPUCores: body.CPUCores, RAMGb: body.RAMGb,
			DiskGb: body.DiskGb, AntiddosEnabled: body.Antiddos,
		})
		if msg := tariffMeetsGame(gameID, order); msg != "" {
			writeError(w, http.StatusBadRequest, msg)
			return
		}
		limits = tariffLimits(tariffJSON, order, gamecatalog.DefaultLimits(gameID))
	}

	limitsJSON, _ := json.Marshal(limits)
	expires := time.Now().Add(time.Duration(periodDays) * 24 * time.Hour)

	var id string
	err = h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.servers ( node_id, game_id, game_version_id, name, limits, user_id, tariff_id,
		                           provisioning_status, expires_at, rental_period_days, config)
		VALUES ( $1::uuid, $2, NULLIF($3, '')::uuid, $4, $5::jsonb, $6::uuid, NULLIF($7, '')::uuid,
		         'provisioning', $8, $9, jsonb_build_object('startup_params', $10::text))
		RETURNING id::text
	`, nodeID, gameID, body.GameVersionID, body.Name, limitsJSON, ownerID, body.TariffID, expires, periodDays,
		h.defaultStartupParams(ctx, gameID)).Scan(&id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось создать сервер")
		return
	}

	h.emitWebhook(ctx, "server.created", map[string]any{
		"server_id": id, "server_name": body.Name, "game_id": gameID,
		"node_id": nodeID, "user_id": ownerID,
	})
	h.launchNewServer(ctx, id, nodeID, gameID, body.Name, limitsJSON)

	audit(ctx, h.dbOf(ctx), claims.UserID, "server.create", "server:"+id,
		map[string]any{"name": body.Name, "owner": ownerEmail, "by_admin": true})

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":          id,
		"server_id":   id,
		"status":      "provisioning",
		"period_days": periodDays,
		"expires_at":  expires.Format(time.RFC3339),
		"owner_email": ownerEmail,
	})
}
