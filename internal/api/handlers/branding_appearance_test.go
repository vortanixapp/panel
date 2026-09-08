package handlers

import (
	"strings"
	"testing"
)

// Настройки оформления сохранялись целиком, а до панели доезжал один цвет:
// палитра и набор блоков никуда не отдавались и потому ни на что не влияли.
func TestBrandingReturnsAppearanceSettings(t *testing.T) {
	body := funcBody(t, readSource(t, "branding.go"), "GetBranding")

	for _, key := range []string{
		"app.site.template.colors",
		"app.site.template.blocks",
		"app.site.template.user_menu_variant",
	} {
		if !strings.Contains(body, key) {
			t.Errorf("настройка %s не читается", key)
		}
	}
	for _, field := range []string{`"colors"`, `"blocks"`, `"user_menu_variant"`} {
		if !strings.Contains(body, field) {
			t.Errorf("поле %s не отдаётся панели", field)
		}
	}
}

// Значения правит человек руками, поэтому сломанный JSON не должен ронять
// брендинг всей панели: логотип и название нужны и при мусоре в палитре.
func TestAppearanceParsersSurviveGarbage(t *testing.T) {
	if got := jsonObjectOfStrings("{не json"); len(got) != 0 {
		t.Errorf("сломанный JSON палитры должен дать пустой набор, получено %v", got)
	}
	if got := jsonObjectOfStrings(`{"primary":"#ff0000","secondary":"","weight":3}`); len(got) != 1 || got["primary"] != "#ff0000" {
		t.Errorf("нестроковые и пустые значения должны отсеиваться, получено %v", got)
	}
	if got := jsonObjectOfBools("[]"); len(got) != 0 {
		t.Errorf("массив вместо объекта должен дать пустой набор, получено %v", got)
	}

	blocks := jsonObjectOfBools(`{"hero":true,"faq":false,"pricing":"false","games":"true","cta":0}`)
	for key, want := range map[string]bool{
		"hero": true, "faq": false, "pricing": false, "games": true, "cta": false,
	} {
		if blocks[key] != want {
			t.Errorf("блок %s: %v, ожидалось %v", key, blocks[key], want)
		}
	}
}
