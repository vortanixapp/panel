package jobs

import (
	"context"
	"log"
	"math"
	"time"

	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

const (
	hourlyBillingTick  = 5 * time.Minute
	hourlyBillingBatch = 200
	hourlyCatchupMax   = 24
	hourlyLowBalance   = 6
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
		       s.hourly_rate::float8, COALESCE(t.currency, 'RUB'), s.billed_until
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
			&s.status, &s.rate, &s.currency, &s.billedUntil) == nil {
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
	if hours > hourlyCatchupMax {
		hours = hourlyCatchupMax
	}

	for i := 0; i < hours; i++ {
		periodStart := s.billedUntil.Add(time.Duration(i) * time.Hour)
		paid, err := r.chargeHour(ctx, s, periodStart)
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

func (r *Runner) chargeHour(ctx context.Context, s hourlyServer, periodStart time.Time) (bool, error) {
	amount := math.Round(s.rate*10000) / 10000
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	var walletID string
	var balance float64
	err = tx.QueryRow(ctx, `
		SELECT id::text, balance FROM core.wallets
		WHERE user_id = $1::uuid AND currency = $2
		ORDER BY is_default DESC
		LIMIT 1
		FOR UPDATE
	`, s.userID, s.currency).Scan(&walletID, &balance)
	if err != nil {
		return false, nil
	}
	if balance < amount {
		return false, nil
	}

	if _, err := tx.Exec(ctx, `
		UPDATE core.wallets SET balance = balance - $2, updated_at = now() WHERE id = $1::uuid
	`, walletID, amount); err != nil {
		return false, err
	}

	var txID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO core.transactions ( wallet_id, type, amount, description, source_type, source_id)
		VALUES ($1::uuid, 'debit', $2, $3, 'server_hourly', $4::uuid)
		RETURNING id::text
	`, walletID, -amount, "Почасовая аренда: "+s.name, s.id).Scan(&txID); err != nil {
		return false, err
	}

	res, err := tx.Exec(ctx, `
		INSERT INTO core.server_hourly_charges (server_id, period_start, amount, currency, tx_id)
		VALUES ($1::uuid, $2, $3, $4, $5::uuid)
		ON CONFLICT DO NOTHING
	`, s.id, periodStart, amount, s.currency, txID)
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

	if _, err := tx.Exec(ctx, `
		UPDATE core.servers SET billed_until = $2 WHERE id = $1::uuid
	`, s.id, periodStart.Add(time.Hour)); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
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
	if hoursLeft > hourlyLowBalance {
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
