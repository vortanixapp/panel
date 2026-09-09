package gamesettings

import "strings"

func init() { registerCodec(FormatArgs, argsCodec{}) }

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
		if _, seen := out[name]; seen {
			continue
		}
		if i+1 < len(tokens) && !IsFlag(tokens[i+1]) {
			out[name] = Unquote(tokens[i+1])
			continue
		}
		out[name] = ""
	}
	return out
}

func (argsCodec) Apply(content string, changes map[string]string) (string, error) {
	return ApplyArgs(content, changes, "-"), nil
}
