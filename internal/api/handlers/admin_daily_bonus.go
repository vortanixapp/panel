package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

var bonusColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

var bonusPrizeTypes = map[string]bool{
	"balance": true, "promo_rent": true, "promo_renew": true, "promo_hosting": true, "promo_game": true,
}

type adminBonusPrizeBody struct {
	Label         *string  `json:"label"`
	Type          *string  `json:"type"`
	Value         *float64 `json:"value"`
	DiscountType  *string  `json:"discount_type"`
	Weight        *int     `json:"weight"`
	Color         *string  `json:"color"`
	Icon          *string  `json:"icon"`
	DurationHours *int     `json:"duration_hours"`
	Active        *bool    `json:"active"`
	SortOrder     *int     `json:"sort_order"`
}

type adminBonusPrize struct {
	ID            string  `json:"id"`
	Label         string  `json:"label"`
	Type          string  `json:"type"`
	Value         float64 `json:"value"`
	DiscountType  string  `json:"discount_type"`
	Weight        int     `json:"weight"`
	Color         string  `json:"color"`
	Icon          string  `json:"icon"`
	DurationHours int     `json:"duration_hours"`
	Active        bool    `json:"active"`
	SortOrder     int     `json:"sort_order"`
	Chance        float64 `json:"chance"`
	Wins          int     `json:"wins"`
}

func (h *Handler) loadAdminBonusPrizes(r *http.Request) ([]adminBonusPrize, error) {
	rows, err := h.readerOf(r.Context()).Query(r.Context(), `
		SELECT p.id::text, p.label, p.type, p.value::float8, COALESCE(p.discount_type, ''), p.weight,
		       COALESCE(p.color, ''), COALESCE(p.icon, ''), COALESCE(p.prize_duration_hours, 0),
		       p.active, p.sort_order,
		       (SELECT count(*) FROM core.daily_bonus_spins s WHERE s.prize_id = p.id)::int
		FROM core.daily_bonus_prizes p
		ORDER BY p.sort_order, p.created_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []adminBonusPrize{}
	total := 0
	for rows.Next() {
		var p adminBonusPrize
		if err := rows.Scan(&p.ID, &p.Label, &p.Type, &p.Value, &p.DiscountType, &p.Weight,
			&p.Color, &p.Icon, &p.DurationHours, &p.Active, &p.SortOrder, &p.Wins); err != nil {
			return nil, err
		}
		if p.Active && p.Weight > 0 {
			total += p.Weight
		}
		out = append(out, p)
	}
	for i := range out {
		if out[i].Active && out[i].Weight > 0 && total > 0 {
			out[i].Chance = round2(float64(out[i].Weight) / float64(total) * 100)
		}
	}
	return out, rows.Err()
}

func (h *Handler) AdminDailyBonus(w http.ResponseWriter, r *http.Request) {
	prizes, err := h.loadAdminBonusPrizes(r)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	var spinsDay, spinsWeek, players int
	var balanceWeek float64
	_ = h.readerOf(r.Context()).QueryRow(r.Context(), `
		SELECT count(*) FILTER (WHERE created_at > now() - INTERVAL '24 hours')::int,
		       count(*)::int,
		       count(DISTINCT user_id)::int,
		       COALESCE(sum(reward_value) FILTER (WHERE reward_type = 'balance'), 0)::float8
		FROM core.daily_bonus_spins
		WHERE created_at > now() - INTERVAL '7 days'
	`).Scan(&spinsDay, &spinsWeek, &players, &balanceWeek)
	writeJSON(w, http.StatusOK, map[string]any{
		"prizes": prizes,
		"stats": map[string]any{
			"spins_24h": spinsDay, "spins_7d": spinsWeek, "players_7d": players, "balance_7d": round2(balanceWeek),
		},
		"cooldown_hours": int(bonusCooldown / time.Hour),
	})
}

func (p *adminBonusPrize) apply(b adminBonusPrizeBody) {
	if b.Label != nil {
		p.Label = strings.TrimSpace(*b.Label)
	}
	if b.Type != nil {
		p.Type = strings.TrimSpace(*b.Type)
	}
	if b.Value != nil {
		p.Value = *b.Value
	}
	if b.DiscountType != nil {
		p.DiscountType = strings.TrimSpace(*b.DiscountType)
	}
	if b.Weight != nil {
		p.Weight = *b.Weight
	}
	if b.Color != nil {
		p.Color = strings.TrimSpace(*b.Color)
	}
	if b.Icon != nil {
		p.Icon = strings.TrimSpace(*b.Icon)
	}
	if b.DurationHours != nil {
		p.DurationHours = *b.DurationHours
	}
	if b.Active != nil {
		p.Active = *b.Active
	}
	if b.SortOrder != nil {
		p.SortOrder = *b.SortOrder
	}
}

func (p *adminBonusPrize) validate() string {
	if p.Label == "" || len([]rune(p.Label)) > 60 {
		return "Название приза обязательно, до 60 символов"
	}
	if !bonusPrizeTypes[p.Type] {
		return "Неизвестный тип приза"
	}
	if p.Value <= 0 {
		return "Значение приза должно быть больше нуля"
	}
	if p.Weight < 0 || p.Weight > 100000 {
		return "Вес должен быть от 0 до 100000"
	}
	if p.Color != "" && !bonusColorRe.MatchString(p.Color) {
		return "Цвет задаётся в формате #RRGGBB"
	}
	if len(p.Icon) > 64 {
		return "Слишком длинное имя иконки"
	}
	if p.DurationHours < 0 || p.DurationHours > 24*365 {
		return "Срок действия промокода — от 0 до 8760 часов"
	}
	if p.Type == "balance" {
		p.DiscountType = ""
		p.DurationHours = 0
		if p.Value > 1000000 {
			return "Слишком большая сумма"
		}
		return ""
	}
	if p.DiscountType != "fixed" {
		p.DiscountType = "percent"
	}
	if p.DiscountType == "percent" && p.Value > 100 {
		return "Скидка в процентах не может быть больше 100"
	}
	return ""
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullIfZero(n int) any {
	if n == 0 {
		return nil
	}
	return n
}

func (h *Handler) AdminDailyBonusCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	var body adminBonusPrizeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	p := adminBonusPrize{Type: "balance", Weight: 10, Active: true}
	_ = h.readerOf(r.Context()).QueryRow(r.Context(),
		`SELECT COALESCE(max(sort_order), 0) + 1 FROM core.daily_bonus_prizes`).Scan(&p.SortOrder)
	p.apply(body)
	if msg := p.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.daily_bonus_prizes
			(label, type, value, discount_type, weight, color, icon, prize_duration_hours, active, sort_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id::text
	`, p.Label, p.Type, p.Value, nullIfEmpty(p.DiscountType), p.Weight, nullIfEmpty(p.Color),
		nullIfEmpty(p.Icon), nullIfZero(p.DurationHours), p.Active, p.SortOrder).Scan(&p.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "daily_bonus.prize_create", "daily_bonus_prize:"+p.ID,
		map[string]any{"label": p.Label, "type": p.Type, "value": p.Value, "weight": p.Weight})
	writeJSON(w, http.StatusCreated, map[string]string{"id": p.ID})
}

