package handlers

import (
	"strings"
	"testing"
)

// Внешние ключи стоят с каскадом: удаление локации стирало все серверы
// клиентов, а удаление хостинг-сервера — их аккаунты. Контейнеры и сайты при
// этом продолжали жить на железе, о котором панель уже не знала.
func TestDeletionChecksDependents(t *testing.T) {
	for _, c := range []struct{ file, fn, table string }{
		{"admin_locations.go", "DeleteAdminLocation", "core.servers"},
		{"hosting_admin.go", "DeleteHostingServer", "core.hosting_accounts"},
		{"admin_tariffs.go", "DeleteTariff", "core.servers"},
	} {
		body := funcBody(t, readSource(t, c.file), c.fn)
		if !strings.Contains(body, "count(*) FROM "+c.table) {
			t.Errorf("%s удаляет, не проверив %s", c.fn, c.table)
		}
		if !strings.Contains(body, "StatusConflict") {
			t.Errorf("%s не объясняет отказ: нужен 409 с числом связанных записей", c.fn)
		}
	}
}

// Удаление сервера уходило без wipe, и каталог с данными клиента оставался на
// ноде навсегда, забивая диск.
func TestServerDeleteWipesDataAndFreesAddress(t *testing.T) {
	body := funcBody(t, readSource(t, "servers.go"), "DeleteServer")

	if !strings.Contains(body, `"wipe": true`) {
		t.Error("файлы сервера остаются на ноде после удаления")
	}
	if !strings.Contains(body, "core.ip_pools") || !strings.Contains(body, "'free'") {
		t.Error("выделенный адрес не возвращается в пул")
	}
}

// Упавшая нода запирала удаление навсегда: запись оставалась, а вместе с ней
// квота, порт, адрес и напоминания о продлении.
func TestServerDeleteHasWayOutWhenNodeIsDown(t *testing.T) {
	body := funcBody(t, readSource(t, "servers.go"), "DeleteServer")

	if !strings.Contains(body, "force") {
		t.Error("нет принудительного удаления при недоступной ноде")
	}
	if !strings.Contains(body, "StatusConflict") {
		t.Error("недоступная нода должна отвечать 409 с объяснением, а не сырым 502")
	}
	if !strings.Contains(body, "warning") {
		t.Error("принудительное удаление должно честно сообщать, что контейнер остался")
	}
	if !strings.Contains(body, "node_cleaned") {
		t.Error("в журнале не видно, снят ли контейнер на ноде")
	}
}
