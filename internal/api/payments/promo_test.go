package payments

import "testing"

// Бонус при пополнении обязан считаться по своим колонкам bonus_percent и
// bonus_fixed. Раньше на его месте использовались поля СКИДКИ, и акция
// «скидка 20% на аренду» начисляла при пополнении +20% сверху внесённого.
func TestCreditWithBonusIgnoresDiscountFields(t *testing.T) {
	discountOnly := &promotionRow{DiscountType: "percent", DiscountValue: 20}
	if got := creditWithBonus(discountOnly, 1000); got != 1000 {
		t.Errorf("скидочная акция не должна увеличивать зачисление: получено %v вместо 1000", got)
	}

	cases := []struct {
		name   string
		promo  *promotionRow
		amount float64
		want   float64
	}{
		{"процент", &promotionRow{BonusPercent: 10}, 1000, 1100},
		{"фиксированный", &promotionRow{BonusFixed: 250}, 1000, 1250},
		{"оба сразу", &promotionRow{BonusPercent: 10, BonusFixed: 50}, 1000, 1150},
		{"копейки округляются", &promotionRow{BonusPercent: 7.5}, 333.33, 358.33},
		{"без акции", nil, 500, 500},
		{"нулевой бонус", &promotionRow{}, 500, 500},
		{"отрицательный бонус не вычитает", &promotionRow{BonusPercent: -50, BonusFixed: -100}, 500, 500},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := creditWithBonus(c.promo, c.amount); got != c.want {
				t.Errorf("creditWithBonus(%v) = %v, ожидалось %v", c.amount, got, c.want)
			}
		})
	}
}

// Скидка на покупку — отдельный расчёт, его правки бонуса не задели.
func TestCalculateDiscountUnchanged(t *testing.T) {
	if got := calculateDiscount(&promotionRow{DiscountType: "percent", DiscountValue: 20}, 1000); got != 200 {
		t.Errorf("процентная скидка = %v, ожидалось 200", got)
	}
	if got := calculateDiscount(&promotionRow{DiscountType: "fixed", DiscountValue: 150}, 1000); got != 150 {
		t.Errorf("фиксированная скидка = %v, ожидалось 150", got)
	}
	if got := calculateDiscount(&promotionRow{DiscountType: "percent", DiscountValue: 500}, 1000); got != 1000 {
		t.Error("процент скидки должен ограничиваться сотней")
	}
	if got := calculateDiscount(&promotionRow{BonusPercent: 50}, 1000); got != 0 {
		t.Errorf("бонусные поля не должны давать скидку, получено %v", got)
	}
}
