package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vortanixapp/panel/internal/status/checks"
)

type Store struct {
	db        *pgxpool.Pool
	retention time.Duration
}

func New(ctx context.Context, databaseURL string, retention time.Duration) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Store{db: pool, retention: retention}, nil
}

func (s *Store) Close() { s.db.Close() }

func (s *Store) Save(ctx context.Context, results []checks.Result) error {
	if len(results) == 0 {
		return nil
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now()
	for _, r := range results {
		if _, err := tx.Exec(ctx, `
			INSERT INTO status.health_checks (service, status, response_time_ms, error, checked_at)
			VALUES ($1, $2, $3, NULLIF($4,''), $5)
		`, r.Key, r.Status, r.LatencyMS, r.Detail, now); err != nil {
			return err
		}

		success := 0
		if r.Status == checks.StatusOperational {
			success = 1
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO status.uptime_daily (service, day, total_checks, successful_checks, worst_status)
			VALUES ($1, $2::date, 1, $3, $4)
			ON CONFLICT (service, day) DO UPDATE SET
				total_checks      = status.uptime_daily.total_checks + 1,
				successful_checks = status.uptime_daily.successful_checks + EXCLUDED.successful_checks,
				worst_status      = CASE
					WHEN $5 > CASE status.uptime_daily.worst_status
						WHEN 'down' THEN 2 WHEN 'degraded' THEN 1 ELSE 0 END
					THEN EXCLUDED.worst_status
					ELSE status.uptime_daily.worst_status
				END
		`, r.Key, now, success, r.Status, checks.Severity(r.Status)); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *Store) Prune(ctx context.Context) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM status.health_checks WHERE checked_at < now() - $1::interval
	`, s.retention.String())
	return err
}

type Day struct {
	Date   string  `json:"date"`
	Uptime float64 `json:"uptime"`
	Status string  `json:"status"`
}

const StatusNoData = "no_data"

func (s *Store) DailyUptime(ctx context.Context, service string, days int) ([]Day, error) {
	rows, err := s.db.Query(ctx, `
		WITH span AS (
			SELECT generate_series(
				(current_date - ($2::int - 1)),
				current_date,
				'1 day'::interval
			)::date AS day
		)
		SELECT span.day,
		       COALESCE(sum(u.total_checks), 0),
		       COALESCE(sum(u.successful_checks), 0),
		       COALESCE(max(CASE u.worst_status
		           WHEN 'down' THEN 2 WHEN 'degraded' THEN 1 ELSE 0 END), -1)
		FROM span
		LEFT JOIN status.uptime_daily u
		       ON u.day = span.day AND ($1 = '' OR u.service = $1)
		GROUP BY span.day
		ORDER BY span.day
	`, service, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Day, 0, days)
	for rows.Next() {
		var (
			day      time.Time
			total    int64
			success  int64
			severity int
		)
		if err := rows.Scan(&day, &total, &success, &severity); err != nil {
			return nil, err
		}
		d := Day{Date: day.Format("2006-01-02"), Uptime: 100, Status: StatusNoData}
		if total > 0 {
			d.Uptime = round2(float64(success) / float64(total) * 100)
			d.Status = statusFromSeverity(severity)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) UptimePercent(ctx context.Context, service string, days int) (float64, error) {
	var total, success int64
	err := s.db.QueryRow(ctx, `
		SELECT COALESCE(sum(total_checks), 0), COALESCE(sum(successful_checks), 0)
		FROM status.uptime_daily
		WHERE day >= current_date - ($2::int - 1) AND ($1 = '' OR service = $1)
	`, service, days).Scan(&total, &success)
	if err != nil {
		return 0, err
	}
	if total == 0 {
		return 100, nil
	}
	return round2(float64(success) / float64(total) * 100), nil
}

type Incident struct {
	Service   string    `json:"service"`
	Status    string    `json:"status"`
	Detail    string    `json:"detail,omitempty"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Minutes   int       `json:"minutes"`
}

func (s *Store) Incidents(ctx context.Context, limit int) ([]Incident, error) {
	rows, err := s.db.Query(ctx, `
		WITH marked AS (
			SELECT service, status, error, checked_at,
			       CASE WHEN checked_at - lag(checked_at) OVER w > interval '5 minutes'
			                 OR lag(checked_at) OVER w IS NULL
			            THEN 1 ELSE 0 END AS is_new
			FROM status.health_checks
			WHERE status <> 'operational'
			WINDOW w AS (PARTITION BY service ORDER BY checked_at)
		), grouped AS (
			SELECT service, status, error, checked_at,
			       sum(is_new) OVER (PARTITION BY service ORDER BY checked_at) AS grp
			FROM marked
		)
		SELECT service,
		       max(CASE status WHEN 'down' THEN 'down' ELSE 'degraded' END),
		       COALESCE(max(error), ''),
		       min(checked_at), max(checked_at)
		FROM grouped
		GROUP BY service, grp
		ORDER BY min(checked_at) DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Incident, 0, limit)
	for rows.Next() {
		var in Incident
		if err := rows.Scan(&in.Service, &in.Status, &in.Detail, &in.StartedAt, &in.EndedAt); err != nil {
			return nil, err
		}
		in.Minutes = int(in.EndedAt.Sub(in.StartedAt).Minutes())
		out = append(out, in)
	}
	return out, rows.Err()
}

func statusFromSeverity(severity int) string {
	switch severity {
	case 2:
		return checks.StatusDown
	case 1:
		return checks.StatusDegraded
	case 0:
		return checks.StatusOperational
	default:
		return StatusNoData
	}
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100
}
