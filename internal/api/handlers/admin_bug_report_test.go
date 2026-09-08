package handlers

import (
	"strings"
	"testing"
)

// Раздел админки, которого нет в резолвере прав, получает пустое право, а
// пустое право adminRBACMiddleware трактует как отказ. Так страница «Сообщить
// об ошибке» отвечала 403 любой роли, включая владельца: маршрут был, форма
// была, а отправить отчёт не мог никто.
func TestBugReportRouteResolvesPermission(t *testing.T) {
	cases := []struct {
		method string
		want   string
	}{
		{"GET", "admin.bug_report.read"},
		{"POST", "admin.bug_report.write"},
	}
	for _, c := range cases {
		got, bypass := rbacResolveAdminPermission("/v1/admin/bug-report", c.method)
		if bypass {
			t.Errorf("%s /v1/admin/bug-report: раздел не должен обходить проверку прав", c.method)
		}
		if got != c.want {
			t.Errorf("%s /v1/admin/bug-report: право %q, ожидалось %q", c.method, got, c.want)
		}
	}
}

// Право, которого нет в общем списке, не попадёт в матрицу групп: хостер не
// увидит его в /admin/groups и не сможет выдать роли.
func TestBugReportPermissionsListed(t *testing.T) {
	keys := rbacAllPermissionKeys()
	labels := rbacPermissionLabels()
	for _, want := range []string{"admin.bug_report.read", "admin.bug_report.write"} {
		found := false
		for _, k := range keys {
			if k == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("права %q нет в rbacAllPermissionKeys", want)
		}
		if labels[want] == "" {
			t.Errorf("у права %q нет подписи для матрицы групп", want)
		}
	}
}

// Форма обещает «обращение появится в поддержке с приоритетом по критичности».
// Пока перевода не было, все отчёты ложились с приоритетом по умолчанию.
func TestBugSeverityMapsToPriority(t *testing.T) {
	want := map[string]string{
		"low":      "low",
		"medium":   "normal",
		"high":     "high",
		"critical": "urgent",
	}
	for severity, priority := range want {
		if got := bugSeverityPriority[severity]; got != priority {
			t.Errorf("критичность %q → приоритет %q, ожидался %q", severity, got, priority)
		}
	}
	if len(bugSeverityPriority) != len(want) {
		t.Errorf("в таблице приоритетов %d ступеней, в форме %d", len(bugSeverityPriority), len(want))
	}
}

func TestBugReportTitleStripsSubjectPrefix(t *testing.T) {
	cases := map[string]string{
		"[BUG][HIGH][panel-ui] Лимит слотов не сохраняется": "Лимит слотов не сохраняется",
		"[BUG][LOW][core-api] Опечатка в ответе":            "Опечатка в ответе",
		"Обычное обращение без префикса":                    "Обычное обращение без префикса",
		"[BUG][MEDIUM][agent]": "",
	}
	for subject, want := range cases {
		if got := bugReportTitle(subject); got != want {
			t.Errorf("bugReportTitle(%q) = %q, ожидалось %q", subject, got, want)
		}
	}
}

// Отчёт без текста разобрать нельзя, а на вид он полноценный: раньше ошибка
// вставки первого сообщения глушилась и обращение оставалось пустым.
func TestBugReportCreateIsTransactional(t *testing.T) {
	src := readSource(t, "admin.go")
	body := funcBody(t, src, "AdminBugReportCreate")

	if !strings.Contains(body, "Begin(ctx)") || !strings.Contains(body, "tx.Commit(ctx)") {
		t.Error("обращение и его первое сообщение должны писаться одной транзакцией")
	}
	if strings.Contains(body, "_, _ = tx.Exec") || strings.Contains(body, "_, _ = h.dbOf") {
		t.Error("вернулась заглушённая ошибка вставки сообщения")
	}
	if !strings.Contains(body, "priority") || !strings.Contains(body, "category") {
		t.Error("обращение должно создаваться с категорией и приоритетом")
	}
}
