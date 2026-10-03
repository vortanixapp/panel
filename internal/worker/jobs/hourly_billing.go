package jobs

import (
	"context"
	"log"
	"time"

	"github.com/vortanixapp/panel/pkg/hourlybill"
	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
	"github.com/vortanixapp/panel/pkg/settingsreg"
)

const (
	hourlyBillingTick  = 5 * time.Minute
	hourlyBillingBatch = 200
)

type hourlyServer struct {
	id          string
	nodeID      string
	userID      string
	name        string
	gameID      string
	limits      []byte
	status      string
	rate        float64
	carry       float64
	currency    string
	billedUntil time.Time
}

func (r *Runner) HourlyBillingLoop(ctx context.Context) {
	ticker := time.NewTicker(hourlyBillingTick)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.heartbeat.Beat(LoopHourlyBilling)
			if r.panelFrozen(ctx) {
				continue
			}
			r.chargeHourlyServers(ctx)
		}
	}
}

func (r *Runner) chargeHourlyServers(ctx context.Context) {
	rows, err := r.db.Query(ctx, `
		SELECT s.id::text, s.node_id::text, COALESCE(s.user_id::text, ''),
		       s.name, s.game_id, s.limits, s.status,
		       s.hourly_rate::float8, s.hourly_carry::float8, COALESCE(t.currency, 'RUB'), s.billed_until
		FROM core.servers s
		LEFT JOIN core.tariffs t ON t.id = s.tariff_id
		WHERE s.payment_mode = 'hourly'
		  AND s.billed_until IS NOT NULL
		  AND s.billed_until <= now()
		  AND s.suspended_at IS NULL
		ORDER BY s.billed_until ASC
		LIMIT $1
	`, hourlyBillingBatch)
	if err != nil {
		return
	}
	var list []hourlyServer
	for rows.Next() {
		var s hourlyServer
		if rows.Scan(&s.id, &s.nodeID, &s.userID, &s.name, &s.gameID, &s.limits,
			&s.status, &s.rate, &s.carry, &s.currency, &s.billedUntil) == nil {
			list = append(list, s)
		}
	}
	rows.Close()

	for _, s := range list {
		r.chargeHourlyServer(ctx, s)
	}
}

func (r *Runner) chargeHourlyServer(ctx context.Context, s hourlyServer) {
	hours := int(time.Since(s.billedUntil)/time.Hour) + 1
	if hours < 1 {
		hours = 1
	}
	if limit := int(settingsreg.BillingHourlyCatchup.Int()); hours > limit {
		hours = limit
	}

	for i := 0; i < hours; i++ {
		periodStart := s.billedUntil.Add(time.Duration(i) * time.Hour)
		paid, err := r.chargeHour(ctx, &s, periodStart)
		if err != nil {
			log.Printf("hourly: server %s: %v", s.id, err)
			return
		}
		if !paid {
			r.suspendServer(ctx, expiredServer{
				id: s.id, nodeID: s.nodeID, userID: s.userID, name: s.name,
				gameID: s.gameID, limits: s.limits, status: s.status,
				expiresAt: periodStart,
			})
			if s.userID != "" {
				r.notifyUser(ctx, s.userID, notify.Event{
					Kind:      notify.KindServerSuspended,
					Title:     i18n.Key("notify.hourly_stopped.title"),
					Body:      i18n.Key("notify.hourly_stopped.body", i18n.Params{"name": s.name}),
					Action:    r.serverAction("notify.action.topup", s.id, ""),
					Meta:      map[string]any{"server_id": s.id},
					DedupeKey: "server.hourly_stopped:" + s.id + ":" + periodStart.Format(time.RFC3339),
				})
			}
			return
		}
	}

	if s.userID != "" && s.rate > 0 {
		r.warnLowHourlyBalance(ctx, s)
	}
}

func (r *Runner) chargeHour(ctx context.Context, s *hourlyServer, periodStart time.Time) (bool, error) {
	charge, carry := hourlybill.Split(s.rate, s.carry)
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	res, err := tx.Exec(ctx, `
		INSERT INTO core.server_hourly_charges (server_id, period_start, amount, currency)
		VALUES ($1::uuid, $2, $3, $4)
		ON CONFLICT DO NOTHING
	`, s.id, periodStart, charge, s.currency)
	if err != nil {
		return false, err
	}
	if res.RowsAffected() == 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE core.servers SET billed_until = $2 WHERE id = $1::uuid
		`, s.id, periodStart.Add(time.Hour)); err != nil {
			return false, err
		}
		return true, tx.Commit(ctx)
	}

	if charge > 0 {
		var walletID string
		var balance float64
		err = tx.QueryRow(ctx, `
			SELECT id::text, balance FROM core.wallets
			WHERE user_id = $1::uuid AND currency = $2
			ORDER BY is_default DESC
			LIMIT 1
			FOR UPDATE
		`, s.userID, s.currency).Scan(&walletID, &balance)
		if err != nil || balance < charge {
			return false, nil
		}
		if _, err := tx.Exec(ctx, `
			UPDATE core.wallets SET balance = balance - $2, updated_at = now() WHERE id = $1::uuid
		`, walletID, charge); err != nil {
			return false, err
		}
		var txID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO core.transactions ( wallet_id, type, amount, description, source_type, source_id)
			VALUES ($1::uuid, 'debit', $2, $3, 'server_hourly', $4::uuid)
			RETURNING id::text
		`, walletID, -charge, "Почасовая аренда: "+s.name, s.id).Scan(&txID); err != nil {
			return false, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE core.server_hourly_charges SET tx_id = $3::uuid
			WHERE server_id = $1::uuid AND period_start = $2
		`, s.id, periodStart, txID); err != nil {
			return false, err
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE core.servers SET billed_until = $2, hourly_carry = $3 WHERE id = $1::uuid
	`, s.id, periodStart.Add(time.Hour), carry); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	s.carry = carry
	return true, nil
}

func (r *Runner) warnLowHourlyBalance(ctx context.Context, s hourlyServer) {
	var balance float64
	if err := r.db.QueryRow(ctx, `
		SELECT balance FROM core.wallets
		WHERE user_id = $1::uuid AND currency = $2
		ORDER BY is_default DESC LIMIT 1
	`, s.userID, s.currency).Scan(&balance); err != nil {
		return
	}
	hoursLeft := int(balance / s.rate)
	if hoursLeft > int(settingsreg.BillingHourlyLowBalance.Int()) {
		return
	}
	r.notifyUser(ctx, s.userID, notify.Event{
		Kind:  notify.KindServerExpiring,
		Title: i18n.Key("notify.hourly_low.title"),
		Body: i18n.Key("notify.hourly_low.body", i18n.Params{
			"name": s.name, "hours": hoursLeft,
		}),
		Action:    r.serverAction("notify.action.topup", s.id, ""),
		Meta:      map[string]any{"server_id": s.id, "hours_left": hoursLeft},
		DedupeKey: "server.hourly_low:" + s.id + ":" + time.Now().Format("2006-01-02T15"),
	})
}
