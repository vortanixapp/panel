package payments

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vortanixapp/panel/pkg/secretbox"
)

const (
	ApplyRent  = "rent"
	ApplyRenew = "renew"
	ApplyTopup = "topup"
	// Веб-хостинг: отдельная область, потому что скидка на игровой сервер к
	// хостинг-тарифу отношения не имеет.
	ApplyHosting = "hosting"
)

type PromoResult struct {
	CreditedAmount float64
	PromoID        string
}

type RentPromoPreview struct {
	Valid     bool    `json:"valid"`
	Error     *string `json:"error"`
	Discount  float64 `json:"discount"`
	BaseCost  float64 `json:"base_cost"`
	FinalCost float64 `json:"final_cost"`
	PromoID   string  `json:"promo_id,omitempty"`
	PromoCode string  `json:"promo_code,omitempty"`
}

type promotionRow struct {
	ID            string
	Code          *string
	Active        bool
	StartsAt      *time.Time
	EndsAt        *time.Time
	AppliesTo     []byte
	DiscountType  string
	DiscountValue float64
	MaxUses       *int
	UsedCount     int
	MinAmount     *float64
	OnlyNewUsers  bool
	Filters       []byte
	BonusPercent  float64
	BonusFixed    float64
}

// ApplyPromo считает сумму к зачислению при пополнении баланса.
//
// Раньше здесь стоял отдельный, куда более простой отбор: запрос смотрел
// только на active и трактовал discount_type/discount_value как БОНУС. То
// есть акция «скидка 20% на аренду» — а форма админки создаёт именно такие,
// пустой тип скидки она приводит к percent — начисляла при пополнении +20%
// сверху внесённой суммы. Мимо проходили и сроки, и лимит применений, и
// applies_to, и min_amount, и only_new_users.
//
// Теперь отбор общий, тот же, что у аренды и продления, с applies_to =
// topup, а бонус берётся из своих колонок bonus_percent и bonus_fixed. Поля
// скидки к пополнению не относятся вовсе: скидка уменьшает цену покупки, а
// не увеличивает зачисление.
func ApplyPromo(ctx context.Context, db *pgxpool.Pool, tenantID, userID, code string, amount float64) (PromoResult, error) {
	out := PromoResult{CreditedAmount: roundMoney(amount)}
	code = strings.TrimSpace(strings.ToUpper(code))
	if code == "" {
		return out, nil
	}
	promo, reason := PickPromotion(ctx, db, tenantID, ApplyTopup, userID, code, "", "", "", amount)
	if promo == nil {
		if reason == "" {
			reason = "Промокод не найден"
		}
		return out, fmt.Errorf("%s", reason)
	}
	out.PromoID = promo.ID
	out.CreditedAmount = creditWithBonus(promo, amount)
	return out, nil
}

// creditWithBonus — сколько зачислить на баланс с учётом бонуса акции.
func creditWithBonus(promo *promotionRow, amount float64) float64 {
	if promo == nil || amount <= 0 {
		return roundMoney(amount)
	}
	bonus := 0.0
	if promo.BonusPercent > 0 {
		bonus += amount * promo.BonusPercent / 100
	}
	if promo.BonusFixed > 0 {
		bonus += promo.BonusFixed
	}
	return roundMoney(amount + bonus)
}

func roundMoney(v float64) float64 {
	return math.Max(0, math.Round(v*100)/100)
}

