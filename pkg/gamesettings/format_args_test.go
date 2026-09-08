package gamesettings

import (
	"reflect"
	"testing"
)

const valheimArgs = `-name "Мой мир" -world Дом -password секрет -public 1 -preset normal -saveinterval 1800`

func TestSplitArgsRespectsQuotes(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"   ", nil},
		{"-a b", []string{"-a", "b"}},
		{`-name "Мой мир"`, []string{"-name", `"Мой мир"`}},
		{`-name 'Мой мир' -x 1`, []string{"-name", `'Мой мир'`, "-x", "1"}},
		{"-a   b\t-c   d", []string{"-a", "b", "-c", "d"}},
		{`-msg "он сказал: привет"`, []string{"-msg", `"он сказал: привет"`}},
		{`-empty ""`, []string{"-empty", `""`}},
	}
	for _, c := range cases {
		got := SplitArgs(c.in)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("SplitArgs(%q) = %#v, ожидалось %#v", c.in, got, c.want)
		}
	}
}

// Прежний разбор был собран с обратной ссылкой `\1`, которую RE2 не
// поддерживает, поэтому regexp.MustCompile падал на любом входе. Проверяем, что
// значения теперь действительно извлекаются.
func TestParseArgsExtractsValues(t *testing.T) {
	flags := []string{"name", "world", "password", "public", "preset", "saveinterval"}
	got := ParseArgs(valheimArgs, flags)
	want := map[string]string{
		"name":         "Мой мир",
		"world":        "Дом",
		"password":     "секрет",
		"public":       "1",
		"preset":       "normal",
		"saveinterval": "1800",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseArgs = %#v, ожидалось %#v", got, want)
	}
}

func TestParseArgsEdgeCases(t *testing.T) {
	flags := []string{"name", "world", "public", "crossplay"}

	// Флаг-переключатель без значения не должен съедать следующий флаг.
	got := ParseArgs("-crossplay -world Дом", flags)
	if got["crossplay"] != "" {
		t.Errorf("переключатель получил значение %q", got["crossplay"])
	}
	if got["world"] != "Дом" {
		t.Errorf("значение соседнего флага потеряно: %#v", got)
	}

	// Пустая строка не должна ни падать, ни выдумывать значения.
	if len(ParseArgs("", flags)) != 0 {
		t.Error("на пустом входе значений быть не должно")
	}

	// Незнакомые флаги игнорируются.
	if v := ParseArgs("-unknown 5 -world Дом", flags); v["world"] != "Дом" || len(v) != 1 {
		t.Errorf("незнакомый флаг повлиял на разбор: %#v", v)
	}

	// Регистр имени флага не важен.
	if v := ParseArgs("-World Дом", flags); v["world"] != "Дом" {
		t.Errorf("разбор чувствителен к регистру: %#v", v)
	}

	// Побеждает первое вхождение — именно его увидит игра.
	if v := ParseArgs("-world Первый -world Второй", flags); v["world"] != "Первый" {
		t.Errorf("при дубле должен побеждать первый: %q", v["world"])
	}

	// Отрицательное число — значение, а не флаг.
	if v := ParseArgs("-public -1", []string{"public"}); v["public"] != "-1" {
		t.Errorf("отрицательное значение принято за флаг: %#v", v)
	}
}

func TestApplyArgsChangesValueInPlace(t *testing.T) {
	got := ApplyArgs(valheimArgs, map[string]string{"world": "Новый"}, "-")
	want := `-name "Мой мир" -world Новый -password секрет -public 1 -preset normal -saveinterval 1800`
	if got != want {
		t.Errorf("получено %q\nожидалось %q", got, want)
	}
}

func TestApplyArgsQuotesValuesWithSpaces(t *testing.T) {
	got := ApplyArgs(`-name Дом`, map[string]string{"name": "Мой большой мир"}, "-")
	if got != `-name "Мой большой мир"` {
		t.Errorf("получено %q", got)
	}
	// Значение с пробелом обязано пережить обратный разбор целиком: прежняя
	// реализация ловила `\S+` и оставляла от «Мой мир» только «"Мой».
	if v := ParseArgs(got, []string{"name"}); v["name"] != "Мой большой мир" {
		t.Errorf("после применения значение стало %q", v["name"])
	}
}

