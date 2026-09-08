package gamesettings

import "strings"

func init() { registerCodec(FormatUEQuery, ueQueryCodec{}) }

// ueQueryCodec — строка запуска Unreal-игр вида
//
//	TheIsland?listen?SessionName=Мой сервер?MaxPlayers=70 -server -log
//
// Первый токен — карта и параметры через вопросительный знак, дальше идут
// обычные флаги. Кодек трогает только первую часть: флаги после неё остаются
// как есть, их правит формат аргументов запуска.
//
// Физический ключ — имя параметра. Особый ключ `@map` обозначает саму карту:
// она стоит первой и без имени, но выбирать её из панели нужно так же, как
// остальное.
type ueQueryCodec struct{}

// MapKey — имя, под которым карта Unreal-игры доступна как обычное поле.
const MapKey = "@map"

// splitUEQuery делит строку на часть с картой и хвост из флагов.
//
// Граница — конец ПЕРВОГО аргумента, а не первый пробел: имя сессии сплошь и
// рядом содержит пробел (`SessionName=Мой сервер`), и в строке запуска такой
// параметр берётся в кавычки. Резать по пробелу значило бы разорвать его
// пополам и потерять всё, что идёт следом.
func splitUEQuery(content string) (query, rest string) {
	trimmed := strings.TrimLeft(content, " \t")
	if trimmed == "" {
		return "", ""
	}
	quote := byte(0)
	for i := 0; i < len(trimmed); i++ {
		ch := trimmed[i]
		switch {
		case quote != 0:
			if ch == quote {
				quote = 0
			}
		case ch == '"' || ch == '\'':
			quote = ch
		case ch == ' ' || ch == '\t':
			return Unquote(trimmed[:i]), trimmed[i:]
		}
	}
	return Unquote(trimmed), ""
}

func (ueQueryCodec) Parse(content string) map[string]string {
	query, _ := splitUEQuery(content)
	query = strings.TrimSpace(query)
	if query == "" {
		return map[string]string{}
	}
	out := map[string]string{}
	parts := strings.Split(query, "?")
	if parts[0] != "" {
		out[MapKey] = parts[0]
	}
	for _, p := range parts[1:] {
		if p == "" {
			continue
		}
		key, value := p, ""
		if i := strings.IndexByte(p, '='); i >= 0 {
			key, value = p[:i], p[i+1:]
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, seen := out[key]; seen {
			continue
		}
		out[key] = value
	}
	return out
}

func (ueQueryCodec) Apply(content string, changes map[string]string) (string, error) {
	query, rest := splitUEQuery(content)
	query = strings.TrimSpace(query)

	parts := []string{}
	if query != "" {
		parts = strings.Split(query, "?")
	}
	if len(parts) == 0 {
		parts = []string{""}
	}

	if m, ok := changes[MapKey]; ok {
		parts[0] = m
	}

	pending := make(map[string]string, len(changes))
	for k, v := range changes {
		if k == MapKey {
			continue
		}
		pending[k] = v
	}

	seen := map[string]bool{}
	for i := 1; i < len(parts); i++ {
		p := parts[i]
		key := p
		if j := strings.IndexByte(p, '='); j >= 0 {
			key = p[:j]
		}
		key = strings.TrimSpace(key)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		val, want := pending[key]
		if !want {
			continue
		}
		delete(pending, key)
		if val == "" {
			// Пустое значение убирает параметр целиком: иначе очистка поля в
			// панели оставила бы прежнее значение в силе.
			parts[i] = ""
			continue
		}
		parts[i] = key + "=" + val
	}

	// Порядок дописывания устойчивый: обход карты в Go случаен, и без сортировки
	// строка запуска менялась бы при каждом сохранении без изменения смысла.
	for _, key := range sortedKeys(pending) {
		if pending[key] == "" {
			continue
		}
		parts = append(parts, key+"="+pending[key])
	}

	kept := parts[:0]
	for i, p := range parts {
		if i > 0 && p == "" {
			continue
		}
		kept = append(kept, p)
	}
	// Если внутри есть пробел, строку запроса надо взять в кавычки: иначе она
	// распадётся на несколько аргументов и игра увидит только начало.
	return QuoteArg(strings.Join(kept, "?")) + rest, nil
}
