package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vortanixapp/panel/pkg/notify"
)

const (
	dunningInterval  = 30 * time.Minute
	dunningSuspended = -1
)

var dunningStages = []int{1, 3, 7}

func (h *Handler) StartServerDunningSweeper(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(dunningInterval)
		defer ticker.Stop()
		h.runDunning(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.runDunning(ctx)
			}
		}
	}()
}

func (h *Handler) runDunning(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, dunningInterval)
	defer cancel()

	h.renewAutoServers(ctx)
	for _, stage := range dunningStages {
		h.remindStage(ctx, stage)
	}
	h.notifySuspended(ctx)
}

type dunningServer struct {
	id        string
	tenantID  string
	userID    string
	name      string
	expiresAt time.Time
	period    int
}

func (h *Handler) remindStage(ctx context.Context, stage int) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		UPDATE core.servers s
		SET dunning_stage = $1, dunning_for = s.expires_at
		WHERE s.id IN (
			SELECT id FROM core.servers
			WHERE expires_at IS NOT NULL
			  AND suspended_at IS NULL
			  AND user_id IS NOT NULL
			  AND expires_at > now()
			  AND expires_at <= now() + make_interval(days => $1::int)
			  AND (dunning_for IS DISTINCT FROM expires_at OR dunning_stage <= 0 OR dunning_stage > $1)
			ORDER BY expires_at ASC
			LIMIT 200
			FOR UPDATE SKIP LOCKED
		)
		RETURNING s.id::text, s.tenant_id::text, s.user_id::text, s.name, s.expires_at,
		          COALESCE(s.rental_period_days, 30)
	`, stage)
	if err != nil {
		return
	}
	list := []dunningServer{}
	for rows.Next() {
		var d dunningServer
		if rows.Scan(&d.id, &d.tenantID, &d.userID, &d.name, &d.expiresAt, &d.period) == nil {
			list = append(list, d)
		}
	}
	rows.Close()

	for _, d := range list {
		price, currency, _, _, _, err := h.serverRenewBaseCostCtx(ctx, d.tenantID, d.id, d.period)
		body := fmt.Sprintf("Аренда сервера «%s» заканчивается %s.",
			d.name, d.expiresAt.Format("02.01.2006 15:04"))
		if err == nil && price > 0 {
			body += fmt.Sprintf(" Продление на %d дн. стоит %.2f %s.", d.period, price, currency)
		}
		body += " После окончания сервер будет остановлен, файлы сохранятся."
		h.notifyServerOwner(ctx, d.tenantID, d.id, notify.Event{
			Kind:   notify.KindServerExpiring,
			Title:  fmt.Sprintf("Осталось %d дн. аренды", stage),
			Body:   body,
			Action: h.serverAction("Продлить", d.id, "/tariff"),
			Meta:   map[string]any{"server_id": d.id, "days": stage},
			// Ключ включает ступень: напоминание за 7 дней и за 1 день — разные
			// события, а повтор той же ступени клиенту не нужен.
			DedupeKey: fmt.Sprintf("server.expiring:%s:%d", d.id, stage),
		})
	}
}

func (h *Handler) notifySuspended(ctx context.Context) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		UPDATE core.servers s
		SET dunning_stage = $1, dunning_for = s.expires_at
		WHERE s.id IN (
			SELECT id FROM core.servers
			WHERE suspended_at IS NOT NULL
			  AND user_id IS NOT NULL
			  AND (dunning_for IS DISTINCT FROM expires_at OR dunning_stage <> $1)
			LIMIT 200
			FOR UPDATE SKIP LOCKED
		)
		RETURNING s.id::text, s.tenant_id::text, s.user_id::text, s.name, s.expires_at,
		          COALESCE(s.rental_period_days, 30)
	`, dunningSuspended)
	if err != nil {
		return
	}
	list := []dunningServer{}
	for rows.Next() {
		var d dunningServer
		if rows.Scan(&d.id, &d.tenantID, &d.userID, &d.name, &d.expiresAt, &d.period) == nil {
			list = append(list, d)
		}
	}
	rows.Close()

	for _, d := range list {
		body := fmt.Sprintf(
			"Сервер «%s» остановлен: срок аренды закончился %s. Файлы сохранены — продлите аренду, и сервер запустится снова.",
			d.name, d.expiresAt.Format("02.01.2006"))
		h.notifyServerOwner(ctx, d.tenantID, d.id, notify.Event{
			Kind:   notify.KindServerSuspended,
			Title:  "Сервер остановлен",
			Body:   body,
			Action: h.serverAction("Продлить", d.id, "/tariff"),
			Meta:   map[string]any{"server_id": d.id},
			// Об остановке сообщает и воркер по истечении срока, и этот
			// подметатель. Ключ не даёт клиенту получить два разных уведомления
			// об одном и том же.
			DedupeKey: "server.suspended:" + d.id + ":" + d.expiresAt.Format(time.RFC3339),
		})
		h.emitWebhook(ctx, d.tenantID, "server.suspended", map[string]any{
			"server_id": d.id, "server_name": d.name, "user_id": d.userID,
			"expired_at": d.expiresAt.Format(time.RFC3339),
		})
	}
}

