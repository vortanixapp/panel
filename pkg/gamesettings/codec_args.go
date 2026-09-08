package gamesettings

import "strings"

func init() { registerCodec(FormatArgs, argsCodec{}) }

// argsCodec — обёртка над разбором аргументов запуска под общий интерфейс.
//
// Сама механика лежит в format_args.go: там квотирование, разбор состоянием и
// сборка обратно. Здесь только приведение к виду «прочитать всё / записать
// изменения», как у файловых форматов.
//
// Физический ключ — имя флага без ведущего знака: `name` найдёт и `-name`,
// и `+name`.
type argsCodec struct{}

func (argsCodec) Parse(content string) map[string]string {
	out := map[string]string{}
	tokens := SplitArgs(content)
	for i, tok := range tokens {
		if !IsFlag(tok) {
			continue
		}
		name := strings.TrimLeft(tok, "-+")
		if name == "" {
			continue
		}
		// Побеждает первое вхождение: именно его увидит игра, разбирая
		// командную строку слева направо.
		if _, seen := out[name]; seen {
			continue
		}
		if i+1 < len(tokens) && !IsFlag(tokens[i+1]) {
			out[name] = Unquote(tokens[i+1])
			continue
		}
		// Флаг без значения — переключатель, для формы это «включено».
		out[name] = ""
	}
	return out
}

func (argsCodec) Apply(content string, changes map[string]string) (string, error) {
	return ApplyArgs(content, changes, "-"), nil
}
