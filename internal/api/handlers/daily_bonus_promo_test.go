package handlers

import (
	"strings"
	"testing"

	"github.com/vortanixapp/panel/internal/api/payments"
)

// Четыре промо-типа приза списывали попытку и писали красивую строку в историю,
// но не выдавали ничего: начислялся только приз типа «баланс».
func TestPromoPrizesMapToScopes(t *testing.T) {
	cases := map[string][]string{
		"promo_rent":    {payments.ApplyRent},
		"promo_renew":   {payments.ApplyRenew},
		"promo_game":    {payments.ApplyRent, payments.ApplyRenew},
		"promo_hosting": {payments.ApplyHosting},
	}
	for prizeType, want := range cases {
		got := promoScopesForPrize(prizeType)
		if len(got) != len(want) {
			t.Errorf("%s: областей %d, ожидалось %d", prizeType, len(got), len(want))
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%s: область %q, ожидалась %q", prizeType, got[i], want[i])
			}
		}
	}

	if promoScopesForPrize("balance") != nil {
		t.Error("приз «баланс» промокодом не выдаётся")
	}
	if promoScopesForPrize("что-то новое") != nil {
		t.Error("неизвестный тип приза не должен молча получать область")
	}
}

// Код диктуют по телефону и переписывают с экрана, поэтому похожие на цифры
// буквы в нём не нужны.
func TestBonusPromoCodeShape(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		code := bonusPromoCode()
		if !strings.HasPrefix(code, "BONUS-") {
			t.Fatalf("код без узнаваемого начала: %s", code)
		}
		tail := strings.TrimPrefix(code, "BONUS-")
		if len(tail) != 8 {
			t.Fatalf("длина кода %d, ожидалось 8: %s", len(tail), code)
		}
		if strings.ContainsAny(tail, "IO01") {
			t.Errorf("в коде есть неотличимые от цифр буквы: %s", code)
		}
		seen[code] = true
	}
	if len(seen) < 190 {
		t.Errorf("коды повторяются слишком часто: уникальных %d из 200", len(seen))
	}
}

// Промокод выдаётся лично и на один раз: иначе выигрыш одного клиента
// достался бы всем, кто увидел код.
func TestBonusPromoIsPersonalAndSingleUse(t *testing.T) {
	body := funcBody(t, readSource(t, "account_pages.go"), "grantBonusPrize")

	if !strings.Contains(body, "user_ids") {
		t.Error("промокод не ограничен пользователем, который его выиграл")
	}
	if !strings.Contains(body, "max_uses") {
		t.Error("промокод не ограничен одним применением")
	}
	if !strings.Contains(body, "promotion_id") || !strings.Contains(body, "promotion_code") {
		t.Error("выданный код не записывается в историю прокруток")
	}
}