func (h *Handler) renewAutoServers(ctx context.Context) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT id::text, tenant_id::text, COALESCE(user_id::text, ''), name, expires_at,
		       COALESCE(rental_period_days, 30)
		FROM core.servers
		WHERE auto_renew = true
		  AND user_id IS NOT NULL
		  AND expires_at IS NOT NULL
		  AND suspended_at IS NULL
		  AND expires_at <= now() + interval '1 day'
		ORDER BY expires_at ASC
		LIMIT 100
	`)
	if err != nil {
		return
	}
	list := []dunningServer{}
	for rows.Next() {
		var d dunningServer
		if rows.Scan(&d.id, &d.tenantID, &d.userID, &d.name, &d.expiresAt, &d.period) == nil && d.userID != "" {
			list = append(list, d)
		}
	}
	rows.Close()

	for _, d := range list {
		h.renewOneAuto(ctx, d)
	}
}

func (h *Handler) renewOneAuto(ctx context.Context, d dunningServer) {
	price, currency, _, _, _, err := h.serverRenewBaseCostCtx(ctx, d.tenantID, d.id, d.period)
	if err != nil || price <= 0 {
		return
	}

	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback(ctx)

	var walletID string
	var balance float64
	if err := tx.QueryRow(ctx, `
		SELECT id::text, balance FROM core.wallets
		WHERE tenant_id = $1 AND user_id = $2::uuid AND UPPER(currency) = UPPER($3)
		FOR UPDATE
	`, d.tenantID, d.userID, currency).Scan(&walletID, &balance); err != nil {
		h.notifyAutoRenewFailed(ctx, d, "нет кошелька в валюте тарифа")
		return
	}
	if balance < price {
		h.notifyAutoRenewFailed(ctx, d,
			fmt.Sprintf("на балансе %.2f %s, нужно %.2f", balance, currency, price))
		return
	}

	if _, err := tx.Exec(ctx, `
		UPDATE core.wallets SET balance = balance - $2, updated_at = now() WHERE id = $1
	`, walletID, price); err != nil {
		return
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO core.transactions (tenant_id, wallet_id, type, amount, description, source_type, source_id)
		VALUES ($1, $2, 'debit', $3, $4, 'server_renew', $5::uuid)
	`, d.tenantID, walletID, -price, "Автопродление: "+d.name, d.id); err != nil {
		return
	}
	if _, err := tx.Exec(ctx, `
		UPDATE core.servers
		SET expires_at = GREATEST(expires_at, now()) + make_interval(days => $2::int),
		    dunning_stage = 0, dunning_for = NULL
		WHERE id = $1
	`, d.id, d.period); err != nil {
		return
	}
	if err := tx.Commit(ctx); err != nil {
		return
	}

	body := fmt.Sprintf("Аренда сервера «%s» продлена на %d дн., списано %.2f %s.",
		d.name, d.period, price, currency)
	h.notifyServerOwner(ctx, d.tenantID, d.id, notify.Event{
		Kind:   notify.KindServerRenewed,
		Title:  "Аренда продлена",
		Body:   body,
		Action: h.serverAction("Открыть сервер", d.id, ""),
		Meta:   map[string]any{"server_id": d.id, "amount": price},
	})
	h.emitWebhook(ctx, d.tenantID, "server.renewed", map[string]any{
		"server_id": d.id, "server_name": d.name, "user_id": d.userID,
		"amount": price, "currency": currency, "days": d.period,
	})
	log.Printf("автопродление: сервер %s продлён на %d дн. за %.2f %s", d.id, d.period, price, currency)
}

func (h *Handler) notifyAutoRenewFailed(ctx context.Context, d dunningServer, reason string) {
	body := fmt.Sprintf(
		"Не удалось продлить аренду сервера «%s» автоматически: %s. Пополните баланс, иначе сервер будет остановлен %s.",
		d.name, reason, d.expiresAt.Format("02.01.2006 15:04"))
	h.notifyServerOwner(ctx, d.tenantID, d.id, notify.Event{
		Kind:   notify.KindPaymentFailed,
		Title:  "Автопродление не прошло",
		Body:   body,
		Action: h.panelAction("Пополнить баланс", "/billing"),
		Meta:   map[string]any{"server_id": d.id},
		DedupeKey: fmt.Sprintf("server.renew_failed:%s:%s", d.id,
			d.expiresAt.Format("2006-01-02")),
	})
}

func (h *Handler) ServerAutoRenew(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	serverID := chi.URLParam(r, "id")
	if !h.requireServerTenant(w, r, claims.TenantID, serverID) {
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if _, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.servers SET auto_renew = $3 WHERE id = $1 AND tenant_id = $2
	`, serverID, claims.TenantID, body.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "server.auto_renew", "server:"+serverID,
		map[string]any{"enabled": body.Enabled})
	writeJSON(w, http.StatusOK, map[string]bool{"auto_renew": body.Enabled})
}
