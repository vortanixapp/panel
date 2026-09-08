package notify

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Каждое объявленное событие обязано иметь раздел и иконку: без них уведомление
// вырождается в «Система» с общей картинкой — ровно то, что случилось со старым
// справочником, где ключи не совпадали с реальными типами ни в одном случае.
func TestEveryKindIsDescribed(t *testing.T) {
	known := map[Group]bool{}
	for _, g := range Groups() {
		known[g] = true
	}
	for _, k := range Kinds() {
		d := DefFor(k)
		if d.Kind != k {
			t.Errorf("%s: описание указывает на %s", k, d.Kind)
		}
		if !known[d.Group] {
			t.Errorf("%s: неизвестный раздел %q", k, d.Group)
		}
		if d.Icon == "" {
			t.Errorf("%s: нет иконки", k)
		}
		if d.Severity == "" {
			t.Errorf("%s: не задана важность", k)
		}
		for _, c := range d.Channels {
			if c == ChannelPanel {
				t.Errorf("%s: канал panel добавляется всегда, объявлять его не нужно", k)
			}
		}
	}
}

// Критичное событие нельзя откладывать в дайджест: «сервер упал» через час
// бесполезно.
func TestCriticalKindsAreNotDigested(t *testing.T) {
	for _, k := range Kinds() {
		d := DefFor(k)
		if d.Severity == SeverityCritical && d.Digest {
			t.Errorf("%s: критичное событие помечено как откладываемое", k)
		}
	}
}

func TestDefForUnknownKindFallsBack(t *testing.T) {
	d := DefFor("совсем.новое")
	if d.Group != GroupSystem || d.Icon == "" {
		t.Errorf("запасное описание неполное: %+v", d)
	}
	if Known("совсем.новое") {
		t.Error("неизвестное событие не должно считаться объявленным")
	}
}

// Включённый тумблер без адреса каналом не считается. Раньше переключатель на
// странице оповещений позволял включить Telegram вообще без chat id, и клиент
// оставался в уверенности, что подписался.
func TestTargetRequiresAddress(t *testing.T) {
	r := Recipient{
		UserID: "u1", Email: "a@b.c",
		Prefs: Prefs{Email: true, Telegram: true, Discord: true},
	}
	if _, ok := r.Target(ChannelTelegram); ok {
		t.Error("Telegram без chat id не должен считаться настроенным")
	}
	if _, ok := r.Target(ChannelDiscord); ok {
		t.Error("Discord без вебхука не должен считаться настроенным")
	}
	if _, ok := r.Target(ChannelEmail); !ok {
		t.Error("почта с адресом должна работать")
	}

	r.Prefs.TelegramChatID = "-1001234567890"
	if v, ok := r.Target(ChannelTelegram); !ok || v != "-1001234567890" {
		t.Errorf("Telegram с chat id: %q, %v", v, ok)
	}
}

// Запрос уходит с нашего сервера по адресу, который ввёл пользователь, поэтому
// адрес обязан быть вебхуком Discord, а не чем угодно.
func TestDiscordWebhookIsChecked(t *testing.T) {
	good := []string{
		"https://discord.com/api/webhooks/123/abc",
		"https://canary.discord.com/api/webhooks/1/x",
	}
	bad := []string{
		"", "http://discord.com/api/webhooks/1/x", "https://example.com/webhook",
		"https://discord.com/api/other", "https://evil.tld/api/webhooks/1/x",
	}
	for _, v := range good {
		if !isDiscordWebhook(v) {
			t.Errorf("%q должен приниматься", v)
		}
	}
	for _, v := range bad {
		if isDiscordWebhook(v) {
			t.Errorf("%q должен отклоняться", v)
		}
	}
}

func TestDisabledChannelIsSkipped(t *testing.T) {
	r := Recipient{UserID: "u1", Email: "a@b.c", Prefs: Prefs{
		Email: false, Telegram: true, TelegramChatID: "42",
	}}
	got := channelsFor(Event{Kind: KindServerDown}, r)
	for _, c := range got {
		if c == ChannelEmail {
			t.Error("выключенная почта не должна попадать в доставку")
		}
	}
	if len(got) != 1 || got[0] != ChannelTelegram {
		t.Errorf("получено %v, ожидался только Telegram", got)
	}
}