func TestApplyArgsKeepsUnknownArguments(t *testing.T) {
	// Клиент мог дописать своё — сохранение настроек не должно это вычищать.
	got := ApplyArgs(`-crossplay -name Дом -nographics -batchmode`,
		map[string]string{"name": "Мир"}, "-")
	want := `-crossplay -name Мир -nographics -batchmode`
	if got != want {
		t.Errorf("получено %q\nожидалось %q", got, want)
	}
}

func TestApplyArgsAppendsMissingFlagsDeterministically(t *testing.T) {
	changes := map[string]string{"world": "Дом", "public": "1", "preset": "hard"}
	first := ApplyArgs("-name Мир", changes, "-")
	// Прежняя реализация дописывала ключи обходом карты, из-за чего порядок
	// строк менялся от запуска к запуску. Здесь порядок обязан быть устойчивым.
	for i := 0; i < 20; i++ {
		if got := ApplyArgs("-name Мир", changes, "-"); got != first {
			t.Fatalf("порядок дописывания неустойчив:\n%q\n%q", first, got)
		}
	}
	want := `-name Мир -preset hard -public 1 -world Дом`
	if first != want {
		t.Errorf("получено %q\nожидалось %q", first, want)
	}
}

func TestApplyArgsPreservesFlagNameCase(t *testing.T) {
	got := ApplyArgs("", map[string]string{"SteamServerName": "Мир"}, "-")
	if got != `-SteamServerName Мир` {
		t.Errorf("регистр имени флага потерян: %q", got)
	}
}

func TestApplyArgsEmptyValueRemovesFlag(t *testing.T) {
	got := ApplyArgs(`-name Мир -password секрет -public 1`,
		map[string]string{"password": ""}, "-")
	want := `-name Мир -public 1`
	if got != want {
		t.Errorf("получено %q\nожидалось %q", got, want)
	}
	// Иначе очистка пароля в панели оставила бы прежний в силе.
	if v := ParseArgs(got, []string{"password"}); len(v) != 0 {
		t.Errorf("пароль остался: %#v", v)
	}
}

func TestApplyArgsSupportsPlusPrefix(t *testing.T) {
	// У CS 1.6 стартовая карта передаётся как +map.
	got := ApplyArgs(`-game cstrike +map de_dust2`, map[string]string{"map": "de_inferno"}, "+")
	if got != `-game cstrike +map de_inferno` {
		t.Errorf("получено %q", got)
	}
	if v := ParseArgs(got, []string{"map"}); v["map"] != "de_inferno" {
		t.Errorf("карта разобралась как %q", v["map"])
	}
	// Дописывание, когда флага ещё нет.
	added := ApplyArgs(`-game cstrike`, map[string]string{"map": "de_nuke"}, "+")
	if added != `-game cstrike +map de_nuke` {
		t.Errorf("дописано неверно: %q", added)
	}
}

func TestApplyArgsRoundTrip(t *testing.T) {
	flags := []string{"name", "world", "password", "public", "preset", "saveinterval"}
	changes := map[string]string{"name": "Другой мир", "public": "0"}

	once := ApplyArgs(valheimArgs, changes, "-")
	twice := ApplyArgs(once, changes, "-")
	if once != twice {
		t.Errorf("применение не идемпотентно:\n%q\n%q", once, twice)
	}

	got := ParseArgs(once, flags)
	if got["name"] != "Другой мир" || got["public"] != "0" {
		t.Errorf("изменения не читаются обратно: %#v", got)
	}
	// Остальные значения не должны были пострадать.
	if got["world"] != "Дом" || got["saveinterval"] != "1800" || got["password"] != "секрет" {
		t.Errorf("соседние значения поехали: %#v", got)
	}
}
