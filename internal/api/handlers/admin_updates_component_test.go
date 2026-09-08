package handlers

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// «Обновить сейчас» на вкладке службы обновления и на вкладке панели — разные
// команды. Признак «применить сейчас» один на установку, и снимает его тот
// компонент, которому он адресован; ошибка в разборе тела означала бы, что
// нажатие на одной вкладке гасит команду с другой.
func TestUpdateComponentOf(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
		ok   bool
	}{
		{"панель", `{"component":"panel-ui"}`, componentPanel, true},
		{"служба обновления", `{"component":"updater"}`, componentUpdater, true},
		{"пробелы обрезаются", `{"component":"  updater  "}`, componentUpdater, true},

		// Панель у клиента и это API обновляются независимо: версия панели,
		// которая ещё не умеет передавать компонент, должна работать как
		// прежде — то есть обновлять панель.
		{"пустое тело", ``, componentPanel, true},
		{"тело без поля", `{}`, componentPanel, true},
		{"поле пустое", `{"component":""}`, componentPanel, true},

		// Агента обновляет воркер отдельным заданием. Приняли бы — флаг
		// взвёлся и остался бы висеть навсегда: за агента службу обновления
		// никто не спрашивает, значит и снять его некому.
		{"агент отклоняется", `{"component":"agent"}`, "", false},
		{"неизвестный компонент", `{"component":"panel"}`, "", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/v1/admin/updates", strings.NewReader(c.body))
			got, ok := updateComponentOf(r)
			if ok != c.ok {
				t.Fatalf("признак пригодности: получено %v, ожидалось %v", ok, c.ok)
			}
			if got != c.want {
				t.Fatalf("компонент: получено %q, ожидалось %q", got, c.want)
			}
		})
	}
}
