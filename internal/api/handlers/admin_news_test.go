package handlers

import (
	"strings"
	"testing"
	"time"
)

// Заголовки у новостей русские, поэтому слаг без транслита выходил бы пустым.
func TestNewsSlugifyTransliterates(t *testing.T) {
	cases := map[string]string{
		"Новая версия панели":    "novaya-versiya-paneli",
		"Hello World":            "hello-world",
		"Скидка 20% на аренду!":  "skidka-20-na-arendu",
		"  Пробелы   по краям  ": "probely-po-krayam",
		"ЖЁСТКИЙ ДИСК":           "zhestkiy-disk",
		"---":                    "",
		"":                       "",
	}
	for in, want := range cases {
		if got := newsSlugify(in); got != want {
			t.Errorf("newsSlugify(%q) = %q, ожидалось %q", in, got, want)
		}
	}
}

// Слаг обязателен и уникален: у заголовка из одних символов он вышел бы
// пустым, и запись просто не создалась бы.
func TestNewsSlugFallback(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 30, 45, 0, time.UTC)

	if got := newsSlugOrFallback("", "Новость дня", now); got != "novost-dnya" {
		t.Errorf("слаг из заголовка = %q", got)
	}
	if got := newsSlugOrFallback("  своя-ссылка ", "Заголовок", now); got != "своя-ссылка" {
		t.Errorf("явно заданный слаг должен побеждать, получено %q", got)
	}
	got := newsSlugOrFallback("", "!!!", now)
	if got != "news-20260831123045" {
		t.Errorf("запасной слаг = %q, ожидался news-20260831123045", got)
	}
}

func TestValidateNews(t *testing.T) {
	if err := validateNews("Нормальный заголовок", "краткое", "slug"); err != nil {
		t.Errorf("обычная новость отклонена: %v", err)
	}
	if err := validateNews(" ", "", "slug"); err == nil {
		t.Error("пустой заголовок должен отвергаться")
	}
	if err := validateNews("х", "", "slug"); err == nil {
		t.Error("заголовок из одной буквы должен отвергаться")
	}
	if err := validateNews(strings.Repeat("a", newsTitleMaxLen+1), "", "s"); err == nil {
		t.Error("слишком длинный заголовок должен отвергаться")
	}
	if err := validateNews("Заголовок", strings.Repeat("a", newsExcerptMaxLen+1), "s"); err == nil {
		t.Error("слишком длинное описание должно отвергаться")
	}
}

// Дата публикации различает три случая: поле не пришло, дату сняли, дату задали.
func TestParseNewsDateDistinguishesClearing(t *testing.T) {
	if v, has, err := parseNewsDate(nil); v != nil || has || err != nil {
		t.Errorf("отсутствие поля = (%v, %v, %v), дату трогать нельзя", v, has, err)
	}
	empty := ""
	if v, has, err := parseNewsDate(&empty); v != nil || !has || err != nil {
		t.Errorf("пустая строка = (%v, %v, %v), ожидалось снятие даты", v, has, err)
	}
	val := "2026-08-31T18:00"
	if v, has, err := parseNewsDate(&val); v == nil || !has || err != nil {
		t.Errorf("дата со временем не разобрана: (%v, %v, %v)", v, has, err)
	}
	bad := "позавчера"
	if _, _, err := parseNewsDate(&bad); err == nil {
		t.Error("мусор в дате должен отвергаться")
	}
}
