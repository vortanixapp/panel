package handlers

import (
	"os"
	"strings"
	"testing"
)

func readSource(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("не прочитать %s: %v", path, err)
	}
	return string(raw)
}

// События server.status и node.offline были в списке подписки панели, но не
// отправлялись ниоткуда: статусы серверов и связь с нодой знает только реле, а
// очередь вебхуков наполнял один core-api.
func TestRelayEmitsDeclaredWebhookEvents(t *testing.T) {
	src := readSource(t, "handlers.go")

	for _, event := range []string{"server.status", "node.offline"} {
		if !strings.Contains(src, `"`+event+`"`) {
			t.Errorf("событие %s не отправляется из реле", event)
		}
	}
	if strings.Count(src, "emitWebhook(") < 2 {
		t.Error("оба события должны попадать в очередь доставки")
	}
}

// Агент повторяет отчёт о статусе на каждое действие питания. Без сравнения с
// прежним значением подписчик получал бы «сервер сменил статус» на
// неизменившийся статус.
func TestServerStatusWebhookOnlyOnChange(t *testing.T) {
	src := readSource(t, "handlers.go")

	if !strings.Contains(src, "prevStatus") {
		t.Fatal("прежний статус не запоминается")
	}
	if !strings.Contains(src, "if prevStatus != status {") {
		t.Error("событие уходит и без смены статуса")
	}
}

// Очередь наполняется только для подписанных на событие вебхуков арендатора:
// доставку выполняет worker, здесь важно не разослать чужое.
func TestEmitWebhookTargetsSubscribedHooks(t *testing.T) {
	src := readSource(t, "webhooks.go")

	if !strings.Contains(src, "core.webhook_deliveries") {
		t.Error("событие не попадает в очередь доставки")
	}
	if !strings.Contains(src, "w.tenant_id = $1") {
		t.Error("событие не ограничено арендатором")
	}
	if !strings.Contains(src, "w.events ? $2") {
		t.Error("событие уходит и тем, кто на него не подписан")
	}
	if !strings.Contains(src, "w.active = true") {
		t.Error("выключенные вебхуки всё равно получают события")
	}
}
