package db

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

const defaultSlowQuery = 500 * time.Millisecond

var (
	metricsOnce sync.Once
	queryTime   *prometheus.HistogramVec
	slowQueries *prometheus.CounterVec
)

type queryStart struct {
	at  time.Time
	sql string
}

type queryStartKey struct{}

type queryTracer struct {
	pool string
	slow time.Duration
}

func slowThreshold() time.Duration {
	raw := strings.TrimSpace(os.Getenv("DB_SLOW_QUERY_MS"))
	if raw == "" {
		return defaultSlowQuery
	}
	ms, err := strconv.Atoi(raw)
	if err != nil || ms < 0 {
		return defaultSlowQuery
	}
	return time.Duration(ms) * time.Millisecond
}

func queryKind(sql string) string {
	sql = strings.TrimSpace(sql)
	if end := strings.IndexAny(sql, " \t\r\n("); end > 0 {
		sql = sql[:end]
	}
	switch kind := strings.ToLower(sql); kind {
	case "select", "insert", "update", "delete", "with", "begin", "commit", "rollback":
		return kind
	}
	return "other"
}

func shortSQL(sql string) string {
	sql = strings.Join(strings.Fields(sql), " ")
	if len(sql) > 300 {
		return sql[:300] + "..."
	}
	return sql
}

func initMetrics() {
	metricsOnce.Do(func() {
		queryTime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query latency",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		}, []string{"pool", "kind"})
		slowQueries = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "db_slow_queries_total",
			Help: "Queries slower than DB_SLOW_QUERY_MS",
		}, []string{"pool"})
		register(queryTime)
		register(slowQueries)
	})
}

func register(c prometheus.Collector) {
	if err := prometheus.Register(c); err != nil {
		if _, dup := err.(prometheus.AlreadyRegisteredError); !dup {
			log.Printf("db metrics: %v", err)
		}
	}
}

func (t *queryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	return context.WithValue(ctx, queryStartKey{}, queryStart{at: time.Now(), sql: data.SQL})
}

func (t *queryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
	started, ok := ctx.Value(queryStartKey{}).(queryStart)
	if !ok {
		return
	}
	elapsed := time.Since(started.at)
	queryTime.WithLabelValues(t.pool, queryKind(started.sql)).Observe(elapsed.Seconds())
	if t.slow > 0 && elapsed >= t.slow {
		slowQueries.WithLabelValues(t.pool).Inc()
		log.Printf("slow query pool=%s took=%s sql=%s", t.pool, elapsed.Round(time.Millisecond), shortSQL(started.sql))
	}
}

type poolCollector struct {
	pool     *pgxpool.Pool
	acquired *prometheus.Desc
	idle     *prometheus.Desc
	total    *prometheus.Desc
	max      *prometheus.Desc
	waits    *prometheus.Desc
	waitSecs *prometheus.Desc
}

func newPoolCollector(name string, pool *pgxpool.Pool) *poolCollector {
	labels := prometheus.Labels{"pool": name}
	desc := func(metric, help string) *prometheus.Desc {
		return prometheus.NewDesc("db_pool_"+metric, help, nil, labels)
	}
	return &poolCollector{
		pool:     pool,
		acquired: desc("acquired_conns", "Connections in use"),
		idle:     desc("idle_conns", "Idle connections"),
		total:    desc("total_conns", "Open connections"),
		max:      desc("max_conns", "Pool size limit"),
		waits:    desc("empty_acquire_total", "Acquires that had to wait for a free connection"),
		waitSecs: desc("acquire_wait_seconds_total", "Total time spent waiting for a connection"),
	}
}

func (c *poolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.acquired
	ch <- c.idle
	ch <- c.total
	ch <- c.max
	ch <- c.waits
	ch <- c.waitSecs
}

func (c *poolCollector) Collect(ch chan<- prometheus.Metric) {
	s := c.pool.Stat()
	ch <- prometheus.MustNewConstMetric(c.acquired, prometheus.GaugeValue, float64(s.AcquiredConns()))
	ch <- prometheus.MustNewConstMetric(c.idle, prometheus.GaugeValue, float64(s.IdleConns()))
	ch <- prometheus.MustNewConstMetric(c.total, prometheus.GaugeValue, float64(s.TotalConns()))
	ch <- prometheus.MustNewConstMetric(c.max, prometheus.GaugeValue, float64(s.MaxConns()))
	ch <- prometheus.MustNewConstMetric(c.waits, prometheus.CounterValue, float64(s.EmptyAcquireCount()))
	ch <- prometheus.MustNewConstMetric(c.waitSecs, prometheus.CounterValue, s.AcquireDuration().Seconds())
}

func newObservedPool(ctx context.Context, name, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	initMetrics()
	cfg.ConnConfig.Tracer = &queryTracer{pool: name, slow: slowThreshold()}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	register(newPoolCollector(name, pool))
	return pool, nil
}
