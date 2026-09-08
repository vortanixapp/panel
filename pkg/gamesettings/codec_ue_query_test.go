package gamesettings

import (
	"reflect"
	"testing"
)

// Строка запроса взята в кавычки, потому что имя сессии содержит пробел. Без
// кавычек она распалась бы на два аргумента, и игра увидела бы только «Мой» —
// ровно так же её прочитает и разборщик.
const arkStartup = `"TheIsland?listen?SessionName=Мой сервер?MaxPlayers=70?ServerPassword=секрет" -server -log`

func TestUEQueryParse(t *testing.T) {
	c, _ := CodecFor(FormatUEQuery)
	got := c.Parse(arkStartup)
	want := map[string]string{
		MapKey:           "TheIsland",
		"listen":         "",
		"SessionName":    "Мой сервер",
		"MaxPlayers":     "70",
		"ServerPassword": "секрет",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("разобрано %#v\nожидалось %#v", got, want)
	}
}

func TestUEQueryKeepsTrailingFlags(t *testing.T) {
	c, _ := CodecFor(FormatUEQuery)
	// Флаги после строки запроса — не наше дело: их правит вкладка параметров
	// запуска, и вычищать их при смене имени сессии нельзя.
	out, err := c.Apply(arkStartup, map[string]string{"SessionName": "Другое"})
	if err != nil {
		t.Fatal(err)
	}
	want := `TheIsland?listen?SessionName=Другое?MaxPlayers=70?ServerPassword=секрет -server -log`
	if out != want {
		t.Errorf("получено %q\nожидалось %q", out, want)
	}
}

// Значение с пробелом обязано остаться в кавычках, иначе строка распадётся на
// несколько аргументов и игра увидит только её начало.
func TestUEQueryQuotesWhenValueHasSpace(t *testing.T) {
	c, _ := CodecFor(FormatUEQuery)
	out, err := c.Apply(`TheIsland?SessionName=Старое -server`, map[string]string{"SessionName": "Новое имя"})
	if err != nil {
		t.Fatal(err)
	}
	want := `"TheIsland?SessionName=Новое имя" -server`
	if out != want {
		t.Errorf("получено %q\nожидалось %q", out, want)
	}
	if got := c.Parse(out)["SessionName"]; got != "Новое имя" {
		t.Errorf("обратный разбор дал %q", got)
	}
}

func TestUEQueryChangesMap(t *testing.T) {
	c, _ := CodecFor(FormatUEQuery)
	out, _ := c.Apply(arkStartup, map[string]string{MapKey: "Ragnarok"})
	if got := c.Parse(out)[MapKey]; got != "Ragnarok" {
		t.Errorf("карта = %q, строка %q", got, out)
	}
	// Остальные параметры не поехали.
	if got := c.Parse(out)["SessionName"]; got != "Мой сервер" {
		t.Errorf("имя сессии пострадало: %q", got)
	}
}

func TestUEQueryAppendsMissing(t *testing.T) {
	c, _ := CodecFor(FormatUEQuery)
	first, _ := c.Apply(arkStartup, map[string]string{"ServerAdminPassword": "админ", "DifficultyOffset": "1.0"})
	got := c.Parse(first)
	if got["ServerAdminPassword"] != "админ" || got["DifficultyOffset"] != "1.0" {
		t.Errorf("новые параметры не записаны: %#v", got)
	}
	// Порядок дописывания устойчивый.
	for i := 0; i < 20; i++ {
		again, _ := c.Apply(arkStartup, map[string]string{"ServerAdminPassword": "админ", "DifficultyOffset": "1.0"})
		if again != first {
			t.Fatalf("порядок неустойчив:\n%q\n%q", first, again)
		}
	}
}

func TestUEQueryEmptyValueRemovesParam(t *testing.T) {
	c, _ := CodecFor(FormatUEQuery)
	out, _ := c.Apply(arkStartup, map[string]string{"ServerPassword": ""})
	if _, still := c.Parse(out)["ServerPassword"]; still {
		t.Errorf("пароль остался: %q", out)
	}
	if got := c.Parse(out)["MaxPlayers"]; got != "70" {
		t.Errorf("соседний параметр пострадал: %q", got)
	}
}

func TestUEQueryIdempotencyAndEmptyInput(t *testing.T) {
	c, _ := CodecFor(FormatUEQuery)
	changes := map[string]string{"SessionName": "Мир", MapKey: "TheCenter"}
	once, _ := c.Apply(arkStartup, changes)
	twice, _ := c.Apply(once, changes)
	if once != twice {
		t.Errorf("не идемпотентно:\n%q\n%q", once, twice)
	}

	// На пустом входе получается корректная строка, а не мусор.
	fresh, err := c.Apply("", map[string]string{MapKey: "TheIsland", "SessionName": "Новый"})
	if err != nil {
		t.Fatal(err)
	}
	got := c.Parse(fresh)
	if got[MapKey] != "TheIsland" || got["SessionName"] != "Новый" {
		t.Errorf("получено %#v из %q", got, fresh)
	}
}
