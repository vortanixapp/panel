package jobs

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

const (
	balanceLowHorizon = 5 * 24 * time.Hour
	diskLowFreeRatio  = 0.1
)

func (r *Runner) HealthWatchLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	r.checkLowBalances(ctx)
	r.checkBalanceThresholds(ctx)
	r.checkNodeDiskSpace(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.checkLowBalances(ctx)
			r.checkBalanceThresholds(ctx)
			r.checkNodeDiskSpace(ctx)
		}
	}
}

func (r *Runner) checkBalanceThresholds(ctx context.Context) {
	rows, err := r.db.Query(ctx, `
		SELECT c.user_id::text, c.balance_threshold::float8, COALESCE(w.balance, 0)::float8, COALESCE(w.currency, 'RUB')
		FROM core.user_notification_channels c
		JOIN core.users u ON u.id = c.user_id AND u.status = 'active'
		LEFT JOIN LATERAL (
			SELECT balance, currency FROM core.wallets w
			WHERE w.user_id = c.user_id
			ORDER BY is_default DESC, created_at
			LIMIT 1
		) w ON true
		WHERE c.balance_threshold IS NOT NULL AND c.balance_threshold > 0
		  AND COALESCE(w.balance, 0) < c.balance_threshold
		LIMIT 1000
	`)
	if err != nil {
		log.Printf("health: проверка порога баланса не выполнена: %v", err)
		return
	}
	type below struct {
		userID, currency   string
		threshold, balance float64
	}
	var list []below
	for rows.Next() {
		var b below
		if rows.Scan(&b.userID, &b.threshold, &b.balance, &b.currency) == nil {
			list = append(list, b)
		}
	}
	rows.Close()

	day := time.Now().Format("2006-01-02")
	for _, b := range list {
		r.notifyUser(ctx, b.userID, notify.Event{
			Kind:  notify.KindBalanceLow,
			Title: i18n.Key("notify.balance_threshold.title"),
			Body: i18n.Key("notify.balance_threshold.body", i18n.Params{
				"balance":   fmt.Sprintf("%.2f", b.balance),
				"threshold": fmt.Sprintf("%.2f", b.threshold),
				"currency":  b.currency,
			}),
			Action:    r.panelAction("notify.action.topup", "/billing#topup"),
			Meta:      map[string]any{"balance": b.balance, "threshold": b.threshold, "currency": b.currency},
			DedupeKey: "balance.threshold:" + b.userID + ":" + day,
		})
	}
}

func (r *Runner) checkLowBalances(ctx context.Context) {
	rows, err := r.db.Query(ctx, `
		WITH due AS (
			SELECT s.user_id,
			       SUM(COALESCE(t.price_monthly, 0))::float8 AS amount,
			       MIN(s.expires_at) AS soonest
			FROM core.servers s
			LEFT JOIN core.tariffs t ON t.id = s.tariff_id
			WHERE s.user_id IS NOT NULL
			  AND s.expires_at IS NOT NULL
			  AND s.expires_at > now()
			  AND s.expires_at < now() + $1::interval
			  AND s.suspended_at IS NULL
			GROUP BY s.user_id
		)
		SELECT d.user_id::text, d.amount,
		       COALESCE(w.balance, 0)::float8, COALESCE(w.currency, 'RUB'), d.soonest
		FROM due d
		LEFT JOIN LATERAL (
			SELECT balance, currency FROM core.wallets w
			WHERE w.user_id = d.user_id
			ORDER BY is_default DESC, created_at
			LIMIT 1
		) w ON true
		WHERE d.amount > 0 AND COALESCE(w.balance, 0) < d.amount
		LIMIT 500
	`, fmt.Sprintf("%d hours", int(balanceLowHorizon.Hours())))
	if err != nil {
		log.Printf("health: проверка баланса не выполнена: %v", err)
		return
	}
	defer rows.Close()

	type lowBalance struct {
		userID, currency string
		due, balance     float64
		soonest          time.Time
	}
	var list []lowBalance
	for rows.Next() {
		var lb lowBalance
		if rows.Scan(&lb.userID, &lb.due, &lb.balance, &lb.currency, &lb.soonest) == nil {
			list = append(list, lb)
		}
	}

	for _, lb := range list {
		if lb.due <= 0 {
			continue
		}
		r.notifyUser(ctx, lb.userID, notify.Event{
			Kind:  notify.KindBalanceLow,
			Title: i18n.Key("notify.balance_low.title"),
			Body: i18n.Key("notify.balance_low.body", i18n.Params{
				"date":     lb.soonest.Format("02.01.2006"),
				"due":      fmt.Sprintf("%.2f", lb.due),
				"balance":  fmt.Sprintf("%.2f", lb.balance),
				"currency": lb.currency,
			}),
			Action: r.panelAction("notify.action.topup", "/billing"),
			Meta: map[string]any{
				"due": lb.due, "balance": lb.balance, "currency": lb.currency,
			},
			DedupeKey: "balance.low:" + lb.userID + ":" + time.Now().Format("2006-01-02"),
		})
	}
}