func PickPromotion(
	ctx context.Context,
	db *pgxpool.Pool,
	tenantID, applyTo, userID, code, tariffID, gameID, locationID string,
	amount float64,
) (*promotionRow, string) {
	code = strings.TrimSpace(strings.ToUpper(code))
	if code != "" {
		row, err := loadPromotionByCode(ctx, db, tenantID, code)
		if err != nil {
			return nil, "Промокод не найден"
		}
		if !isPromotionEligible(ctx, db, &row, applyTo, userID, tariffID, gameID, locationID, amount) {
			return nil, "Промокод недоступен"
		}
		return &row, ""
	}
	rows, err := db.Query(ctx, `
		SELECT id::text, code, active, starts_at, ends_at, applies_to,
		       COALESCE(discount_type, 'percent'), discount_value::float8,
		       max_uses, used_count, min_amount::float8, only_new_users, filters,
		       bonus_percent::float8, bonus_fixed::float8
		FROM core.promotions
		WHERE tenant_id = $1 AND active = true AND (code IS NULL OR code = '')
		ORDER BY created_at DESC LIMIT 50
	`, tenantID)
	if err != nil {
		return nil, ""
	}
	defer rows.Close()
	var best *promotionRow
	bestBenefit := 0.0
	for rows.Next() {
		row, scanErr := scanPromotionRow(rows)
		if scanErr != nil {
			continue
		}
		if !isPromotionEligible(ctx, db, &row, applyTo, userID, tariffID, gameID, locationID, amount) {
			continue
		}
		benefit := calculateDiscount(&row, amount)
		if benefit > bestBenefit {
			bestBenefit = benefit
			copy := row
			best = &copy
		}
	}
	return best, ""
}

func ApplyRentDiscount(promo *promotionRow, baseCost float64) RentPromoPreview {
	baseCost = math.Max(0, math.Round(baseCost*100)/100)
	out := RentPromoPreview{Valid: false, BaseCost: baseCost, FinalCost: baseCost}
	if promo == nil {
		return out
	}
	discount := calculateDiscount(promo, baseCost)
	final := math.Max(0, math.Round((baseCost-discount)*100)/100)
	out.Valid = true
	out.Discount = discount
	out.FinalCost = final
	out.PromoID = promo.ID
	if promo.Code != nil {
		out.PromoCode = *promo.Code
	}
	return out
}

