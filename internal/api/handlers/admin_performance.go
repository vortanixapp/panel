package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultPrometheusURL = "http://prometheus:9090"
	prometheusTimeout    = 8 * time.Second
	minRateWindowSec     = 60
)

type perfRange struct {
	span time.Duration
	step time.Duration
}

var perfRanges = map[string]perfRange{
	"1h":  {time.Hour, 15 * time.Second},
	"6h":  {6 * time.Hour, time.Minute},
	"24h": {24 * time.Hour, 5 * time.Minute},
	"7d":  {7 * 24 * time.Hour, 30 * time.Minute},
}

type perfQuery struct {
	expr   string
	labels []string
	prefix string
}

type perfChart struct {
	id      string
	unit    string
	queries []perfQuery
}

type perfSeries struct {
	Name   string       `json:"name"`
	Points [][2]float64 `json:"points"`
}

type perfChartResult struct {
	ID     string       `json:"id"`
	Unit   string       `json:"unit"`
	Series []perfSeries `json:"series"`
}

func perfCharts(window string) []perfChart {
	api := `service="core-api"`
	apiLatency := `service="core-api",route!~".*/stream"`
	relay := `service="agent-relay"`
	return []perfChart{
		{id: "api_rps", unit: "reqps", queries: []perfQuery{{expr: fmt.Sprintf(`sum(rate(http_requests_total{%s}[%s]))`, api, window)}}},
		{id: "api_routes_rps", unit: "reqps", queries: []perfQuery{{expr: fmt.Sprintf(`topk(8, sum by (route) (rate(http_requests_total{%s}[%s])))`, api, window), labels: []string{"route"}}}},
		{id: "api_p95", unit: "s", queries: []perfQuery{{expr: fmt.Sprintf(`histogram_quantile(0.95, sum by (le) (rate(http_request_duration_seconds_bucket{%s}[%s])))`, apiLatency, window)}}},
		{id: "api_routes_p95", unit: "s", queries: []perfQuery{{expr: fmt.Sprintf(`topk(8, histogram_quantile(0.95, sum by (le, route) (rate(http_request_duration_seconds_bucket{%s}[%s]))))`, apiLatency, window), labels: []string{"route"}}}},
		{id: "api_5xx", unit: "reqps", queries: []perfQuery{{expr: fmt.Sprintf(`sum(rate(http_requests_total{%s,status=~"5.."}[%s]))`, api, window)}}},
		{id: "db_p95", unit: "s", queries: []perfQuery{{expr: fmt.Sprintf(`histogram_quantile(0.95, sum by (le, pool) (rate(db_query_duration_seconds_bucket[%s])))`, window), labels: []string{"pool"}}}},
		{id: "db_slow", unit: "perMin", queries: []perfQuery{{expr: fmt.Sprintf(`sum by (pool) (rate(db_slow_queries_total[%s])) * 60`, window), labels: []string{"pool"}}}},
		{id: "db_pool", unit: "short", queries: []perfQuery{
			{expr: `db_pool_acquired_conns`, labels: []string{"pool"}, prefix: "busy "},
			{expr: `db_pool_max_conns`, labels: []string{"pool"}, prefix: "limit "},
		}},
		{id: "db_pool_wait", unit: "s", queries: []perfQuery{{expr: fmt.Sprintf(`rate(db_pool_acquire_wait_seconds_total[%s])`, window), labels: []string{"pool"}}}},
		{id: "goroutines", unit: "short", queries: []perfQuery{{expr: `go_goroutines{job="api"}`}}},
		{id: "memory", unit: "bytes", queries: []perfQuery{{expr: `process_resident_memory_bytes{job="api"}`}}},
		{id: "relay_rps", unit: "reqps", queries: []perfQuery{{expr: fmt.Sprintf(`sum(rate(http_requests_total{%s}[%s]))`, relay, window)}}},
		{id: "relay_p95", unit: "s", queries: []perfQuery{{expr: fmt.Sprintf(`histogram_quantile(0.95, sum by (le) (rate(http_request_duration_seconds_bucket{%s}[%s])))`, relay, window)}}},
	}
}

func prometheusBase() string {
	if raw := strings.TrimRight(strings.TrimSpace(os.Getenv("PROMETHEUS_URL")), "/"); raw != "" {
		return raw
	}
	return defaultPrometheusURL
}

type promMatrix struct {
	Status string `json:"status"`
	Data   struct {
		Result []struct {
			Metric map[string]string `json:"metric"`
			Values [][2]any          `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

func promRange(ctx context.Context, client *http.Client, query string, start, end time.Time, step time.Duration) (*promMatrix, error) {
	form := url.Values{}
	form.Set("query", query)
	form.Set("start", strconv.FormatInt(start.Unix(), 10))
	form.Set("end", strconv.FormatInt(end.Unix(), 10))
	form.Set("step", strconv.Itoa(int(step.Seconds())))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, prometheusBase()+"/api/v1/query_range", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus: %s", resp.Status)
	}
	var out promMatrix
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Status != "success" {
		return nil, fmt.Errorf("prometheus: %s", out.Status)
	}
	return &out, nil
}

func seriesName(metric map[string]string, labels []string, prefix string) string {
	parts := make([]string, 0, len(labels))
	for _, key := range labels {
		if v := metric[key]; v != "" {
			parts = append(parts, v)
		}
	}
	return prefix + strings.Join(parts, " ")
}

func matrixToSeries(m *promMatrix, q perfQuery) []perfSeries {
	out := make([]perfSeries, 0, len(m.Data.Result))
	for _, r := range m.Data.Result {
		points := make([][2]float64, 0, len(r.Values))
		for _, v := range r.Values {
			ts, okTs := v[0].(float64)
			raw, okVal := v[1].(string)
			if !okTs || !okVal {
				continue
			}
			val, err := strconv.ParseFloat(raw, 64)
			if err != nil || math.IsNaN(val) || math.IsInf(val, 0) {
				continue
			}
			points = append(points, [2]float64{ts, val})
		}
		if len(points) == 0 {
			continue
		}
		out = append(out, perfSeries{Name: seriesName(r.Metric, q.labels, q.prefix), Points: points})
	}
	return out
}

func (h *Handler) AdminPerformance(w http.ResponseWriter, r *http.Request) {
	span, ok := perfRanges[r.URL.Query().Get("range")]
	if !ok {
		span = perfRanges["1h"]
	}
	window := fmt.Sprintf("%ds", max(minRateWindowSec, int(span.step.Seconds())*4))
	charts := perfCharts(window)

	ctx, cancel := context.WithTimeout(r.Context(), prometheusTimeout)
	defer cancel()
	client := &http.Client{Timeout: prometheusTimeout}
	end := time.Now()
	start := end.Add(-span.span)

	results := make([]perfChartResult, len(charts))
	var mu sync.Mutex
	var wg sync.WaitGroup
	succeeded := 0
	for i, chart := range charts {
		results[i] = perfChartResult{ID: chart.id, Unit: chart.unit, Series: []perfSeries{}}
		for _, q := range chart.queries {
			wg.Add(1)
			go func(i int, q perfQuery) {
				defer wg.Done()
				matrix, err := promRange(ctx, client, q.expr, start, end, span.step)
				if err != nil {
					return
				}
				series := matrixToSeries(matrix, q)
				mu.Lock()
				results[i].Series = append(results[i].Series, series...)
				succeeded++
				mu.Unlock()
			}(i, q)
		}
	}
	wg.Wait()

	if succeeded == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"available": false, "charts": []perfChartResult{}})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"available": true,
		"step":      int(span.step.Seconds()),
		"charts":    results,
	})
}
