package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type backupScheduleBody struct {
	Enabled   *bool   `json:"enabled"`
	Frequency *string `json:"frequency"`
	HourUTC   *int    `json:"hour_utc"`
	DayOfWeek *int    `json:"day_of_week"`
	KeepCount *int    `json:"keep_count"`
}

func (h *Handler) GetServerBackupSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims, ok := h.authorizeServerTab(w, r, id, "backup_schedule_read")
	if !ok {
		return
	}
	var (
		enabled              bool
		frequency            string
		hour, dow, keepCount int
		lastRun, lastErr     *string
	)
	err := h.readerOf(r.Context()).QueryRow(r.Context(), `
		SELECT enabled, frequency, hour_utc, day_of_week, keep_count,
		       last_run_at::text, last_error
		FROM core.server_backup_schedules WHERE server_id = $1 AND tenant_id = $2
	`, id, claims.TenantID).Scan(&enabled, &frequency, &hour, &dow, &keepCount, &lastRun, &lastErr)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled": false, "frequency": "daily", "hour_utc": 4,
			"day_of_week": 0, "keep_count": 7,
			"last_run_at": nil, "last_error": nil,
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled": enabled, "frequency": frequency, "hour_utc": hour,
		"day_of_week": dow, "keep_count": keepCount,
		"last_run_at": lastRun, "last_error": lastErr,
	})
}

// Расписание бэкапов правит владелец или тот, кому доверены настройки: раньше
// достаточно было состоять в том же арендаторе.
func (h *Handler) PutServerBackupSchedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	claims, ok := h.authorizeServerTab(w, r, id, "backup_schedule_write")
	if !ok {
		return
	}
	var body backupScheduleBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	enabled := body.Enabled != nil && *body.Enabled
	frequency := "daily"
	if body.Frequency != nil {
		frequency = *body.Frequency
	}
	if frequency != "daily" && frequency != "weekly" {
		writeError(w, http.StatusBadRequest, "frequency must be daily or weekly")
		return
	}
	hour := 4
	if body.HourUTC != nil {
		hour = *body.HourUTC
	}
	if hour < 0 || hour > 23 {
		writeError(w, http.StatusBadRequest, "hour_utc must be 0..23")
		return
	}
	dow := 0
	if body.DayOfWeek != nil {
		dow = *body.DayOfWeek
	}
	if dow < 0 || dow > 6 {
		writeError(w, http.StatusBadRequest, "day_of_week must be 0..6")
		return
	}
	keep := 7
	if body.KeepCount != nil {
		keep = *body.KeepCount
	}
	if keep < 1 || keep > 30 {
		writeError(w, http.StatusBadRequest, "keep_count must be 1..30")
		return
	}

	if _, err := h.dbOf(r.Context()).Exec(r.Context(), `
		INSERT INTO core.server_backup_schedules
			(server_id, tenant_id, enabled, frequency, hour_utc, day_of_week, keep_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (server_id) DO UPDATE SET
			enabled     = EXCLUDED.enabled,
			frequency   = EXCLUDED.frequency,
			hour_utc    = EXCLUDED.hour_utc,
			day_of_week = EXCLUDED.day_of_week,
			keep_count  = EXCLUDED.keep_count,
			updated_at  = now()
	`, id, claims.TenantID, enabled, frequency, hour, dow, keep); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "server.backup_schedule", "server:"+id,
		map[string]any{"enabled": enabled, "frequency": frequency, "hour_utc": hour})
	writeJSON(w, http.StatusOK, map[string]any{
		"enabled": enabled, "frequency": frequency, "hour_utc": hour,
		"day_of_week": dow, "keep_count": keep,
	})
}
