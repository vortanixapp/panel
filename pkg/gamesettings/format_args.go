package gamesettings

import (
	"sort"
	"strings"
)

// Аргументы командной строки как формат настроек.
//
// Для части игр это единственный способ что-то настроить: общий шаблон
// entrypoint конфига не создаёт, а имя мира, число мест и пароль игра берёт из
// аргументов. Так устроен Valheim, так же настраиваются ARK, Conan, Squad и
// Soulmask.
//
// Прежний разбор был собран на регулярном выражении с обратной ссылкой `\1`
// («закрывающая кавычка совпадает с открывающей»). Движок regexp в Go — RE2,
// обратных ссылок он не поддерживает, поэтому regexp.MustCompile падал ещё до
// сопоставления, и parseValheimStartupParams паниковал на любом входе, включая
// пустую строку. Значений он не вернул ни разу за всё время жизни кода.
//
// Регулярным выражением это и не решается: кавычки требуют разбора состоянием.
// Поэтому здесь обычный разборщик по символам.

// SplitArgs разбивает строку аргументов на токены, уважая кавычки.
//
// Токен сохраняется в том виде, в каком записан, вместе с кавычками: строка
// пересобирается из токенов, и снимать кавычки на этом шаге значило бы менять
// не относящиеся к делу аргументы при каждом сохранении.
func SplitArgs(s string) []string {
	var (
		out   []string
		cur   strings.Builder
		quote rune
		open  bool
	)
	flush := func() {
		if open {
			out = append(out, cur.String())
			cur.Reset()
			open = false
		}
	}
	for _, r := range s {
		switch {
		case quote != 0:
			cur.WriteRune(r)
			if r == quote {
				quote = 0
			}
		case r == '"' || r == '\'':
			quote = r
			cur.WriteRune(r)
			open = true
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			flush()
		default:
			cur.WriteRune(r)
			open = true
		}
	}
	flush()
	return out
}

// Unquote снимает парные кавычки с токена.
func Unquote(tok string) string {
	if len(tok) >= 2 {
		if (tok[0] == '"' && tok[len(tok)-1] == '"') || (tok[0] == '\'' && tok[len(tok)-1] == '\'') {
			return tok[1 : len(tok)-1]
		}
	}
	return tok
}

// QuoteArg оборачивает значение в кавычки, если без них оно распадётся на
// несколько аргументов.
func QuoteArg(v string) string {
	if v == "" {
		return `""`
	}
	if !strings.ContainsAny(v, " \t\"'") {
		return v
	}
	// Кавычку внутри значения не экранируем, а заменяем на одинарную: у игр,
	// читающих аргументы, единого правила экранирования нет, и попытка его
	// придумать даёт строку, которую не разберёт ни одна из них.
	return `"` + strings.ReplaceAll(v, `"`, `'`) + `"`
}

// IsFlag сообщает, что токен — это флаг, а не значение.
//
// Отрицательное число флагом не считаем: у Valheim таких значений нет, но у
// смещения сложности в ARK они появляются, и принять «-1» за флаг значило бы
// потерять значение соседнего.
func IsFlag(tok string) bool {
	if len(tok) < 2 {
		return false
	}
	if tok[0] != '-' && tok[0] != '+' {
		return false
	}
	c := tok[1]
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '-'
}

// ParseArgs достаёт значения перечисленных флагов из строки аргументов.
//
// Имя флага задаётся без ведущего знака: "name" найдёт и "-name", и "+name" —
// у CS 1.6 карта передаётся как "+map", у Valheim мир как "-world".
func ParseArgs(s string, flags []string) map[string]string {
	tokens := SplitArgs(s)
	want := make(map[string]string, len(flags))
	for _, f := range flags {
		want[strings.ToLower(f)] = f
	}

	out := map[string]string{}
	for i, tok := range tokens {
		if !IsFlag(tok) {
			continue
		}
		name, ok := want[strings.ToLower(strings.TrimLeft(tok, "-+"))]
		if !ok {
			continue
		}
		if _, seen := out[name]; seen {
			// Побеждает первое вхождение: именно его увидит игра, разбирая
			// командную строку слева направо.
			continue
		}
		if i+1 < len(tokens) && !IsFlag(tokens[i+1]) {
			out[name] = Unquote(tokens[i+1])
		} else {
			// Флаг без значения — переключатель, для формы это «включено».
			out[name] = ""
		}
	}
	return out
}

// ApplyArgs правит значения флагов в строке аргументов.
//
// Аргументы, которых нет в changes, остаются нетронутыми и на своих местах:
// клиент мог дописать туда что-то своё, и сохранение настроек не должно это
// вычищать. Отсутствующие флаги дописываются в конец в порядке имён, чтобы
// результат не зависел от обхода карты.
func ApplyArgs(s string, changes map[string]string, prefix string) string {
	if prefix == "" {
		prefix = "-"
	}
	tokens := SplitArgs(s)
	pending := make(map[string]string, len(changes))
	// Исходное написание имени храним отдельно: искать флаг надо без учёта
	// регистра, а дописывать — ровно так, как его назвал профиль. Иначе
	// -SteamServerName у Soulmask превратился бы в -steamservername.
	original := make(map[string]string, len(changes))
	for k, v := range changes {
		lower := strings.ToLower(k)
		pending[lower] = v
		original[lower] = k
	}

	out := make([]string, 0, len(tokens)+2*len(changes))
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if !IsFlag(tok) {
			out = append(out, tok)
			continue
		}
		key := strings.ToLower(strings.TrimLeft(tok, "-+"))
		val, ok := pending[key]
		if !ok {
			out = append(out, tok)
			continue
		}
		delete(pending, key)

		hasValue := i+1 < len(tokens) && !IsFlag(tokens[i+1])
		if val == "" {
			// Пустое значение убирает флаг вместе со старым значением: иначе
			// очистка поля в панели оставляла бы прежнее значение в силе.
			if hasValue {
				i++
			}
			continue
		}
		out = append(out, tok, QuoteArg(val))
		if hasValue {
			i++
		}
	}

	rest := make([]string, 0, len(pending))
	for k := range pending {
		rest = append(rest, k)
	}
	sort.Strings(rest)
	for _, k := range rest {
		if pending[k] == "" {
			continue
		}
		out = append(out, prefix+original[k], QuoteArg(pending[k]))
	}
	return strings.Join(out, " ")
}
