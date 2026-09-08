package main

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/vortanix/vortanix/internal/status/checks"
	"github.com/vortanix/vortanix/internal/status/store"
)

//go:embed static
var staticFiles embed.FS

const uptimeWindow = 90

type componentView struct {
	checks.Result
	UptimePercent float64     `json:"uptime_percent"`
	UptimeDaily   []store.Day `json:"uptime_daily"`
}

type statusResponse struct {
	Overall     string           `json:"overall"`
	UpdatedAt   string           `json:"updated_at"`
	Components  []componentView  `json:"components"`
	UptimeDaily []store.Day      `json:"uptime_daily,omitempty"`
	Incidents   []store.Incident `json:"incidents,omitempty"`
}

func main() {
	_ = godotenv.Load()

	port := env("PORT", "8090")
	prober := checks.NewProber(
		checks.LoadTargets(),
		envDuration("STATUS_TIMEOUT", 8*time.Second),
		envDuration("STATUS_DEGRADED_AFTER", 3*time.Second),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var db *store.Store
	if url := env("DATABASE_URL", ""); url != "" {
		s, err := store.New(ctx, url, envDuration("STATUS_RETENTION", 24*time.Hour))
		if err != nil {
			log.Fatalf("database: %v", err)
		}
		defer s.Close()
		db = s
		go recordLoop(ctx, prober, db, envDuration("STATUS_CHECK_PERIOD", time.Minute))
	} else {
		log.Printf("DATABASE_URL не задан: график аптайма и инциденты отключены")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/v1/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		writeJSON(w, buildStatus(r.Context(), prober, db))
	})
	if assets, err := fs.Sub(staticFiles, "static"); err == nil {
		mux.Handle("/assets/", http.FileServer(http.FS(assets)))
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		b, err := fs.ReadFile(staticFiles, "static/index.html")
		if err != nil {
			http.Error(w, "page unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	})

	srv := &http.Server{
		Addr:              os.Getenv("BIND_ADDR") + ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("status-page listening on :%s (%d targets)", port, len(prober.Targets()))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func recordLoop(ctx context.Context, prober *checks.Prober, db *store.Store, period time.Duration) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	record(ctx, prober, db)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			record(ctx, prober, db)
		}
	}
}

func record(ctx context.Context, prober *checks.Prober, db *store.Store) {
	if err := db.Save(ctx, prober.Run(ctx)); err != nil {
		log.Printf("status-page: не удалось записать проверку: %v", err)
		return
	}
	if err := db.Prune(ctx); err != nil {
		log.Printf("status-page: очистка истории: %v", err)
	}
}

func buildStatus(ctx context.Context, prober *checks.Prober, db *store.Store) statusResponse {
	results := prober.Run(ctx)
	resp := statusResponse{
		Overall:    checks.Overall(results),
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
		Components: make([]componentView, 0, len(results)),
	}

	for _, r := range results {
		view := componentView{Result: r, UptimePercent: 100}
		if db != nil {
			if percent, err := db.UptimePercent(ctx, r.Key, uptimeWindow); err == nil {
				view.UptimePercent = percent
			}
			if daily, err := db.DailyUptime(ctx, r.Key, uptimeWindow); err == nil {
				view.UptimeDaily = withToday(daily, []checks.Result{r})
			}
		}
		resp.Components = append(resp.Components, view)
	}

	if db != nil {
		if daily, err := db.DailyUptime(ctx, "", uptimeWindow); err == nil {
			resp.UptimeDaily = withToday(daily, results)
		}
		if incidents, err := db.Incidents(ctx, 20); err == nil {
			resp.Incidents = incidents
		}
	}
	return resp
}

func withToday(daily []store.Day, results []checks.Result) []store.Day {
	if len(daily) == 0 || len(results) == 0 {
		return daily
	}
	today := daily[len(daily)-1]
	worst := checks.StatusOperational
	success := 0
	for _, r := range results {
		worst = checks.Worse(worst, r.Status)
		if r.Status == checks.StatusOperational {
			success++
		}
	}
	live := float64(success) / float64(len(results)) * 100
	fresh := today.Status == store.StatusNoData

	if fresh || checks.Severity(worst) > checks.Severity(today.Status) {
		today.Status = worst
	}
	if fresh || live < today.Uptime {
		today.Uptime = float64(int64(live*100+0.5)) / 100
	}
	daily[len(daily)-1] = today
	return daily
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	raw := env(key, "")
	if raw == "" {
		return fallback
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return d
	}
	if n, err := strconv.Atoi(raw); err == nil && n > 0 {
		return time.Duration(n) * time.Second
	}
	return fallback
}
