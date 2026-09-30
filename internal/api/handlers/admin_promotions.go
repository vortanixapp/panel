package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type promotionBody struct {
	Title          *string  `json:"title"`
	Code           *string  `json:"code"`
	DiscountType   *string  `json:"type"`
	Value          *float64 `json:"value"`
	Active         *bool    `json:"active"`
	StartsAt       *string  `json:"starts_at"`
	EndsAt         *string  `json:"ends_at"`
	MaxUses        *int     `json:"max_uses"`
	MaxUsesPerUser *int     `json:"max_uses_per_user"`
	MinAmount      *float64 `json:"min_amount"`
	OnlyNewUsers   *bool    `json:"only_new_users"`
	Description    *string  `json:"description"`

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

func validatePromotion(body promotionBody, discountType, startsAt, endsAt any) string {
	if body.Value != nil {
		if *body.Value < 0 {
			return "Размер скидки не может быть отрицательным"
		}
		if discountType == "percent" && *body.Value > 100 {
			return "Процентная скидка не может быть больше 100"
		}
	}
	if body.BonusPercent != nil && (*body.BonusPercent < 0 || *body.BonusPercent > 999) {
		return "Бонус к пополнению в процентах: от 0 до 999"
	}
	if body.BonusFixed != nil && *body.BonusFixed < 0 {
		return "Фиксированный бонус не может быть отрицательным"
	}
	if body.MinAmount != nil && *body.MinAmount < 0 {
		return "Минимальная сумма не может быть отрицательной"
	}
	if body.MaxUses != nil && *body.MaxUses < 0 {
		return "Лимит использований не может быть отрицательным"
	}
	if body.MaxUsesPerUser != nil && *body.MaxUsesPerUser < 0 {
		return "Лимит использований на пользователя не может быть отрицательным"
	}
	from, hasFrom := startsAt.(time.Time)
	to, hasTo := endsAt.(time.Time)
	if hasFrom && hasTo && !to.After(from) {
		return "Дата окончания должна быть позже даты начала"
	}
	return ""
}

const topupBonusRequired = "Для акции только на пополнение задайте бонус к сумме: скидка при пополнении не действует"

func onlyTopup(applies *[]string) bool {
	if applies == nil || len(*applies) == 0 {
		return false
	}
	for _, item := range *applies {
		if strings.ToLower(strings.TrimSpace(item)) != "topup" {
			return false
		}
	}
	return true
}

func hasTopupBonus(body promotionBody) bool {
	return (body.BonusPercent != nil && *body.BonusPercent > 0) ||
		(body.BonusFixed != nil && *body.BonusFixed > 0)
}

func promotionWriteError(err error) string {
	if strings.Contains(err.Error(), "promotions_") && strings.Contains(err.Error(), "code") {
		return "Промокод с таким кодом уже существует"
	}
	return "Не удалось сохранить акцию"
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

var promoApplyTargets = map[string]bool{"rent": true, "renew": true, "topup": true, "hosting": true}

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
		writeError(w, http.StatusBadRequest, "Тип скидки: percent, fixed или пусто")
		return
	}
	startsAt, ok := parseDate(body.StartsAt)
	if !ok {
		writeError(w, http.StatusBadRequest, "Некорректная дата начала")
		return
	}
	endsAt, ok := parseDate(body.EndsAt)
	if !ok {
		writeError(w, http.StatusBadRequest, "Некорректная дата окончания")
		return
	}
	if msg := validatePromotion(body, discountType, startsAt, endsAt); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if body.AppliesTo != nil && onlyTopup(body.AppliesTo) {
		var bonusPercent, bonusFixed float64
		_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
			SELECT bonus_percent::float8, bonus_fixed::float8 FROM core.promotions WHERE id = $1
		`, id).Scan(&bonusPercent, &bonusFixed)
		if body.BonusPercent != nil {
			bonusPercent = *body.BonusPercent
		}
		if body.BonusFixed != nil {
			bonusFixed = *body.BonusFixed
		}
		if bonusPercent <= 0 && bonusFixed <= 0 {
			writeError(w, http.StatusBadRequest, topupBonusRequired)
			return
		}
	}
	appliesTo, hasApplies := normalizeAppliesTo(body.AppliesTo)
	filtersPatch := promoFiltersPatch(body)

	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.promotions SET
			title          = COALESCE($2, title),
			code           = CASE WHEN $3::bool THEN $4::text ELSE code END,
			discount_type  = CASE WHEN $5::bool THEN $6::text ELSE discount_type END,
			discount_value = COALESCE($7, discount_value),
			active         = COALESCE($8, active),
			starts_at      = CASE WHEN $9::bool THEN $10::timestamptz ELSE starts_at END,
			ends_at        = CASE WHEN $11::bool THEN $12::timestamptz ELSE ends_at END,
			max_uses       = CASE WHEN $13::bool THEN NULLIF($14::int, 0) ELSE max_uses END,
			max_uses_per_user = CASE WHEN $15::bool THEN NULLIF($16::int, 0) ELSE max_uses_per_user END,
			min_amount     = CASE WHEN $17::bool THEN $18::numeric ELSE min_amount END,
			only_new_users = COALESCE($19, only_new_users),
			description    = COALESCE($20, description),
			applies_to     = CASE WHEN $21::bool THEN $22::jsonb ELSE applies_to END,
			bonus_percent  = COALESCE($23, bonus_percent),
			bonus_fixed    = COALESCE($24, bonus_fixed),
			filters        = CASE WHEN $25::jsonb IS NULL THEN filters ELSE filters || $25::jsonb END,
			updated_at     = now()
		WHERE id = $1
	`,
		id,
		body.Title,
		body.Code != nil, normalizePromoCode(body.Code),
		body.DiscountType != nil, discountType,
		body.Value, body.Active,
		body.StartsAt != nil, startsAt,
		body.EndsAt != nil, endsAt,
		body.MaxUses != nil, body.MaxUses,
		body.MaxUsesPerUser != nil, body.MaxUsesPerUser,
		body.MinAmount != nil, body.MinAmount,
		body.OnlyNewUsers, body.Description,
		hasApplies, appliesTo,
		body.BonusPercent, body.BonusFixed,
		filtersPatch,
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, promotionWriteError(err))
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Акция не найдена")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "promotion.update", "promotion:"+id, nil)
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
		`DELETE FROM core.promotions WHERE id = $1`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Акция не найдена")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "promotion.delete", "promotion:"+id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
