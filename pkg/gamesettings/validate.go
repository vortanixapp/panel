package gamesettings

import (
	"fmt"
	"strconv"
	"strings"
)

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

func Validate(f Field, raw string) (string, error) {
	v := strings.TrimSpace(raw)

	if f.ReadOnly {
		return "", fail(f.Key, "поле «%s» доступно только для чтения", f.Label)
	}

	if v == "" {
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

	default:
		if f.MaxLen > 0 && len([]rune(v)) > f.MaxLen {
			return "", fail(f.Key, "поле «%s» длиннее %d символов", f.Label, f.MaxLen)
		}
		if f.Pattern != nil && !f.Pattern.MatchString(v) {
			return "", fail(f.Key, "поле «%s» заполнено в неверном формате", f.Label)
		}
		return v, nil
	}
}

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

func ParseBool(v string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on", "enabled":
		return true, true
	case "0", "false", "no", "off", "disabled":
		return false, true
	}
	return false, false
}

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
