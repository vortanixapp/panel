package handlers

import (
	"strings"
	"testing"
)

func TestWebhooksRejectUnknownPayment(t *testing.T) {
	for _, c := range []struct{ file, fn string }{
		{"webhooks_payments.go", "YooKassaWebhook"},
		{"webhooks_payments.go", "FreekassaWebhook"},
		{"webhooks_payments.go", "RobokassaWebhook"},
		{"webhooks_payments.go", "PayPalWebhook"},
		{"webhooks_payments.go", "NowPaymentsWebhook"},
		{"webhooks_stripe.go", "StripeWebhook"},
	} {
		body := funcBody(t, readSource(t, c.file), c.fn)
		if !strings.Contains(body, "knownPayment") {
			t.Errorf("%s зачисляет по идентификатору, которого нет в базе", c.fn)
		}
	}
}

func TestUnknownPaymentIsCheckedBeforeUse(t *testing.T) {
	body := funcBody(t, readSource(t, "webhooks_amount.go"), "knownPayment")

	if !strings.Contains(body, "uuid.Parse") {
		t.Error("идентификатор не проверяется: запрос упадёт на приведении типа")
	}
	if !strings.Contains(body, "SELECT EXISTS") {
		t.Error("существование платежа не проверяется")
	}
}

func TestAmountCheckSkipsForeignCurrency(t *testing.T) {
	body := funcBody(t, readSource(t, "webhooks_amount.go"), "paymentCoversOrder")

	if !strings.Contains(body, "EqualFold") {
		t.Error("валюты сравниваются без учёта регистра — крипто-платежи будут отвергаться")
	}
	if !strings.Contains(body, "paidAmount+0.009 < orderAmount") {
		t.Error("нет допуска на округление: кассы округляют по-разному")
	}
	if !strings.Contains(body, "paidAmount <= 0") {
		t.Error("касса без суммы в событии перестанет зачислять платежи")
	}
}

func TestWalletDebitChecksCurrency(t *testing.T) {
	body := funcBody(t, readSource(t, "rental.go"), "debitWalletForRentSource")

	if !strings.Contains(body, "currency = $3") {
		t.Error("валюта кошелька не проверяется при списании: рублёвый заказ оплатится гривневым кошельком один к одному")
	}
}

func TestTariffDeletionBlockedByLiveServers(t *testing.T) {
	body := funcBody(t, readSource(t, "admin_tariffs.go"), "DeleteTariff")

	if !strings.Contains(body, "count(*) FROM core.servers") {
		t.Error("тариф удаляется без проверки серверов на нём — продление станет бесплатным")
	}
	if !strings.Contains(body, "StatusConflict") {
		t.Error("отказ должен быть 409 с объяснением, а не тихим успехом")
	}
	if strings.Contains(body, "_, _ = h.dbOf") {
		t.Error("ошибка удаления снова глушится")
	}
}
