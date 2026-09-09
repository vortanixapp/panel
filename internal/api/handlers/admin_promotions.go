package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type promotionBody struct {
	Title        *string  `json:"title"`
	Code         *string  `json:"code"`
	DiscountType *string  `json:"type"`
	Value        *float64 `json:"value"`
	Active       *bool    `json:"active"`
	StartsAt     *string  `json:"starts_at"`
	EndsAt       *string  `json:"ends_at"`
	MaxUses      *int     `json:"max_uses"`
	MinAmount    *float64 `json:"min_amount"`
	OnlyNewUsers *bool    `json:"only_new_users"`
	Description  *string  `json:"description"`

	AppliesTo *[]string `json:"applies_to"`

	BonusPercent *float64 `json:"bonus_percent"`
	BonusFixed   *float64 `json:"bonus_fixed"`

	TariffIDs   *[]string `json:"tariff_ids"`
	GameIDs     *[]string `json:"game_ids"`
	LocationIDs *[]string `json:"location_ids"`
	UserIDs     *[]string `json:"user_ids"`
}

func parseDate(v *string) (any, bool) {
	if v == nil {
		return nil, true
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil, true
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return nil, false
}

func isoOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

func normalizeDiscountType(v *string) (any, bool) {
	if v == nil {
		return nil, true
	}
	switch strings.TrimSpace(strings.ToLower(*v)) {
	case "", "none", "null":
		return nil, true
	case "percent":
		return "percent", true
	case "fixed":
		return "fixed", true
	default:
		return nil, false
	}
}

func normalizePromoCode(v *string) any {
	if v == nil {
		return nil
	}
	code := strings.ToUpper(strings.TrimSpace(*v))
	if code == "" {
		return nil
	}
	return code
}

var promoApplyTargets = map[string]bool{"rent": true, "renew": true, "topup": true}

func normalizeAppliesTo(v *[]string) (any, bool) {
	if v == nil {
		return nil, false
	}
	seen := map[string]bool{}
	out := []string{}
	for _, item := range *v {
		key := strings.ToLower(strings.TrimSpace(item))
		if !promoApplyTargets[key] || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	encoded, err := json.Marshal(out)
	if err != nil {
		return nil, false
	}
	return string(encoded), true
}

func promoFiltersPatch(body promotionBody) any {
	patch := map[string][]string{}
	add := func(key string, v *[]string) {
		if v == nil {
			return
		}
		out := []string{}
		for _, item := range *v {
			if s := strings.TrimSpace(item); s != "" {
				out = append(out, s)
			}
		}
		patch[key] = out
	}
	add("tariff_ids", body.TariffIDs)
	add("game_ids", body.GameIDs)
	add("location_ids", body.LocationIDs)
	add("user_ids", body.UserIDs)
	if len(patch) == 0 {
		return nil
	}
	encoded, err := json.Marshal(patch)
	if err != nil {
		return nil
	}
	return string(encoded)
}

func (h *Handler) UpdatePromotion(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	var body promotionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	discountType, ok := normalizeDiscountType(body.DiscountType)
	if !ok {
		writeError(w, http.StatusBadRequest, "тип скидки: percent, fixed или пусто")
		return
	}
	startsAt, ok := parseDate(body.StartsAt)
	if !ok {
		writeError(w, http.StatusBadRequest, "некорректная дата начала")
		return
	}
	endsAt, ok := parseDate(body.EndsAt)
	if !ok {
		writeError(w, http.StatusBadRequest, "некорректная дата окончания")
		return
	}
	appliesTo, hasApplies := normalizeAppliesTo(body.AppliesTo)
	filtersPatch := promoFiltersPatch(body)

	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.promotions SET
			title          = COALESCE($3, title),
			code           = CASE WHEN $4::bool THEN $5::text ELSE code END,
			discount_type  = CASE WHEN $6::bool THEN $7::text ELSE discount_type END,
			discount_value = COALESCE($8, discount_value),
			active         = COALESCE($9, active),
			starts_at      = CASE WHEN $10::bool THEN $11::timestamptz ELSE starts_at END,
			ends_at        = CASE WHEN $12::bool THEN $13::timestamptz ELSE ends_at END,
			max_uses       = CASE WHEN $14::bool THEN $15::int ELSE max_uses END,
			min_amount     = CASE WHEN $16::bool THEN $17::numeric ELSE min_amount END,
			only_new_users = COALESCE($18, only_new_users),
			description    = COALESCE($19, description),
			applies_to     = CASE WHEN $20::bool THEN $21::jsonb ELSE applies_to END,
			bonus_percent  = COALESCE($22, bonus_percent),
			bonus_fixed    = COALESCE($23, bonus_fixed),
			filters        = CASE WHEN $24::jsonb IS NULL THEN filters ELSE filters || $24::jsonb END,
			updated_at     = now()
		WHERE id = $1 AND tenant_id = $2
	`,
		id, claims.TenantID,
		body.Title,
		body.Code != nil, normalizePromoCode(body.Code),
		body.DiscountType != nil, discountType,
		body.Value, body.Active,
		body.StartsAt != nil, startsAt,
		body.EndsAt != nil, endsAt,
		body.MaxUses != nil, body.MaxUses,
		body.MinAmount != nil, body.MinAmount,
		body.OnlyNewUsers, body.Description,
		hasApplies, appliesTo,
		body.BonusPercent, body.BonusFixed,
		filtersPatch,
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "акция не найдена")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "promotion.update", "promotion:"+id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *Handler) DeletePromotion(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	tag, err := h.dbOf(r.Context()).Exec(r.Context(),
		`DELETE FROM core.promotions WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "акция не найдена")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "promotion.delete", "promotion:"+id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
