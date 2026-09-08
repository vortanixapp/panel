package handlers

import (
	"encoding/json"
	"testing"
)

func promoStr(s string) *string { return &s }

// Код ищется по UPPER(code), а записывался как введён: «promo» и «PROMO»
// ложились двумя записями, из которых находилась случайная. Пустой код должен
// становиться NULL, иначе вторую безкодовую акцию завести нельзя.
func TestNormalizePromoCode(t *testing.T) {
	if got := normalizePromoCode(promoStr("  promo10 ")); got != "PROMO10" {
		t.Errorf("код = %v, ожидалось PROMO10", got)
	}
	if got := normalizePromoCode(promoStr("   ")); got != nil {
		t.Errorf("пустой код должен становиться NULL, получено %v", got)
	}
	if got := normalizePromoCode(nil); got != nil {
		t.Errorf("отсутствие поля не должно менять код, получено %v", got)
	}
}

// Акция «только бонус на пополнение, без скидки» была невыразима: пустой тип
// приводился к percent и превращался в скидку 0%, которую движок считал
// настоящей.
func TestNormalizeDiscountTypeAllowsNone(t *testing.T) {
	for _, in := range []string{"", "none", "null"} {
		got, ok := normalizeDiscountType(promoStr(in))
		if !ok || got != nil {
			t.Errorf("normalizeDiscountType(%q) = (%v, %v), ожидался NULL", in, got, ok)
		}
	}
	if got, ok := normalizeDiscountType(promoStr("PERCENT")); !ok || got != "percent" {
		t.Errorf("percent разобран неверно: (%v, %v)", got, ok)
	}
	if got, ok := normalizeDiscountType(promoStr("fixed")); !ok || got != "fixed" {
		t.Errorf("fixed разобран неверно: (%v, %v)", got, ok)
	}
	if _, ok := normalizeDiscountType(promoStr("подарок")); ok {
		t.Error("неизвестный тип скидки должен отвергаться")
	}
}

func TestNormalizeAppliesTo(t *testing.T) {
	in := []string{"Rent", "topup", "rent", "мусор", " renew "}
	got, ok := normalizeAppliesTo(&in)
	if !ok {
		t.Fatal("список не разобран")
	}
	var parsed []string
	if err := json.Unmarshal([]byte(got.(string)), &parsed); err != nil {
		t.Fatalf("результат не json: %v", err)
	}
	if len(parsed) != 3 || parsed[0] != "rent" || parsed[1] != "topup" || parsed[2] != "renew" {
		t.Errorf("разобрано %v, ожидалось [rent topup renew]", parsed)
	}

	empty := []string{}
	if got, ok := normalizeAppliesTo(&empty); !ok || got.(string) != "[]" {
		t.Errorf("пустой список = %v, ожидался []", got)
	}
	if _, ok := normalizeAppliesTo(nil); ok {
		t.Error("отсутствие поля не должно перезаписывать applies_to")
	}
}

// Ограничения дописываются в filters точечно: акция, у которой меняют только
// список тарифов, не должна терять список игр.
func TestPromoFiltersPatchOnlyTouchesGivenKeys(t *testing.T) {
	if got := promoFiltersPatch(promotionBody{}); got != nil {
		t.Errorf("без ограничений патч должен быть пустым, получено %v", got)
	}

	tariffs := []string{"t1", " t2 ", ""}
	patch := promoFiltersPatch(promotionBody{TariffIDs: &tariffs})
	var parsed map[string][]string
	if err := json.Unmarshal([]byte(patch.(string)), &parsed); err != nil {
		t.Fatalf("патч не json: %v", err)
	}
	if len(parsed) != 1 {
		t.Errorf("в патч попали лишние ключи: %v", parsed)
	}
	if len(parsed["tariff_ids"]) != 2 || parsed["tariff_ids"][1] != "t2" {
		t.Errorf("список тарифов разобран неверно: %v", parsed["tariff_ids"])
	}

	// Явно пустой список — это очистка ограничения, а не отсутствие поля.
	empty := []string{}
	patch = promoFiltersPatch(promotionBody{UserIDs: &empty})
	_ = json.Unmarshal([]byte(patch.(string)), &parsed)
	if ids, ok := parsed["user_ids"]; !ok || len(ids) != 0 {
		t.Errorf("очистка списка пользователей не сохранилась: %v", parsed)
	}
}

// Время в датах: акция «до 18:00 сегодня» должна настраиваться.
func TestParseDateKeepsTime(t *testing.T) {
	for _, in := range []string{
		"2026-08-31T18:00",
		"2026-08-31T18:00:00",
		"2026-08-31 18:00",
		"2026-08-31T18:00:00Z",
	} {
		got, ok := parseDate(promoStr(in))
		if !ok || got == nil {
			t.Errorf("дата %q не разобрана", in)
		}
	}
	if got, ok := parseDate(promoStr("2026-08-31")); !ok || got == nil {
		t.Error("дата без времени должна приниматься по-прежнему")
	}
	if _, ok := parseDate(promoStr("вчера")); ok {
		t.Error("мусор в дате должен отвергаться")
	}
	if got, ok := parseDate(promoStr("")); !ok || got != nil {
		t.Error("пустая дата означает «не задано»")
	}
}