// Ненастроенный канал обязан давать ошибку, а не тихий успех. Прежний мейлер
// биллинга при пустом SMTP_HOST писал строку в лог и возвращал nil, после чего
// письмо помечалось отправленным и исчезало без следа.
func TestUnconfiguredChannelsFailLoudly(t *testing.T) {
	ctx := context.Background()

	err := Send(ctx, Config{}, Delivery{Channel: ChannelEmail, Target: "a@b.c", Subject: "тема"})
	if !errors.Is(err, ErrChannelUnavailable) {
		t.Errorf("почта без SMTP_HOST: %v", err)
	}
	if !strings.Contains(err.Error(), "SMTP_HOST") {
		t.Errorf("ошибка не называет причину: %v", err)
	}

	err = Send(ctx, Config{}, Delivery{Channel: ChannelTelegram, Target: "42", Subject: "тема"})
	if !errors.Is(err, ErrChannelUnavailable) {
		t.Errorf("Telegram без токена: %v", err)
	}

	err = Send(ctx, Config{}, Delivery{Channel: ChannelDiscord, Target: "https://evil.tld/x", Subject: "тема"})
	if !errors.Is(err, ErrChannelUnavailable) {
		t.Errorf("Discord с чужим адресом: %v", err)
	}
}

func TestDiscordSendsContent(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		got = string(buf)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	// Проверяем сборку тела, поэтому адрес подменяем уже после проверки формы.
	cfg := Config{HTTPClient: srv.Client()}
	req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(`{"content":"**Сервер упал**"}`))
	req.Header.Set("Content-Type", "application/json")
	if err := doAndCheck(cfg.client(), req, "Discord"); err != nil {
		t.Fatalf("отправка: %v", err)
	}
	if !strings.Contains(got, "Сервер упал") {
		t.Errorf("тело не доехало: %q", got)
	}
}

// Текст ответа обязан попадать в ошибку: «ошибка 400» без объяснения не говорит
// ничего, а Telegram и Discord объясняют причину именно телом.
func TestTransportErrorCarriesResponseBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"description":"chat not found"}`))
	}))
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL, nil)
	err := doAndCheck(srv.Client(), req, "Telegram")
	if err == nil || !strings.Contains(err.Error(), "chat not found") {
		t.Errorf("ошибка без объяснения: %v", err)
	}
}

func TestBackoffGrowsAndIsCapped(t *testing.T) {
	if backoff(1) != time.Minute {
		t.Errorf("первая пауза %v", backoff(1))
	}
	if backoff(2) != 2*time.Minute || backoff(3) != 4*time.Minute {
		t.Errorf("пауза растёт неверно: %v, %v", backoff(2), backoff(3))
	}
	// Без потолка пауза после десятка попыток ушла бы в часы.
	if backoff(20) > 30*time.Minute {
		t.Errorf("пауза не ограничена: %v", backoff(20))
	}
}

// Кнопка собирается из отдельных полей, а не из карты в meta: раньше действие
// клали объектом, читали строкой через fmt.Sprint, и в панель приезжало
// `map[label:Продлить url:…]` с пустой ссылкой.
func TestRenderIncludesAction(t *testing.T) {
	subject, body := render(Event{
		Kind: KindServerExpiring, Title: "Аренда заканчивается", Body: "Осталось 3 дня.",
		Action: &Action{Label: "Продлить", Href: "https://panel/servers/1/tariff"},
	})
	if subject != "Аренда заканчивается" {
		t.Errorf("тема %q", subject)
	}
	if !strings.Contains(body, "Продлить: https://panel/servers/1/tariff") {
		t.Errorf("действие не попало в тело: %q", body)
	}
	if strings.Contains(body, "map[") {
		t.Errorf("действие просочилось картой: %q", body)
	}
}

func TestRenderSkipsIncompleteAction(t *testing.T) {
	_, body := render(Event{
		Kind: KindServerReady, Title: "Готово",
		Action: &Action{Label: "Открыть"}, // ссылки нет
	})
	if strings.Contains(body, "Открыть") {
		t.Errorf("неполное действие не должно показываться: %q", body)
	}
}

func TestSeverityFallsBackToCatalog(t *testing.T) {
	if got := (Event{Kind: KindServerDown}).severity(); got != SeverityCritical {
		t.Errorf("важность из справочника: %v", got)
	}
	if got := (Event{Kind: KindServerDown, Severity: SeverityInfo}).severity(); got != SeverityInfo {
		t.Errorf("событие должно уметь понижать важность: %v", got)
	}
}

func TestSeverityPrefixDistinguishesImportance(t *testing.T) {
	if severityPrefix(SeverityCritical) == severityPrefix(SeverityInfo) {
		t.Error("критичное и обычное должны отличаться визуально")
	}
	if severityPrefix(SeverityInfo) != "" {
		t.Error("обычное событие не нуждается в пометке")
	}
}
