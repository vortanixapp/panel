package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/vortanixapp/panel/internal/api/payments"
	"github.com/vortanixapp/panel/pkg/regions"
)

type taxRateRow struct {
	ID          int64   `json:"id"`
	Country     string  `json:"country"`
	Subdivision string  `json:"subdivision"`
	Label       string  `json:"label"`
	Rate        float64 `json:"rate"`
	Enabled     bool    `json:"enabled"`
	Region      string  `json:"region"`
}

func (h *Handler) AdminTaxRates(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	rows, err := h.readerOf(ctx).Query(ctx, `
		SELECT id, country, subdivision, label, rate::float8, enabled
		FROM core.tax_rates
		ORDER BY country, subdivision
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()

	items := []taxRateRow{}
	for rows.Next() {
		var it taxRateRow
		if rows.Scan(&it.ID, &it.Country, &it.Subdivision, &it.Label, &it.Rate, &it.Enabled) != nil {
			continue
		}
		it.Region = regions.RegionOf(it.Country)
		items = append(items, it)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"rates":          items,
		"seller_country": h.sellerCountry(h.loadTenantSettingStrings(ctx)),
	})
}

func (h *Handler) AdminTaxRatesUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Rates []struct {
			ID      int64    `json:"id"`
			Rate    *float64 `json:"rate"`
			Enabled *bool    `json:"enabled"`
		} `json:"rates"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Rates) == 0 || len(body.Rates) > 500 {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	for _, it := range body.Rates {
		if it.Rate != nil && (*it.Rate < 0 || *it.Rate > 100) {
			writeError(w, http.StatusUnprocessableEntity, "Ставка налога — от 0 до 100 %")
			return
		}
	}
	ctx := r.Context()
	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer tx.Rollback(ctx)
	for _, it := range body.Rates {
		if _, err := tx.Exec(ctx, `
			UPDATE core.tax_rates SET
				rate = COALESCE($2::numeric, rate),
				enabled = COALESCE($3, enabled),
				updated_at = now()
			WHERE id = $1
		`, it.ID, it.Rate, it.Enabled); err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "settings.update", "accounting/tax-rates",
		map[string]any{"count": len(body.Rates)})
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "count": len(body.Rates)})
}

func (h *Handler) AdminFiscalSystems(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"systems": payments.FiscalSystems()})
}
