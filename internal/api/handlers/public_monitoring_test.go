package handlers

import (
	"strings"
	"testing"
)

// Публичную страницу мониторинга включает владелец сервера, но проверка стояла
// только в одной ручке из четырёх: онлайн, история нагрузки и баннер отдавались
// по одному id, что бы владелец ни настроил.
func TestPublicMonitoringChecksPublicFlag(t *testing.T) {
	content := readSource(t, "content.go")
	for _, fn := range []string{
		"MonitoringPublicLive",
		"MonitoringPublicStats",
		"MonitoringPublicBanner",
	} {
		if !strings.Contains(funcBody(t, content, fn), "publicMonitoringAllowed") {
			t.Errorf("%s отдаёт данные без проверки публичности", fn)
		}
	}

	// Образец, по которому равняются остальные, остаётся на месте.
	if !strings.Contains(funcBody(t, readSource(t, "monitoring.go"), "MonitoringPublic"), "PublicEnabled") {
		t.Error("основная публичная страница перестала проверять флаг")
	}
}

// Раздел базы знаний не был заведён в правах вовсе, а пустое право middleware
// трактует как отказ: маршруты отвечали 403 любой роли, включая владельца.
func TestKnowledgeBaseHasPermissions(t *testing.T) {
	src := readSource(t, "admin_rbac.go")

	for _, key := range []string{"admin.kb.read", "admin.kb.write"} {
		if !strings.Contains(src, `"`+key+`"`) {
			t.Errorf("право %s не объявлено — владелец не получит его автоматически", key)
		}
	}
	// funcBody берёт только методы, а резолвер — обычная функция.
	if !strings.Contains(src, `case "kb":`) {
		t.Error("маршруты базы знаний не сопоставлены с правом")
	}
}
