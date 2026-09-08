package handlers

import (
	"encoding/json"
	"testing"
)

func decodeActions(t *testing.T, raw []byte) []map[string]any {
	t.Helper()
	var out []map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("не разобрать результат: %v (%s)", err, raw)
	}
	return out
}

// Правила сборки действий взяты из старой панели дословно. Каждое из них легко
// потерять при переносе, а цена потери — молча неработающий плагин.
func TestNormalizePluginActionsFollowsOldPanelRules(t *testing.T) {
	in := []any{
		// Пустой путь — строка пропускается.
		map[string]any{"path": "  ", "action": "append_lines", "lines": "a"},
		// Выход за пределы каталога — тоже.
		map[string]any{"path": "../etc/passwd", "action": "append_lines", "lines": "a"},
		// Неизвестное действие молча становится ensure_contains.
		map[string]any{"path": "cfg/server.cfg", "action": "нет-такого", "lines": "sv_gravity 800"},
		// Пустой список строк — строка пропускается.
		map[string]any{"path": "cfg/empty.cfg", "action": "append_lines", "lines": "   \n\n  "},
		// replace_regex без pattern — пропускается.
		map[string]any{"path": "cfg/re.cfg", "action": "replace_regex", "replacement": "x"},
		// Нормальный replace_regex: lines не сохраняются.
		map[string]any{"path": "cfg/ok.cfg", "action": "replace_regex", "pattern": "^a$", "replacement": "b", "lines": "мусор"},
		// Многострочный текст режется, пустые строки выбрасываются.
		map[string]any{"path": "cfg/multi.cfg", "action": "write_file", "lines": "one\r\ntwo\r\r\nthree\n\n", "create_if_missing": "on"},
	}

	got := decodeActions(t, normalizePluginActions(in))
	if len(got) != 3 {
		t.Fatalf("осталось %d действий, ожидалось 3: %+v", len(got), got)
	}

	if got[0]["action"] != "ensure_contains" {
		t.Errorf("неизвестное действие должно становиться ensure_contains, получено %v", got[0]["action"])
	}
	if got[0]["create_if_missing"] != false {
		t.Error("галочка без значения должна быть выключена")
	}

	if got[1]["pattern"] != "^a$" || got[1]["replacement"] != "b" {
		t.Errorf("replace_regex собран неверно: %+v", got[1])
	}
	if _, has := got[1]["lines"]; has {
		t.Error("для replace_regex строки не сохраняются")
	}

	lines, _ := got[2]["lines"].([]any)
	if len(lines) != 3 || lines[0] != "one" || lines[1] != "two" || lines[2] != "three" {
		t.Errorf("многострочный текст разобран неверно: %+v", lines)
	}
	if got[2]["create_if_missing"] != true {
		t.Error("значение «on» должно включать галочку")
	}

	if string(normalizePluginActions(nil)) != "[]" {
		t.Error("пустой вход должен давать пустой массив, а не null")
	}
}

func TestNormalizeInstallPath(t *testing.T) {
	cases := map[string]string{
		"  /addons/amxmodx/  ": "addons/amxmodx",
		"addons":               "addons",
		"":                     "",
		"/":                    "",
	}
	for in, want := range cases {
		got, err := normalizeInstallPath(in)
		if err != nil || got != want {
			t.Errorf("normalizeInstallPath(%q) = (%q, %v), ожидалось %q", in, got, err, want)
		}
	}
	if _, err := normalizeInstallPath("addons/../../etc"); err == nil {
		t.Error("путь с «..» должен отвергаться")
	}
}

// «Все игры» и пустой список — одно и то же; выбор без единой игры должен
// отвергаться с понятной ошибкой, как в старой панели.
func TestNormalizeSupportedGames(t *testing.T) {
	if got, err := normalizeSupportedGames(true, []any{"cs16"}); err != nil || len(got) != 0 {
		t.Errorf("режим «все игры» = (%v, %v), ожидался пустой список", got, err)
	}
	got, err := normalizeSupportedGames(false, []any{"CS16", "cs16", " rust ", "*", ""})
	if err != nil {
		t.Fatalf("неожиданная ошибка: %v", err)
	}
	if len(got) != 2 || got[0] != "cs16" || got[1] != "rust" {
		t.Errorf("список игр = %v, ожидалось [cs16 rust]", got)
	}
	if _, err := normalizeSupportedGames(false, []any{"*", " "}); err == nil {
		t.Error("пустой выбор без «все игры» должен отвергаться")
	}
}
