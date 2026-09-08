package handlers

import (
	"strings"
	"testing"
)

// Вебхук кассы приходит от чужого сервера, заголовка арендатора в нём нет и
// быть не может. Раньше всё обслуживание шло центральной базой, платёж
// арендатора там не находился, и оплата не зачислялась вовсе.
func TestWebhooksResolveTenantByPayment(t *testing.T) {
	for _, c := range []struct{ file, fn string }{
		{"webhooks_payments.go", "YooKassaWebhook"},
		{"webhooks_payments.go", "FreekassaWebhook"},
		{"webhooks_payments.go", "RobokassaWebhook"},
		{"webhooks_payments.go", "PayPalWebhook"},
		{"webhooks_payments.go", "NowPaymentsWebhook"},
		{"webhooks_stripe.go", "StripeWebhook"},
	} {
		body := funcBody(t, readSource(t, c.file), c.fn)
		if !strings.Contains(body, "withPaymentTenant") {
			t.Errorf("%s не выбирает базу по платежу", c.fn)
		}
		if strings.Contains(body, "completeTopupPayment(r.Context()") {
			t.Errorf("%s зачисляет в базу из запроса, а не в найденную", c.fn)
		}
	}
}

// Поиск идёт по базам арендаторов, поэтому чинятся и платежи, созданные до
// правки: реестра для них нет, а поиск работает.
func TestPaymentPoolSearchesTenantDatabases(t *testing.T) {
	src := readSource(t, "webhooks_tenant.go")

	if !strings.Contains(funcBody(t, src, "poolForPayment"), "uuid.Parse") {
		t.Error("идентификатор не проверяется: запрос упадёт на приведении типа")
	}
	search := funcBody(t, src, "poolWithRow")
	if !strings.Contains(search, "h.db") {
		t.Error("центральная база не проверяется: установка с одним арендатором сломается")
	}
	if !strings.Contains(search, "Slugs(ctx)") {
		t.Error("базы арендаторов не перебираются")
	}
}

// Картинки новостей запрашиваются тегом <img>, без заголовка арендатора: их
// читали из центральной базы, где новостей арендатора не бывает.
func TestNewsImageFindsTenantDatabase(t *testing.T) {
	body := funcBody(t, readSource(t, "admin_news.go"), "ServeNewsImage")

	if strings.Contains(body, "h.db.QueryRow") {
		t.Error("картинка снова читается из центральной базы")
	}
	if !strings.Contains(body, "poolWithRow") {
		t.Error("база арендатора не ищется по самой картинке")
	}
}

// Сверка суммы — второй рубеж после подписи. Она не должна отвергать платёж,
// когда касса назвала сумму в своей валюте: у крипто-касс это монета, а не
// валюта заказа.
func TestAmountCheckSkipsForeignCurrency(t *testing.T) {
	body := funcBody(t, readSource(t, "webhooks_tenant.go"), "paymentCoversOrder")

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

// Кошелёк выбирает клиент, поэтому валюта обязана быть в отборе: иначе рублёвый
// заказ оплачивается гривневым кошельком один к одному.
func TestWalletDebitChecksCurrency(t *testing.T) {
	body := funcBody(t, readSource(t, "rental.go"), "debitWalletForRentSource")

	if !strings.Contains(body, "currency = $3") {
		t.Error("валюта кошелька не проверяется при списании")
	}
}

// Удаление тарифа обнуляло ссылку у серверов, и продление считалось бесплатным.
func TestTariffDeletionBlockedByLiveServers(t *testing.T) {
	body := funcBody(t, readSource(t, "admin_tariffs.go"), "DeleteTariff")

	if !strings.Contains(body, "count(*) FROM core.servers") {
		t.Error("тариф удаляется без проверки серверов на нём")
	}
	if !strings.Contains(body, "StatusConflict") {
		t.Error("отказ должен быть 409 с объяснением, а не тихим успехом")
	}
	if strings.Contains(body, "_, _ = h.dbOf") {
		t.Error("ошибка удаления снова глушится")
	}
}
