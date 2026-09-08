package handlers

import (
	"context"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Вебхуки касс приходят от чужого сервера: заголовка X-Tenant-Slug в них нет и
// быть не может, поэтому withTenantPool пул не ставит, и всё обслуживание шло
// центральной базой. Платёж арендатора лежит в его собственной базе — там он не
// находился, зачисления не происходило, а касса получала «оплачено».
//
// Базу выбираем по самому платежу: сначала центральная (установка с одним
// арендатором), затем базы арендаторов по очереди. Так же чинятся и платежи,
// созданные до этой правки: реестра для них нет, а поиск работает.
func (h *Handler) poolForPayment(ctx context.Context, paymentID string) *pgxpool.Pool {
	id := strings.TrimSpace(paymentID)
	if id == "" {
		return nil
	}
	// Идентификаторы платежей — UUID. Кассы присылают их обратно как есть, но
	// проверяем: с мусором запрос упадёт на приведении типа, а не вернёт «нет».
	if _, err := uuid.Parse(id); err != nil {
		return nil
	}

	return h.poolWithRow(ctx, `SELECT EXISTS(SELECT 1 FROM core.payments WHERE id = $1::uuid)`, id)
}

// poolWithRow ищет базу, в которой есть строка по заданному условию: сначала
// центральную (установка с одним арендатором), затем базы арендаторов.
//
// Нужен всюду, где запрос приходит без арендатора и прийти с ним не может:
// вебхуки касс, картинки в теге <img>, публичные страницы. Раньше такие ручки
// обслуживались центральной базой и отвечали «нет данных» при живых данных.
func (h *Handler) poolWithRow(ctx context.Context, existsSQL string, args ...any) *pgxpool.Pool {
	rowExists := func(pool *pgxpool.Pool) bool {
		var exists bool
		if err := pool.QueryRow(ctx, existsSQL, args...).Scan(&exists); err != nil {
			return false
		}
		return exists
	}

	if h.db != nil && rowExists(h.db) {
		return h.db
	}
	if h.tenants == nil {
		return nil
	}
	slugs, err := h.tenants.Slugs(ctx)
	if err != nil {
		log.Printf("поиск базы арендатора: список недоступен: %v", err)
		return nil
	}
	for _, slug := range slugs {
		pool, err := h.tenants.Pool(ctx, slug)
		if err != nil {
			continue
		}
		if rowExists(pool) {
			return pool
		}
	}
	return nil
}

// paymentCoversOrder сверяет сумму, о которой отчиталась касса, с суммой
// заказа. Подпись события проверяется отдельно, так что подделать сумму нельзя;
// это второй рубеж — от чужой настройки кассы и от собственных ошибок в сумме
// заказа. Сверяем только когда касса назвала ту же валюту: у крипто-касс
// оплаченное приходит в монете, и сравнивать его с рублями заказа бессмысленно.
func (h *Handler) paymentCoversOrder(
	ctx context.Context, paymentID string, paidAmount float64, paidCurrency, provider string,
) bool {
	var orderAmount float64
	var orderCurrency string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT amount, COALESCE(currency, 'RUB') FROM core.payments WHERE id = $1::uuid
	`, paymentID).Scan(&orderAmount, &orderCurrency); err != nil {
		log.Printf("%s: сумма заказа %s не прочитана: %v", provider, paymentID, err)
		return false
	}
	if paidAmount <= 0 || paidCurrency == "" {
		// Касса не назвала сумму: полагаемся на статус события, как раньше.
		return true
	}
	if !strings.EqualFold(strings.TrimSpace(paidCurrency), strings.TrimSpace(orderCurrency)) {
		log.Printf("%s: валюта оплаты %s не совпадает с валютой заказа %s по %s — сверка суммы пропущена",
			provider, paidCurrency, orderCurrency, paymentID)
		return true
	}
	// Копеечный допуск: кассы округляют по-разному.
	if paidAmount+0.009 < orderAmount {
		log.Printf("%s: оплачено меньше заказа по %s: касса %.2f, заказ %.2f",
			provider, paymentID, paidAmount, orderAmount)
		return false
	}
	return true
}

// withPaymentTenant возвращает контекст с базой того арендатора, чей это
// платёж. Пул кладём в контекст, а не передаём параметром: его достаёт dbOf, и
// вместе с зачислением в нужную базу попадают проверка подписи, уведомление
// клиенту и событие вебхука — все они ходят тем же способом.
func (h *Handler) withPaymentTenant(ctx context.Context, paymentID string) (context.Context, bool) {
	pool := h.poolForPayment(ctx, paymentID)
	if pool == nil {
		return ctx, false
	}
	return context.WithValue(ctx, tenantPoolKey{}, pool), true
}
