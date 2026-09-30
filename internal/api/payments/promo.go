package payments

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ApplyRent    = "rent"
	ApplyRenew   = "renew"
	ApplyTopup   = "topup"
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
	ID             string
	Code           *string
	Active         bool
	StartsAt       *time.Time
	EndsAt         *time.Time
	AppliesTo      []byte
	DiscountType   string
	DiscountValue  float64
	MaxUses        *int
	UsedCount      int
	MinAmount      *float64
	OnlyNewUsers   bool
	Filters        []byte
	BonusPercent   float64
	BonusFixed     float64
	MaxUsesPerUser *int
}

func ApplyPromo(ctx context.Context, db *pgxpool.Pool, userID, code string, amount float64) (PromoResult, error) {
	out := PromoResult{CreditedAmount: roundMoney(amount)}
	code = strings.TrimSpace(strings.ToUpper(code))
	if code == "" {
		return out, nil
	}
	promo, reason := PickPromotion(ctx, db, ApplyTopup, userID, code, "", "", "", amount)
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
	applyTo, userID, code, tariffID, gameID, locationID string,
	amount float64,
) (*promotionRow, string) {
	code = strings.TrimSpace(strings.ToUpper(code))
	if code != "" {
		row, err := loadPromotionByCode(ctx, db, code)
		if err != nil {
			return nil, "Промокод не найден"
		}
		if promoUsedUp(ctx, db, &row, userID) {
			return nil, "Вы уже использовали этот промокод"
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
		       bonus_percent::float8, bonus_fixed::float8, max_uses_per_user
		FROM core.promotions
		WHERE active = true AND (code IS NULL OR code = '')
		ORDER BY created_at DESC LIMIT 50
	`)
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
		if promoUsedUp(ctx, db, &row, userID) ||
			!isPromotionEligible(ctx, db, &row, applyTo, userID, tariffID, gameID, locationID, amount) {
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

func promoUsedUp(ctx context.Context, db *pgxpool.Pool, promo *promotionRow, userID string) bool {
	if promo.MaxUsesPerUser == nil || userID == "" {
		return false
	}
	var used int
	err := db.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM core.promotion_uses WHERE promotion_id = $1::uuid AND user_id = $2::uuid
	`, promo.ID, userID).Scan(&used)
	return err == nil && used >= *promo.MaxUsesPerUser
}

func ClaimPromoUsage(ctx context.Context, db *pgxpool.Pool, promoID, userID string) (bool, error) {
	if promoID == "" {
		return true, nil
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	claimed, err := ClaimPromoUsageTx(ctx, tx, promoID, userID, true)
	if err != nil || !claimed {
		return false, err
	}
	return true, tx.Commit(ctx)
}

func ClaimPromoUsageTx(ctx context.Context, tx pgx.Tx, promoID, userID string, strict bool) (bool, error) {
	var active bool
	var maxUses, perUser *int
	var used int
	var endsAt *time.Time
	err := tx.QueryRow(ctx, `
		SELECT active, max_uses, max_uses_per_user, used_count, ends_at
		FROM core.promotions WHERE id = $1::uuid FOR UPDATE
	`, promoID).Scan(&active, &maxUses, &perUser, &used, &endsAt)
	if err != nil {
		return false, err
	}
	if strict && (!active || (endsAt != nil && endsAt.Before(time.Now()))) {
		return false, nil
	}
	if maxUses != nil && used >= *maxUses {
		return false, nil
	}
	if userID != "" {
		if perUser != nil {
			var byUser int
			if err := tx.QueryRow(ctx, `
				SELECT COUNT(*)::int FROM core.promotion_uses WHERE promotion_id = $1::uuid AND user_id = $2::uuid
			`, promoID, userID).Scan(&byUser); err != nil {
				return false, err
			}
			if byUser >= *perUser {
				return false, nil
			}
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO core.promotion_uses (promotion_id, user_id) VALUES ($1::uuid, $2::uuid)
		`, promoID, userID); err != nil {
			return false, err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE core.promotions SET used_count = used_count + 1, updated_at = now() WHERE id = $1::uuid
	`, promoID); err != nil {
		return false, err
	}
	return true, nil
}

func ReleasePromoUsage(ctx context.Context, db *pgxpool.Pool, promoID, userID string) error {
	if promoID == "" {
		return nil
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if userID != "" {
		if _, err := tx.Exec(ctx, `
			DELETE FROM core.promotion_uses
			WHERE id = (
				SELECT id FROM core.promotion_uses
				WHERE promotion_id = $1::uuid AND user_id = $2::uuid
				ORDER BY created_at DESC LIMIT 1
			)
		`, promoID, userID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE core.promotions SET used_count = GREATEST(used_count - 1, 0), updated_at = now() WHERE id = $1::uuid
	`, promoID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func RemainingPromoUses(ctx context.Context, db *pgxpool.Pool, promoID, userID string) int {
	var maxUses, perUser *int
	var used int
	err := db.QueryRow(ctx, `
		SELECT max_uses, max_uses_per_user, used_count FROM core.promotions WHERE id = $1::uuid
	`, promoID).Scan(&maxUses, &perUser, &used)
	if err != nil {
		return -1
	}
	remaining := -1
	if maxUses != nil {
		remaining = max(*maxUses-used, 0)
	}
	if perUser != nil && userID != "" {
		var byUser int
		_ = db.QueryRow(ctx, `
			SELECT COUNT(*)::int FROM core.promotion_uses WHERE promotion_id = $1::uuid AND user_id = $2::uuid
		`, promoID, userID).Scan(&byUser)
		left := max(*perUser-byUser, 0)
		if remaining < 0 || left < remaining {
			remaining = left
		}
	}
	return remaining
}

func loadPromotionByCode(ctx context.Context, db *pgxpool.Pool, code string) (promotionRow, error) {
	row := promotionRow{}
	err := db.QueryRow(ctx, `
		SELECT id::text, code, active, starts_at, ends_at, applies_to,
		       COALESCE(discount_type, 'percent'), discount_value::float8,
		       max_uses, used_count, min_amount::float8, only_new_users, filters,
		       bonus_percent::float8, bonus_fixed::float8, max_uses_per_user
		FROM core.promotions
		WHERE UPPER(code) = $1
		LIMIT 1
	`, code).Scan(
		&row.ID, &row.Code, &row.Active, &row.StartsAt, &row.EndsAt, &row.AppliesTo,
		&row.DiscountType, &row.DiscountValue, &row.MaxUses, &row.UsedCount,
		&row.MinAmount, &row.OnlyNewUsers, &row.Filters,
		&row.BonusPercent, &row.BonusFixed, &row.MaxUsesPerUser,
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
		&row.BonusPercent, &row.BonusFixed, &row.MaxUsesPerUser,
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
				if !anyIDInList(gameAliases(ctx, db, gameID), ids) {
					return false
				}
			}
			if ids, ok := filters["location_ids"].([]any); ok && len(ids) > 0 && locationID != "" {
				if !idInList(locationID, ids) {
					return false
				}
			}
			if ids, ok := filters["user_ids"].([]any); ok && len(ids) > 0 {
				if userID == "" || !idInList(userID, ids) {
					return false
				}
			}
		}
	}
	return true
}

func gameAliases(ctx context.Context, db *pgxpool.Pool, gameID string) []string {
	aliases := []string{gameID}
	var id, slug string
	err := db.QueryRow(ctx, `
		SELECT id::text, slug FROM core.games WHERE id::text = $1 OR slug = $1 LIMIT 1
	`, gameID).Scan(&id, &slug)
	if err == nil {
		aliases = append(aliases, id, slug)
	}
	return aliases
}

func anyIDInList(ids []string, list []any) bool {
	for _, id := range ids {
		if idInList(id, list) {
			return true
		}
	}
	return false
}

func idInList(id string, list []any) bool {
	for _, item := range list {
		if fmt.Sprint(item) == id {
			return true
		}
	}
	return false
}
