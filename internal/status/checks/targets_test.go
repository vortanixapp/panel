package checks

import (
	"net/url"
	"strings"
	"testing"
)

// Прежде этот тест сверял адреса целей с таблицей DNS в руководстве по проду.
// В открытом репозитории такого документа нет: он описывает инфраструктуру
// одной конкретной установки. Сверять не с чем, поэтому проверяем то, что
// имеет смысл в любой установке, — что список целей вообще пригоден к работе.
//
// Сами адреса в defaults пока указывают на серверы одной установки. Это
// временно: цели обязаны приходить из настроек, иначе у постороннего страница
// состояния показывает чужие сервисы.
func TestDefaultTargetsAreUsable(t *testing.T) {
	if len(defaults) == 0 {
		t.Fatal("список целей пуст — страница состояния не покажет ничего")
	}

	seen := map[string]bool{}
	for _, target := range defaults {
		if target.Key == "" {
			t.Errorf("цель %q без ключа: по ключу различают записи в истории", target.Name)
		}
		if seen[target.Key] {
			t.Errorf("ключ %q встречается дважды — одна цель затрёт историю другой", target.Key)
		}
		seen[target.Key] = true

		parsed, err := url.Parse(target.URL)
		if err != nil {
			t.Errorf("%s: неразбираемый адрес %q: %v", target.Key, target.URL, err)
			continue
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			t.Errorf("%s: схема %q, а опрос идёт по HTTP", target.Key, parsed.Scheme)
		}
		if parsed.Hostname() == "" {
			t.Errorf("%s: в адресе %q нет хоста", target.Key, target.URL)
		}
		// Путь приклеивается к адресу: без ведущей косой черты склейка даёт
		// мусор вроде https://example.comhealth.
		if target.Path != "" && !strings.HasPrefix(target.Path, "/") {
			t.Errorf("%s: путь %q без ведущей косой черты", target.Key, target.Path)
		}
	}
}

func TestBillingTargetChecksAPIThroughProxy(t *testing.T) {
	for _, target := range defaults {
		if target.Key != "billing" {
			continue
		}
		if target.Path != "/api/health" {
			t.Errorf("путь проверки биллинга %q: нужен /api/health, иначе "+
				"опрашивается веб-интерфейс, а не API, и мёртвый API остаётся незамеченным",
				target.Path)
		}
		return
	}
	t.Fatal("в defaults нет цели billing")
}