func (h *Handler) AdminDailyBonusUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	var body adminBonusPrizeBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	var p adminBonusPrize
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT id::text, label, type, value::float8, COALESCE(discount_type, ''), weight,
		       COALESCE(color, ''), COALESCE(icon, ''), COALESCE(prize_duration_hours, 0), active, sort_order
		FROM core.daily_bonus_prizes WHERE id::text = $1
	`, id).Scan(&p.ID, &p.Label, &p.Type, &p.Value, &p.DiscountType, &p.Weight,
		&p.Color, &p.Icon, &p.DurationHours, &p.Active, &p.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "prize not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	p.apply(body)
	if msg := p.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	_, err = h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.daily_bonus_prizes SET
			label = $2, type = $3, value = $4, discount_type = $5, weight = $6, color = $7, icon = $8,
			prize_duration_hours = $9, active = $10, sort_order = $11, updated_at = now()
		WHERE id = $1::uuid
	`, p.ID, p.Label, p.Type, p.Value, nullIfEmpty(p.DiscountType), p.Weight, nullIfEmpty(p.Color),
		nullIfEmpty(p.Icon), nullIfZero(p.DurationHours), p.Active, p.SortOrder)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "daily_bonus.prize_update", "daily_bonus_prize:"+p.ID,
		map[string]any{"label": p.Label, "type": p.Type, "value": p.Value, "weight": p.Weight, "active": p.Active})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) AdminDailyBonusDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	var label string
	err := h.dbOf(r.Context()).QueryRow(r.Context(),
		`DELETE FROM core.daily_bonus_prizes WHERE id::text = $1 RETURNING label`, id).Scan(&label)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "prize not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "daily_bonus.prize_delete", "daily_bonus_prize:"+id,
		map[string]any{"label": label})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
