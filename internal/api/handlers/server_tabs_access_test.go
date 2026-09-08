package handlers

import (
	"strings"
	"testing"
)

// Вкладки планировщика, firewall, портов и друзей проверяли только арендатора:
// зная id, любой вошедший клиент правил чужой сервер. Тест сторожит возврат к
// ensureServerForTenant в этих ручках.
func TestServerTabsUseOwnerAwareAuthorization(t *testing.T) {
	src := readSource(t, "server_ext.go")

	for _, fn := range []string{
		"ServerCronList", "ServerCronCreate", "ServerCronDelete", "ServerCronToggle",
		"ServerFirewallList", "ServerFirewallCreate", "ServerFirewallDelete", "ServerFirewallToggle",
		"ServerPortsList", "ServerPortsCreate", "ServerPortsDelete",
		"ServerFriendsList", "ServerFriendsAdd", "ServerFriendsRemove", "ServerFriendsUpdate",
	} {
		body := funcBody(t, src, fn)
		if strings.Contains(body, "ensureServerForTenant") {
			t.Errorf("%s снова проверяет только арендатора", fn)
		}
		if !strings.Contains(body, "authorizeServerTab") {
			t.Errorf("%s не проверяет владельца и права друга", fn)
		}
	}
}

// Журнал установки и расписание бэкапов страдали тем же: проверялся только
// арендатор, и чужой сервер читался по id.
func TestBackupScheduleAndInstallLogCheckOwner(t *testing.T) {
	for _, c := range []struct{ file, fn string }{
		{"backup_schedule.go", "GetServerBackupSchedule"},
		{"backup_schedule.go", "PutServerBackupSchedule"},
		{"install_log.go", "ServerInstallLog"},
	} {
		body := funcBody(t, readSource(t, c.file), c.fn)
		if strings.Contains(body, "ensureServerForTenant") {
			t.Errorf("%s снова проверяет только арендатора", c.fn)
		}
		if !strings.Contains(body, "authorizeServerTab") {
			t.Errorf("%s не проверяет владельца и права друга", c.fn)
		}
	}
}

// Право на вкладку портов должно быть объявлено: без него друг с полным
// доступом упирался бы в отказ, а проверка сводилась бы к «владелец».
func TestPortsPermissionsDeclared(t *testing.T) {
	access := readSource(t, "server_access.go")
	for _, key := range []string{"can_view_ports", "can_ports_manage"} {
		if !strings.Contains(access, key) {
			t.Errorf("право %s не объявлено в наборе прав", key)
		}
	}

	agent := readSource(t, "server_agent.go")
	for _, action := range []string{"ports_list", "ports_create", "ports_delete", "friends_list"} {
		if !strings.Contains(agent, action) {
			t.Errorf("действие %s не сопоставлено ни с одним правом", action)
		}
	}
}

// Закрытие обращения искало тикет по одному id, без арендатора и автора.
func TestCloseSupportTicketChecksOwner(t *testing.T) {
	body := funcBody(t, readSource(t, "content.go"), "CloseSupportTicket")

	if !strings.Contains(body, "tenant_id") {
		t.Error("закрытие тикета не ограничено арендатором")
	}
	if !strings.Contains(body, "isStaffRole") || !strings.Contains(body, "claims.UserID") {
		t.Error("закрыть чужой тикет по-прежнему может кто угодно")
	}
}

// funcBody возвращает тело функции по её имени: от строки объявления до
// следующего объявления верхнего уровня.
func funcBody(t *testing.T, src, name string) string {
	t.Helper()
	marker := ") " + name + "("
	i := strings.Index(src, marker)
	if i < 0 {
		t.Fatalf("функция %s не найдена", name)
	}
	rest := src[i:]
	if j := strings.Index(rest[1:], "\nfunc "); j >= 0 {
		return rest[:j+1]
	}
	return rest
}
