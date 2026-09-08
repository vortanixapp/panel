package gamesettings

import (
	"errors"
	"regexp"
	"testing"
)

func intField(key string, min, max *float64) Field {
	return Field{Key: key, Label: key, Kind: KindInt, Section: SectionGeneral, Min: min, Max: max}
}

func TestValidateNumbers(t *testing.T) {
	minP, maxP := Range(1, 5000)
	f := intField("max_players", minP, maxP)

	if got, err := Validate(f, " 20 "); err != nil || got != "20" {
		t.Fatalf("Validate(20) = %q, %v", got, err)
	}
	if _, err := Validate(f, "0"); err == nil {
		t.Error("значение ниже нижней границы должно отклоняться")
	}
	if _, err := Validate(f, "5001"); err == nil {
		t.Error("значение выше верхней границы должно отклоняться")
	}
	if _, err := Validate(f, "двадцать"); err == nil {
		t.Error("нечисло должно отклоняться")
	}
}

// Прежние правила хранили границы числами и проверяли их как `rule.MinInt != 0`,
// поэтому нижняя граница «не меньше нуля» не работала вовсе: у tf_bot_quota
// отрицательное значение проходило насквозь. Границы стали указателями ради
// этого случая — тест закрепляет разницу.
func TestValidateZeroIsARealLowerBound(t *testing.T) {
	f := intField("tf_bot_quota", AtLeast(0), AtMost(1000))

	if _, err := Validate(f, "-1"); err == nil {
		t.Fatal("при нижней границе 0 отрицательное значение должно отклоняться")
	}
	if got, err := Validate(f, "0"); err != nil || got != "0" {
		t.Fatalf("сам ноль должен приниматься: %q, %v", got, err)
	}

	// Без границ вовсе отрицательное значение законно.
	free := intField("offset", nil, nil)
	if _, err := Validate(free, "-5"); err != nil {
		t.Fatalf("без границ отрицательное значение должно приниматься: %v", err)
	}
}

func TestValidateBoolWritesGameSpelling(t *testing.T) {
	cases := []struct {
		name  string
		field Field
		in    string
		want  string
	}{
		{"Palworld пишет True", Field{Key: "b", Label: "b", Kind: KindBool, True: "True", False: "False"}, "true", "True"},
		{"Palworld пишет False", Field{Key: "b", Label: "b", Kind: KindBool, True: "True", False: "False"}, "false", "False"},
		{"Source пишет 1", Field{Key: "b", Label: "b", Kind: KindBool, True: "1", False: "0"}, "true", "1"},
		{"Minecraft пишет true", Field{Key: "b", Label: "b", Kind: KindBool}, "true", "true"},
		{"на входе принимаем и 1", Field{Key: "b", Label: "b", Kind: KindBool, True: "True", False: "False"}, "1", "True"},
		{"на входе принимаем и yes", Field{Key: "b", Label: "b", Kind: KindBool, True: "True", False: "False"}, "yes", "True"},
	}
	for _, c := range cases {
		got, err := Validate(c.field, c.in)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: получено %q, ожидалось %q", c.name, got, c.want)
		}
	}

	if _, err := Validate(Field{Key: "b", Label: "b", Kind: KindBool}, "возможно"); err == nil {
		t.Error("непонятное булево должно отклоняться")
	}
}

func TestNormalizeForFormReadsAnySpelling(t *testing.T) {
	f := Field{Key: "b", Label: "b", Kind: KindBool, True: "True", False: "False"}
	for in, want := range map[string]string{
		"True": "true", "False": "false", "1": "true", "0": "false",
		"enabled": "true", "off": "false",
	} {
		if got := NormalizeForForm(f, in); got != want {
			t.Errorf("NormalizeForForm(%q) = %q, ожидалось %q", in, got, want)
		}
	}
	// Строковое поле не трогаем, только пробелы по краям.
	s := Field{Key: "s", Label: "s", Kind: KindString}
	if got := NormalizeForForm(s, "  Мой сервер  "); got != "Мой сервер" {
		t.Errorf("строку исказило: %q", got)
	}
}

func TestValidateEnum(t *testing.T) {
	f := Field{Key: "difficulty", Label: "Сложность", Kind: KindEnum, Options: []Option{
		{Value: "peaceful", Label: "Мирная"}, {Value: "hard", Label: "Высокая"},
	}}
	if got, err := Validate(f, "hard"); err != nil || got != "hard" {
		t.Fatalf("Validate(hard) = %q, %v", got, err)
	}
	if _, err := Validate(f, "nightmare"); err == nil {
		t.Error("значение вне списка должно отклоняться")
	}
}

func TestValidateStringLimits(t *testing.T) {
	f := Field{Key: "hostname", Label: "Имя", Kind: KindString, MaxLen: 5}
	// Длину считаем в символах, а не байтах: иначе кириллица укладывалась бы
	// вдвое короче латиницы при одном и том же ограничении.
	if _, err := Validate(f, "Сервер"); err == nil {
		t.Error("шесть символов при пределе пять должны отклоняться")
	}
	if got, err := Validate(f, "Сервъ"); err != nil || got != "Сервъ" {
		t.Fatalf("пять кириллических символов должны проходить: %q, %v", got, err)
	}

	p := Field{Key: "map", Label: "Карта", Kind: KindString, Pattern: regexp.MustCompile(`^[A-Za-z0-9_-]+$`)}
	if _, err := Validate(p, "de dust2"); err == nil {
		t.Error("пробел в имени карты должен отклоняться")
	}
}

func TestValidateEmptyOnlyWhenClearable(t *testing.T) {
	required := Field{Key: "max_players", Label: "Мест", Kind: KindInt}
	if _, err := Validate(required, ""); err == nil {
		t.Error("пустое значение обязательного поля должно отклоняться")
	}

	optional := Field{Key: "sv_password", Label: "Пароль", Kind: KindString, Clearable: true}
	got, err := Validate(optional, "  ")
	if err != nil || got != "" {
		t.Fatalf("очищаемое поле должно принимать пустоту: %q, %v", got, err)
	}
}

func TestValidateRejectsReadOnly(t *testing.T) {
	f := Field{Key: "server_port", Label: "Порт", Kind: KindInt, ReadOnly: true}
	if _, err := Validate(f, "27015"); err == nil {
		t.Fatal("поле только для чтения должно отклоняться")
	}
}

func TestValidationErrorCarriesKey(t *testing.T) {
	f := intField("max_players", AtLeast(1), nil)
	_, err := Validate(f, "0")
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("ожидалась ValidationError, получено %T", err)
	}
	// Ключ нужен панели, чтобы подсветить конкретное поле, а не всю форму.
	if ve.Key != "max_players" {
		t.Errorf("ключ %q, ожидался max_players", ve.Key)
	}
}