func IncrementPromoUsage(ctx context.Context, db *pgxpool.Pool, promoID string) error {
	if promoID == "" {
		return nil
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var maxUses *int
	var used int
	var active bool
	err = tx.QueryRow(ctx, `
		SELECT max_uses, used_count, active FROM core.promotions WHERE id = $1 FOR UPDATE
	`, promoID).Scan(&maxUses, &used, &active)
	if err != nil {
		return err
	}
	if !active {
		return nil
	}
	if maxUses != nil && used >= *maxUses {
		return nil
	}
	_, err = tx.Exec(ctx, `UPDATE core.promotions SET used_count = used_count + 1, updated_at = now() WHERE id = $1`, promoID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func loadPromotionByCode(ctx context.Context, db *pgxpool.Pool, tenantID, code string) (promotionRow, error) {
	row := promotionRow{}
	err := db.QueryRow(ctx, `
		SELECT id::text, code, active, starts_at, ends_at, applies_to,
		       COALESCE(discount_type, 'percent'), discount_value::float8,
		       max_uses, used_count, min_amount::float8, only_new_users, filters,
		       bonus_percent::float8, bonus_fixed::float8
		FROM core.promotions
		WHERE tenant_id = $1 AND UPPER(code) = $2
		LIMIT 1
	`, tenantID, code).Scan(
		&row.ID, &row.Code, &row.Active, &row.StartsAt, &row.EndsAt, &row.AppliesTo,
		&row.DiscountType, &row.DiscountValue, &row.MaxUses, &row.UsedCount,
		&row.MinAmount, &row.OnlyNewUsers, &row.Filters,
		&row.BonusPercent, &row.BonusFixed,
	)
	return row, err
}

func scanPromotionRow(rows interface {
	Scan(dest ...any) error
}) (promotionRow, error) {
	var row promotionRow
	err := rows.Scan(
		&row.ID, &row.Code, &row.Active, &row.StartsAt, &row.EndsAt, &row.AppliesTo,
		&row.DiscountType, &row.DiscountValue, &row.MaxUses, &row.UsedCount,
		&row.MinAmount, &row.OnlyNewUsers, &row.Filters,
		&row.BonusPercent, &row.BonusFixed,
	)
	return row, err
}

func calculateDiscount(promo *promotionRow, amount float64) float64 {
	if promo == nil {
		return 0
	}
	dt := strings.ToLower(promo.DiscountType)
	val := promo.DiscountValue
	var discount float64
	switch dt {
	case "fixed", "amount":
		discount = math.Max(0, val)
	case "percent", "percentage", "":
		pct := math.Max(0, math.Min(100, val))
		discount = amount * (pct / 100)
	default:
		discount = amount * (val / 100)
	}
	return math.Round(discount*100) / 100
}

func isPromotionEligible(
	ctx context.Context,
	db *pgxpool.Pool,
	promo *promotionRow,
	applyTo, userID, tariffID, gameID, locationID string,
	amount float64,
) bool {
	if promo == nil || !promo.Active {
		return false
	}
	now := time.Now()
	if promo.StartsAt != nil && promo.StartsAt.After(now) {
		return false
	}
	if promo.EndsAt != nil && promo.EndsAt.Before(now) {
		return false
	}
	if promo.MaxUses != nil && promo.UsedCount >= *promo.MaxUses {
		return false
	}
	if promo.MinAmount != nil && amount < *promo.MinAmount {
		return false
	}
	if len(promo.AppliesTo) > 0 {
		var applies []string
		if json.Unmarshal(promo.AppliesTo, &applies) == nil && len(applies) > 0 {
			found := false
			for _, a := range applies {
				if strings.EqualFold(a, applyTo) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	if promo.OnlyNewUsers && userID != "" {
		var serverCount int
		_ = db.QueryRow(ctx, `SELECT COUNT(*) FROM core.servers WHERE user_id = $1`, userID).Scan(&serverCount)
		if serverCount > 0 {
			return false
		}
	}
	if len(promo.Filters) > 0 {
		var filters map[string]any
		if json.Unmarshal(promo.Filters, &filters) == nil {
			if ids, ok := filters["tariff_ids"].([]any); ok && len(ids) > 0 && tariffID != "" {
				if !idInList(tariffID, ids) {
					return false
				}
			}
			if ids, ok := filters["game_ids"].([]any); ok && len(ids) > 0 && gameID != "" {
				if !idInList(gameID, ids) {
					return false
				}
			}
			if ids, ok := filters["location_ids"].([]any); ok && len(ids) > 0 && locationID != "" {
				if !idInList(locationID, ids) {
					return false
				}
			}
			// Персональные промокоды: список пользователей, которым акция
			// доступна. В отличие от трёх списков выше, пустой userID здесь не
			// пропускает проверку — иначе именной код срабатывал бы у любого,
			// кого не удалось определить.
			if ids, ok := filters["user_ids"].([]any); ok && len(ids) > 0 {
				if userID == "" || !idInList(userID, ids) {
					return false
				}
			}
		}
	}
	return true
}

func idInList(id string, list []any) bool {
	for _, item := range list {
		if fmt.Sprint(item) == id {
			return true
		}
	}
	return false
}

func LoadProviderConfig(ctx context.Context, db *pgxpool.Pool, box *secretbox.Box, tenantID, provider string) (map[string]any, error) {
	var raw []byte
	err := db.QueryRow(ctx, `
		SELECT config FROM core.payment_providers
		WHERE tenant_id = $1 AND provider = $2 AND enabled = true
		LIMIT 1
	`, tenantID, provider).Scan(&raw)
	if err != nil {
		return nil, err
	}
	plain, err := box.DecryptJSON(raw)
	if err != nil {
		return nil, err
	}
	cfg := map[string]any{}
	if len(plain) > 0 {
		_ = json.Unmarshal(plain, &cfg)
	}
	return cfg, nil
}
