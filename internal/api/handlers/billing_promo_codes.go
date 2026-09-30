package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

type userPromoCode struct {
	ID            string     `json:"id"`
	Code          string     `json:"code"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	DiscountType  string     `json:"discount_type"`
	DiscountValue float64    `json:"discount_value"`
	BonusPercent  float64    `json:"bonus_percent"`
	BonusFixed    float64    `json:"bonus_fixed"`
	MinAmount     *float64   `json:"min_amount"`
	AppliesTo     []string   `json:"applies_to"`
	StartsAt      *time.Time `json:"starts_at"`
	EndsAt        *time.Time `json:"ends_at"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
}

func promoCodeStatus(active bool, startsAt, endsAt *time.Time, maxUses *int, used int, now time.Time) string {
	switch {
	case maxUses != nil && used >= *maxUses:
		return "used"
	case endsAt != nil && endsAt.Before(now):
		return "expired"
	case !active:
		return "disabled"
	case startsAt != nil && startsAt.After(now):
		return "upcoming"
	}
	return "active"
}

func (h *Handler) BillingPromoCodes(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, COALESCE(code, ''), title, COALESCE(description, ''),
		       COALESCE(discount_type, ''), discount_value::float8,
		       bonus_percent::float8, bonus_fixed::float8, min_amount::float8,
		       applies_to, starts_at, ends_at, active, max_uses, used_count, created_at
		FROM core.promotions
		WHERE COALESCE(code, '') <> ''
		  AND filters->'user_ids' @> to_jsonb($1::text)
		ORDER BY created_at DESC
		LIMIT 100
	`, claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load promo codes")
		return
	}
	defer rows.Close()

	now := time.Now()
	list := []userPromoCode{}
	for rows.Next() {
		var p userPromoCode
		var appliesRaw []byte
		var active bool
		var maxUses *int
		var used int
		if rows.Scan(&p.ID, &p.Code, &p.Title, &p.Description, &p.DiscountType, &p.DiscountValue,
			&p.BonusPercent, &p.BonusFixed, &p.MinAmount, &appliesRaw, &p.StartsAt, &p.EndsAt,
			&active, &maxUses, &used, &p.CreatedAt) != nil {
			continue
		}
		p.AppliesTo = []string{}
		_ = json.Unmarshal(appliesRaw, &p.AppliesTo)
		p.Status = promoCodeStatus(active, p.StartsAt, p.EndsAt, maxUses, used, now)
		list = append(list, p)
	}
	writeJSON(w, http.StatusOK, map[string]any{"promo_codes": list})
}
