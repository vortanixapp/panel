package gamesettings

import (
	"encoding/json"
	"strings"
	"testing"
)

const factorioJSON = `{
  "name": "Мой сервер",
  "description": "Описание",
  "max_players": 20,
  "visibility": {
    "public": true,
    "lan": false
  },
  "autosave_interval": 10,
  "allow_commands": "admins-only"
}`

func TestJSONParseNested(t *testing.T) {
	c, _ := CodecFor(FormatJSON)
	got := c.Parse(factorioJSON)
	for k, want := range map[string]string{
		"name": "Мой сервер", "description": "Описание", "max_players": "20",
		"visibility.public": "true", "visibility.lan": "false",
		"autosave_interval": "10", "allow_commands": "admins-only",
	} {
		if got[k] != want {
			t.Errorf("%s = %q, ожидалось %q", k, got[k], want)
		}
	}
}

// json.Marshal развалил бы порядок ключей и отступы: один изменённый параметр
// выдавал бы себя за переписанный целиком файл.
func TestJSONApplyKeepsOrderAndIndentation(t *testing.T) {
	c, _ := CodecFor(FormatJSON)
	out, err := c.Apply(factorioJSON, map[string]string{
		"name":              "Новое имя",
		"max_players":       "64",
		"visibility.public": "false",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "name": "Новое имя",
  "description": "Описание",
  "max_players": 64,
  "visibility": {
    "public": false,
    "lan": false
  },
  "autosave_interval": 10,
  "allow_commands": "admins-only"
}`
	checkLines(t, "factorio", out, want)
}

// Типы обязаны сохраняться: игры читают конфиг схемой, и число в кавычках им
// не подходит.
func TestJSONKeepsValueTypes(t *testing.T) {
	c, _ := CodecFor(FormatJSON)
	out, _ := c.Apply(factorioJSON, map[string]string{
		"max_players":       "30",
		"visibility.public": "false",
		"name":              "123",
	})
	if !strings.Contains(out, `"max_players": 30`) {
		t.Errorf("число получило кавычки:\n%s", out)
	}
	if !strings.Contains(out, `"public": false`) {
		t.Errorf("булево записано неверно:\n%s", out)
	}
	// Было строкой — остаётся строкой, даже если значение выглядит числом.
	if !strings.Contains(out, `"name": "123"`) {
		t.Errorf("строка потеряла кавычки:\n%s", out)
	}
	if !json.Valid([]byte(out)) {
		t.Errorf("результат не является корректным JSON:\n%s", out)
	}
}

func TestJSONEscapesAndKeepsCyrillic(t *testing.T) {
	c, _ := CodecFor(FormatJSON)
	out, _ := c.Apply(factorioJSON, map[string]string{"name": `Ка"вы"чки \ и слэш`})
	if !json.Valid([]byte(out)) {
		t.Fatalf("невалидный JSON:\n%s", out)
	}
	if got := c.Parse(out)["name"]; got != `Ка"вы"чки \ и слэш` {
		t.Errorf("обратный разбор дал %q", got)
	}
	// Кириллица не должна превращаться в \u-последовательности.
	out2, _ := c.Apply(factorioJSON, map[string]string{"description": "Привет"})
	if !strings.Contains(out2, "Привет") {
		t.Errorf("кириллица экранирована:\n%s", out2)
	}
}

func TestJSONAppendsMissingField(t *testing.T) {
	c, _ := CodecFor(FormatJSON)
	out, err := c.Apply(factorioJSON, map[string]string{"game_password": "секрет"})
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("невалидный JSON:\n%s", out)
	}
	if got := c.Parse(out)["game_password"]; got != "секрет" {
		t.Errorf("новое поле не записано: %q\n%s", got, out)
	}
	// Прежние значения на месте.
	if got := c.Parse(out)["name"]; got != "Мой сервер" {
		t.Errorf("старое значение поехало: %q", got)
	}
}

func TestJSONAppendsIntoNestedObject(t *testing.T) {
	c, _ := CodecFor(FormatJSON)
	out, err := c.Apply(factorioJSON, map[string]string{"visibility.steam": "true"})
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("невалидный JSON:\n%s", out)
	}
	if got := c.Parse(out)["visibility.steam"]; got != "true" {
		t.Errorf("вложенное поле не записано: %q\n%s", got, out)
	}
	if got := c.Parse(out)["visibility.lan"]; got != "false" {
		t.Errorf("соседнее поле поехало: %q", got)
	}
}

func TestJSONMissingParentIsAnError(t *testing.T) {
	c, _ := CodecFor(FormatJSON)
	// Достраивать игре структуру, которой она не ждёт, мы не беремся — это
	// скорее опечатка в описании профиля.
	if _, err := c.Apply(factorioJSON, map[string]string{"нетТакого.поле": "1"}); err == nil {
		t.Fatal("ожидалась ошибка для несуществующего раздела")
	}
}

const enshroudedJSON = `{
  "name": "Сервер",
  "slotCount": 16,
  "userGroups": [
    {
      "name": "Admin",
      "password": "старый",
      "canKickBan": true
    },
    {
      "name": "Friend",
      "password": "друг",
      "canKickBan": false
    }
  ]
}`

// Выбор элемента массива по значению поля нужен ровно для Enshrouded, где
// пароли лежат в списке групп доступа.
func TestJSONArraySelector(t *testing.T) {
	c, _ := CodecFor(FormatJSON)
	got := c.Parse(enshroudedJSON)
	if got["userGroups[0].name"] != "Admin" || got["userGroups[1].password"] != "друг" {
		t.Errorf("массив разобран неверно: %#v", got)
	}

	out, err := c.Apply(enshroudedJSON, map[string]string{
		"userGroups[name=Admin].password": "новый",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("невалидный JSON:\n%s", out)
	}
	after := c.Parse(out)
	if after["userGroups[0].password"] != "новый" {
		t.Errorf("пароль администратора не изменён: %q", after["userGroups[0].password"])
	}
	// Второй элемент трогать было нельзя.
	if after["userGroups[1].password"] != "друг" {
		t.Errorf("пароль друга изменился: %q", after["userGroups[1].password"])
	}
}

func TestJSONArraySelectorMissingElement(t *testing.T) {
	c, _ := CodecFor(FormatJSON)
	// Группы с таким именем нет — молча создавать её не нужно, но и падать тоже.
	_, err := c.Apply(enshroudedJSON, map[string]string{"userGroups[name=Нет].password": "x"})
	if err == nil {
		t.Fatal("ожидалась ошибка для несуществующего элемента массива")
	}
}

func TestJSONIdempotencyAndRoundTrip(t *testing.T) {
	c, _ := CodecFor(FormatJSON)
	changes := map[string]string{"name": "Мир", "max_players": "48", "game_password": "пароль"}
	once, err := c.Apply(factorioJSON, changes)
	if err != nil {
		t.Fatal(err)
	}
	twice, _ := c.Apply(once, changes)
	if once != twice {
		t.Errorf("не идемпотентно:\n%q\n%q", once, twice)
	}
	got := c.Parse(once)
	for k, v := range changes {
		if got[k] != v {
			t.Errorf("%s = %q, ожидалось %q", k, got[k], v)
		}
	}
}

func TestJSONBrokenInputIsAnError(t *testing.T) {
	c, _ := CodecFor(FormatJSON)
	if _, err := c.Apply(`{"name": `, map[string]string{"name": "x"}); err == nil {
		t.Fatal("ожидалась ошибка на битом JSON")
	}
	// Разбор битого файла просто ничего не находит, но не падает.
	if got := c.Parse(`{"name": `); len(got) != 0 {
		t.Errorf("на битом входе разобрано %#v", got)
	}
}

func TestJSONEmptyObjectGetsField(t *testing.T) {
	c, _ := CodecFor(FormatJSON)
	out, err := c.Apply("{}", map[string]string{"name": "Сервер"})
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("невалидный JSON: %s", out)
	}
	if got := c.Parse(out)["name"]; got != "Сервер" {
		t.Errorf("получено %q из %s", got, out)
	}
}
