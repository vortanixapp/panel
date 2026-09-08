package handlers

import (
	"strings"
	"testing"
)

// Экраны кабинета показывали сотруднику чужие серверы: выборка расширялась по
// роли, а не по разделу. Для чужих серверов есть раздел администратора, и
// широкий список должен оставаться только там.
func TestUserPanelListsOnlyOwnServers(t *testing.T) {
	for _, c := range []struct{ file, fn string }{
		{"servers.go", "ListServers"},
		{"servers_enriched.go", "MyServers"},
		{"dashboard.go", "Dashboard"},
	} {
		body := funcBody(t, readSource(t, c.file), c.fn)
		if strings.Contains(body, "isStaffRole") {
			t.Errorf("%s снова расширяет выборку по роли", c.fn)
		}
		if !strings.Contains(body, "user_id = $2") {
			t.Errorf("%s не ограничивает выборку владельцем", c.fn)
		}
	}
}

// Признак «все владельцы» передаётся явно: раньше его подменяла роль, и любой
// сотрудник, открывший кабинет, получал список всего арендатора.
func TestEnrichedListTakesExplicitScope(t *testing.T) {
	src := readSource(t, "servers_enriched.go")

	if !strings.Contains(src, "allOwners bool") {
		t.Fatal("область выборки не передаётся явным признаком")
	}
	body := funcBody(t, src, "listEnrichedServers")
	if strings.Contains(body, "isStaffRole") {
		t.Error("выборка снова зависит от роли вызывающего")
	}

	// Раздел администратора обязан остаться широким: он для того и сделан.
	admin := funcBody(t, readSource(t, "legacy_remaining.go"), "AdminListServers")
	if !strings.Contains(admin, "true") {
		t.Error("список администратора перестал показывать чужие серверы")
	}
}

// Список кабинета кэшировался под общим для арендатора ключом, хотя зависел от
// того, кто смотрит: ответ сотрудника доставался следующему клиенту.
func TestServersCacheKeyIsPerUser(t *testing.T) {
	body := funcBody(t, readSource(t, "servers.go"), "ListServers")

	if !strings.Contains(body, `":servers:" + claims.UserID`) {
		t.Error("ключ кэша не различает пользователей")
	}
}
