package handlers

import (
	"strings"
	"testing"
)

// Разрушающие и денежные действия проверяли только принадлежность сервера
// арендатору, то есть «сервер того же хостера». Любой клиент панели мог удалить
// или переустановить сервер соседа, открыть его консоль и оплатить его
// продление со своего кошелька, зная только id.
func TestDestructiveActionsCheckOwner(t *testing.T) {
	for _, c := range []struct{ file, fn, action string }{
		{"servers.go", "DeleteServer", "delete"},
		{"server_lifecycle.go", "ReinstallServer", "reinstall"},
		{"console.go", "CreateConsoleTicket", "console_attach"},
		{"legacy_remaining.go", "ServerRenew", "renew"},
		{"server_ext.go", "ServerBackupList", "files_list"},
		{"server_plugins_maps.go", "ServerPluginsList", "settings_read"},
		{"server_plugins_maps.go", "ServerMapsList", "settings_read"},
	} {
		body := funcBody(t, readSource(t, c.file), c.fn)
		if !strings.Contains(body, "authorizeServerAction") {
			t.Errorf("%s не проверяет владельца сервера", c.fn)
			continue
		}
		if !strings.Contains(body, `"`+c.action+`"`) {
			t.Errorf("%s проверяет не то действие: ожидалось %q", c.fn, c.action)
		}
	}
}

// Удаление сервера другу не доверяют ни при каких правах: у действия нет
// сопоставленного права, значит проходят только владелец и сотрудник.
func TestDeleteStaysOwnerOnly(t *testing.T) {
	if got := agentActionPermission("delete"); got != "" {
		t.Errorf("удаление получило право друга %q — его нельзя делегировать", got)
	}
	if got := agentActionPermission("renew"); got != "" {
		t.Errorf("продление получило право друга %q — деньги списываются с чужого кошелька", got)
	}
}

// Смотреть консоль и вводить команды — разные права. Без первого друг с
// доступом к консоли потерял бы её вовсе.
func TestConsolePermissionsAreSeparate(t *testing.T) {
	if got := agentActionPermission("console_attach"); got != "can_view_console" {
		t.Errorf("просмотр консоли сопоставлен с %q", got)
	}
	if got := agentActionPermission("console_command"); got != "can_console_command" {
		t.Errorf("ввод команд сопоставлен с %q", got)
	}

	// Признак права на ввод кладётся в билет: шлюз видит только его.
	body := funcBody(t, readSource(t, "console.go"), "CreateConsoleTicket")
	if !strings.Contains(body, "can_command") {
		t.Error("билет консоли не несёт признак права на ввод команд")
	}
}

// Ответ в обращении писался по одному id, без арендатора и автора, а ошибка
// вставки глушилась с успешным ответом.
func TestTicketReplyChecksAuthor(t *testing.T) {
	body := funcBody(t, readSource(t, "content.go"), "ReplySupportTicket")

	if !strings.Contains(body, "tenant_id = $2") {
		t.Error("обращение ищется без арендатора")
	}
	if !strings.Contains(body, "ticketUserID != claims.UserID") {
		t.Error("в чужое обращение по-прежнему можно писать")
	}
	if strings.Contains(body, "_, _ = h.dbOf") {
		t.Error("ошибка сохранения снова глушится")
	}
}
