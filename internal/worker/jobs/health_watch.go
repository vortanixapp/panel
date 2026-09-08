package jobs

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/notify"
)

// Наблюдение за тем, что ломается не сразу: кончающиеся деньги и кончающееся
// место на диске.
//
// События balance.low и disk.low были объявлены в справочнике оповещений, но не
// отправлялись ниоткуда. Клиент узнавал о нехватке денег в момент остановки
// сервера, а владелец панели о заполненном диске — когда на ноду переставали
// вставать новые серверы.

const (
	// За сколько дней до конца аренды считаем, что деньги нужны уже сейчас.
	balanceLowHorizon = 5 * 24 * time.Hour
	// Порог свободного места на ноде: ниже этой доли шлём предупреждение.
	diskLowFreeRatio = 0.1
)

func (r *Runner) HealthWatchLoop(ctx context.Context) {
	// Раз в час: и низкий баланс, и заполняющийся диск — события медленные,
	// проверять их чаще бессмысленно, а письмо каждую минуту вредно.
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	r.checkLowBalances(ctx)
	r.checkNodeDiskSpace(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.checkLowBalances(ctx)
			r.checkNodeDiskSpace(ctx)
		}
	}
}

// checkLowBalances ищет тех, кому не хватит денег на продление.
//
// Сравниваем не с произвольным порогом в рублях, а со стоимостью того, что
// скоро придётся оплатить: «мало» — это когда на счету меньше, чем стоит
// продление собственных серверов.
func (r *Runner) checkLowBalances(ctx context.Context) {
	// Кошелёк подставляется после подсчёта суммы, а не соединением: у клиента
	// их может быть несколько, и соединение множило бы строки серверов, а с
	// ними и сумму к оплате.
	rows, err := r.db.Query(ctx, `
		WITH due AS (
			SELECT s.tenant_id, s.user_id,
			       SUM(COALESCE(t.price_monthly, 0))::float8 AS amount,
			       MIN(s.expires_at) AS soonest
			FROM core.servers s
			LEFT JOIN core.tariffs t ON t.id = s.tariff_id
			WHERE s.user_id IS NOT NULL
			  AND s.expires_at IS NOT NULL
			  AND s.expires_at > now()
			  AND s.expires_at < now() + $1::interval
			  AND s.suspended_at IS NULL
			GROUP BY s.tenant_id, s.user_id
		)
		SELECT d.tenant_id::text, d.user_id::text, d.amount,
		       COALESCE(w.balance, 0)::float8, COALESCE(w.currency, 'RUB'), d.soonest
		FROM due d
		LEFT JOIN LATERAL (
			SELECT balance, currency FROM core.wallets w
			WHERE w.user_id = d.user_id AND w.tenant_id = d.tenant_id
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
		tenantID, userID, currency string
		due, balance               float64
		soonest                    time.Time
	}
	var list []lowBalance
	for rows.Next() {
		var lb lowBalance
		if rows.Scan(&lb.tenantID, &lb.userID, &lb.due, &lb.balance, &lb.currency, &lb.soonest) == nil {
			list = append(list, lb)
		}
	}

	for _, lb := range list {
		if lb.due <= 0 {
			continue
		}
		r.notifyUser(ctx, lb.tenantID, lb.userID, notify.Event{
			Kind:  notify.KindBalanceLow,
			Title: "На счету не хватает средств",
			Body: fmt.Sprintf(
				"До %s нужно продлить серверы на %.2f %s, а на счету %.2f %s. "+
					"Пополните баланс, иначе серверы будут остановлены.",
				lb.soonest.Format("02.01.2006"), lb.due, lb.currency, lb.balance, lb.currency),
			Action: r.panelAction("Пополнить баланс", "/billing/topup"),
			Meta: map[string]any{
				"due": lb.due, "balance": lb.balance, "currency": lb.currency,
			},
			// Один раз в сутки на человека: пока баланс не пополнен, условие
			// выполняется на каждом проходе цикла.
			DedupeKey: "balance.low:" + lb.userID + ":" + time.Now().Format("2006-01-02"),
		})
	}
}

// checkNodeDiskSpace предупреждает персонал о заканчивающемся месте на нодах.
func (r *Runner) checkNodeDiskSpace(ctx context.Context) {
	// Размеры лежат в text_value: сборщик пишет их строкой (df -B1 отдаёт байты,
	// более старые сборы — вида «50G»). Приводить типы в SQL нельзя, одна
	// нечисловая строка уронила бы весь запрос, поэтому разбираем в Go.
	rows, err := r.db.Query(ctx, `
		SELECT n.tenant_id::text, n.id::text, COALESCE(n.fqdn, ''),
		       COALESCE(m.total, ''), COALESCE(m.avail, '')
		FROM core.nodes n
		JOIN LATERAL (
			SELECT
				MAX(text_value) FILTER (WHERE metric_type = 'disk_total') AS total,
				MAX(text_value) FILTER (WHERE metric_type = 'disk_available') AS avail
			FROM core.node_metrics
			WHERE node_id = n.id
			  AND metric_type IN ('disk_total', 'disk_available')
			  AND measured_at > now() - interval '6 hours'
		) m ON true
		WHERE m.total IS NOT NULL AND m.avail IS NOT NULL
		LIMIT 500
	`)
	if err != nil {
		log.Printf("health: проверка дисков не выполнена: %v", err)
		return
	}
	defer rows.Close()

	type lowDisk struct {
		tenantID, nodeID, name string
		total, avail           float64
	}
	var list []lowDisk
	for rows.Next() {
		var tenantID, nodeID, name, totalText, availText string
		if rows.Scan(&tenantID, &nodeID, &name, &totalText, &availText) != nil {
			continue
		}
		total, okTotal := parseSizeBytes(totalText)
		avail, okAvail := parseSizeBytes(availText)
		if !okTotal || !okAvail || total <= 0 || avail >= total*diskLowFreeRatio {
			continue
		}
		list = append(list, lowDisk{
			tenantID: tenantID, nodeID: nodeID, name: name,
			total: total, avail: avail,
		})
	}

	for _, ld := range list {
		name := ld.name
		if name == "" {
			name = "нода"
		}
		percent := 0.0
		if ld.total > 0 {
			percent = ld.avail / ld.total * 100
		}
		r.notifyTenantStaff(ctx, ld.tenantID, notify.Event{
			Kind:  notify.KindDiskLow,
			Title: "Мало места на ноде",
			Body: fmt.Sprintf(
				"На ноде «%s» осталось %.1f%% свободного места (%s из %s). "+
					"Новые серверы на неё встать не смогут.",
				name, percent, humanBytes(ld.avail), humanBytes(ld.total)),
			Action: r.panelAction("Открыть локации", "/admin/locations"),
			Meta: map[string]any{
				"node_id": ld.nodeID, "free_bytes": ld.avail, "total_bytes": ld.total,
			},
			DedupeKey: "disk.low:" + ld.nodeID + ":" + time.Now().Format("2006-01-02"),
		})
	}
}

// notifyTenantStaff рассылает событие владельцу панели и администраторам.
//
// Диск и обновления — забота персонала, а не клиента: клиенту сообщать не о чем,
// сделать он с этим ничего не может.
func (r *Runner) notifyTenantStaff(ctx context.Context, tenantID string, e notify.Event) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text FROM core.users
		WHERE tenant_id = $1 AND role IN ('owner', 'admin')
	`, tenantID)
	if err != nil {
		log.Printf("оповещение %s: персонал арендатора %s не найден: %v", e.Kind, tenantID, err)
		return
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		r.notifyUser(ctx, tenantID, id, e)
	}
}

// parseSizeBytes разбирает размер из метрики: и голые байты, и «50G», «512M».
func parseSizeBytes(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0, false
	}
	// «ГиБ» и «Г» означают одно и то же, поэтому хвост единиц снимаем до
	// разбора буквы множителя: иначе «10GiB» не разбирается вовсе.
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

// humanBytes — размер в привычном виде, для текста письма.
func humanBytes(v float64) string {
	const unit = 1024.0
	units := []string{"Б", "КБ", "МБ", "ГБ", "ТБ", "ПБ"}
	i := 0
	for v >= unit && i < len(units)-1 {
		v /= unit
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f %s", v, units[i])
	}
	return fmt.Sprintf("%.1f %s", v, units[i])
}
