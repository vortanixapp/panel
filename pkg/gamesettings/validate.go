package gamesettings

import (
	"fmt"
	"strconv"
	"strings"
)

// ValidationError — отказ по конкретному полю.
//
// Ключ нужен отдельно от текста: панель подсвечивает то поле, где ошибка, а не
// показывает общее сообщение над формой. Раньше проверки возвращали строку вида
// "max_players: max 5000", и панели приходилось бы разбирать её обратно.
type ValidationError struct {
	Key     string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Key + ": " + e.Message
}

func fail(key, format string, args ...any) error {
	return &ValidationError{Key: key, Message: fmt.Sprintf(format, args...)}
}

// Validate приводит значение из формы к тому виду, в котором оно уйдёт в файл,
// либо объясняет, почему значение не годится.
//
// Значение всегда приходит строкой: панель не знает типов файлов игры, и
// договориться о строке проще, чем поддерживать разбор JSON-типов на каждой
// стороне. Тип задаёт схема, и она же отвечает за перевод.
func Validate(f Field, raw string) (string, error) {
	v := strings.TrimSpace(raw)

	if f.ReadOnly {
		return "", fail(f.Key, "поле «%s» доступно только для чтения", f.Label)
	}

	if v == "" {
		// Пустое значение раньше проходило насквозь и оседало в базе, навсегда
		// закрывая собой то, что реально лежит в конфиге. Теперь очистка — явное
		// свойство поля: пароль снять можно, число игроков стереть нельзя.
		if !f.Clearable {
			return "", fail(f.Key, "поле «%s» не может быть пустым", f.Label)
		}
		return "", nil
	}

	switch f.Kind {
	case KindBool:
		on, ok := ParseBool(v)
		if !ok {
			return "", fail(f.Key, "поле «%s» принимает только да или нет", f.Label)
		}
		return f.BoolLiteral(on), nil

	case KindEnum:
		for _, o := range f.Options {
			if o.Value == v {
				return v, nil
			}
		}
		return "", fail(f.Key, "недопустимое значение поля «%s»: %s", f.Label, v)

	case KindInt:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return "", fail(f.Key, "поле «%s» принимает целое число", f.Label)
		}
		if err := checkRange(f, float64(n)); err != nil {
			return "", err
		}
		return strconv.FormatInt(n, 10), nil

	case KindFloat:
		n, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return "", fail(f.Key, "поле «%s» принимает число", f.Label)
		}
		if err := checkRange(f, n); err != nil {
			return "", err
		}
		return v, nil

	default: // KindString, KindText
		if f.MaxLen > 0 && len([]rune(v)) > f.MaxLen {
			return "", fail(f.Key, "поле «%s» длиннее %d символов", f.Label, f.MaxLen)
		}
		if f.Pattern != nil && !f.Pattern.MatchString(v) {
			return "", fail(f.Key, "поле «%s» заполнено в неверном формате", f.Label)
		}
		return v, nil
	}
}

// checkRange проверяет границы.
//
// Границы — указатели именно ради этого места. В прежних правилах они были
// числами и проверялись как `rule.MinInt != 0`, поэтому граница «не меньше нуля»
// молча не работала: у tf_bot_quota отрицательное значение проходило насквозь.
func checkRange(f Field, n float64) error {
	if f.Min != nil && n < *f.Min {
		return fail(f.Key, "поле «%s» не может быть меньше %s", f.Label, trimNumber(*f.Min))
	}
	if f.Max != nil && n > *f.Max {
		return fail(f.Key, "поле «%s» не может быть больше %s", f.Label, trimNumber(*f.Max))
	}
	return nil
}

func trimNumber(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// ParseBool понимает все написания, которые встречаются в конфигах игр: Palworld
// пишет True, Source-игры 1, Minecraft true.
func ParseBool(v string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on", "enabled":
		return true, true
	case "0", "false", "no", "off", "disabled":
		return false, true
	}
	return false, false
}

// NormalizeForForm переводит значение из файла в тот вид, который ждёт панель.
//
// Обратная сторона Validate: в файле лежит True, 1 или enabled, а тумблеру нужно
// одно из двух известных ему написаний.
func NormalizeForForm(f Field, raw string) string {
	v := strings.TrimSpace(raw)
	if f.Kind != KindBool {
		return v
	}
	if on, ok := ParseBool(v); ok {
		if on {
			return "true"
		}
		return "false"
	}
	return v
}
