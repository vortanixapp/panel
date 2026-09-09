package gamesettings

import (
	"fmt"
	"sort"
	"strings"
)

func sortStrings(s []string) { sort.Strings(s) }

type Codec interface {
	Parse(content string) map[string]string

	Apply(content string, changes map[string]string) (string, error)
}

var codecs = map[Format]Codec{}

func registerCodec(f Format, c Codec) {
	if _, busy := codecs[f]; busy {
		panic("gamesettings: кодек формата " + string(f) + " уже зарегистрирован")
	}
	codecs[f] = c
}

func CodecFor(f Format) (Codec, bool) {
	c, ok := codecs[f]
	return c, ok
}

type ErrUnsupportedFormat struct{ Format Format }

func (e *ErrUnsupportedFormat) Error() string {
	return "неизвестный формат конфигурации: " + string(e.Format)
}

func ParseFile(f ConfigFile, content string) (map[string]string, error) {
	c, ok := CodecFor(f.Format)
	if !ok {
		return nil, &ErrUnsupportedFormat{Format: f.Format}
	}
	return c.Parse(content), nil
}

func ApplyFile(f ConfigFile, content string, changes map[string]string) (string, error) {
	c, ok := CodecFor(f.Format)
	if !ok {
		return "", &ErrUnsupportedFormat{Format: f.Format}
	}
	if len(changes) == 0 {
		return content, nil
	}
	if strings.TrimSpace(content) == "" && f.Template != "" {
		content = f.Template
	}
	out, err := c.Apply(content, changes)
	if err != nil {
		return "", fmt.Errorf("%s: %w", f.Path, err)
	}
	return out, nil
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}

func lineEnding(s string) string {
	if strings.Contains(s, "\r\n") {
		return "\r\n"
	}
	return "\n"
}

func joinLines(lines []string, ending string) string {
	return strings.Join(lines, ending)
}

func isCommentLine(line string, markers ...string) bool {
	t := strings.TrimSpace(line)
	if t == "" {
		return true
	}
	for _, m := range markers {
		if strings.HasPrefix(t, m) {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sortStrings(out)
	return out
}