func (r *Runner) checkNodeDiskSpace(ctx context.Context) {
	rows, err := r.db.Query(ctx, `
		SELECT n.id::text, COALESCE(n.name, n.fqdn, ''),
		       CASE WHEN d.stats_at > now() - interval '10 minutes'
		            THEN COALESCE((d.stats->'host'->>'disk_total_mb')::float8, 0) ELSE 0 END,
		       CASE WHEN d.stats_at > now() - interval '10 minutes'
		            THEN COALESCE((d.stats->'host'->>'disk_free_mb')::float8, 0) ELSE 0 END,
		       COALESCE(m.total, ''), COALESCE(m.avail, '')
		FROM core.nodes n
		LEFT JOIN core.node_daemons d ON d.node_id = n.id
		LEFT JOIN LATERAL (
			SELECT
				MAX(text_value) FILTER (WHERE metric_type = 'disk_total') AS total,
				MAX(text_value) FILTER (WHERE metric_type = 'disk_available') AS avail
			FROM core.node_metrics
			WHERE node_id = n.id
			  AND metric_type IN ('disk_total', 'disk_available')
			  AND measured_at > now() - interval '6 hours'
		) m ON true
		LIMIT 500
	`)
	if err != nil {
		log.Printf("health: проверка дисков не выполнена: %v", err)
		return
	}
	defer rows.Close()

	type lowDisk struct {
		nodeID, name string
		total, avail float64
	}
	var list []lowDisk
	for rows.Next() {
		var nodeID, name, totalText, availText string
		var totalMB, freeMB float64
		if rows.Scan(&nodeID, &name, &totalMB, &freeMB, &totalText, &availText) != nil {
			continue
		}
		total, avail := totalMB*1024*1024, freeMB*1024*1024
		if totalMB <= 0 {
			var okTotal, okAvail bool
			total, okTotal = parseSizeBytes(totalText)
			avail, okAvail = parseSizeBytes(availText)
			if !okTotal || !okAvail {
				continue
			}
		}
		if total <= 0 || avail >= total*diskLowFreeRatio {
			continue
		}
		list = append(list, lowDisk{
			nodeID: nodeID, name: name,
			total: total, avail: avail,
		})
	}

	for _, ld := range list {
		name := i18n.Raw(ld.name)
		if ld.name == "" {
			name = i18n.Key("notify.disk_low.unnamed")
		}
		percent := 0.0
		if ld.total > 0 {
			percent = ld.avail / ld.total * 100
		}
		r.notifyStaff(ctx, notify.Event{
			Kind:  notify.KindDiskLow,
			Title: i18n.Key("notify.disk_low.title"),
			Body: i18n.Key("notify.disk_low.body", i18n.Params{
				"node":    name,
				"percent": fmt.Sprintf("%.1f", percent),
				"free":    sizeMsg(ld.avail),
				"total":   sizeMsg(ld.total),
			}),
			Action: r.panelAction("notify.action.agent", "/admin/daemons/"+ld.nodeID),
			Meta: map[string]any{
				"node_id": ld.nodeID, "free_bytes": ld.avail, "total_bytes": ld.total,
			},
			DedupeKey: "disk.low:" + ld.nodeID + ":" + time.Now().Format("2006-01-02"),
		})
	}
}

func parseSizeBytes(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0, false
	}
	s = strings.TrimSuffix(s, "B")
	s = strings.TrimSuffix(s, "I")
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}

	mult := 1.0
	switch s[len(s)-1] {
	case 'K':
		mult = 1 << 10
	case 'M':
		mult = 1 << 20
	case 'G':
		mult = 1 << 30
	case 'T':
		mult = 1 << 40
	case 'P':
		mult = 1 << 50
	}
	if mult > 1 {
		s = strings.TrimSpace(s[:len(s)-1])
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	if err != nil || v < 0 {
		return 0, false
	}
	return v * mult, true
}

func sizeMsg(v float64) i18n.Msg {
	const unit = 1024.0
	units := []string{"b", "kb", "mb", "gb", "tb", "pb"}
	i := 0
	for v >= unit && i < len(units)-1 {
		v /= unit
		i++
	}
	value := fmt.Sprintf("%.1f", v)
	if i == 0 {
		value = fmt.Sprintf("%.0f", v)
	}
	return i18n.Key("notify.size."+units[i], i18n.Params{"value": value})
}
