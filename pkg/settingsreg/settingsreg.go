package settingsreg

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Kind string

const (
	KindInt     Kind = "int"
	KindBool    Kind = "bool"
	KindString  Kind = "string"
	KindEnum    Kind = "enum"
	KindIntList Kind = "intlist"
	KindMulti   Kind = "multi"
)

const (
	UnitNone    = ""
	UnitMs      = "ms"
	UnitSec     = "sec"
	UnitMin     = "min"
	UnitHour    = "hour"
	UnitDay     = "day"
	UnitMB      = "mb"
	UnitPercent = "percent"
	UnitCount   = "count"
)

const (
	cacheTTL      = 5 * time.Second
	retryAfter    = time.Second
	loadTimeout   = 3 * time.Second
	maxListItems  = 24
	maxStringSize = 500
)

type Setting struct {
	Key     string
	Group   string
	Section string
	Kind    Kind
	Default string
	Min     int64
	Max     int64
	Unit    string
	Options []string
	Public  bool
	Agent   bool
}

type Loader func(ctx context.Context, keys []string) (map[string]string, error)

var (
	registry []*Setting
	byKey    = map[string]*Setting{}
)

func def(s Setting) *Setting {
	if _, dup := byKey[s.Key]; dup {
		panic("settingsreg: duplicate key " + s.Key)
	}
	p := &s
	registry = append(registry, p)
	byKey[s.Key] = p
	return p
}

func All() []*Setting {
	out := make([]*Setting, len(registry))
	copy(out, registry)
	return out
}

func Lookup(key string) (*Setting, bool) {
	s, ok := byKey[key]
	return s, ok
}

type snapshot struct {
	mu     sync.Mutex
	loader Loader
	values map[string]string
	loaded time.Time
}

var store = &snapshot{}

func Init(loader Loader) {
	store.mu.Lock()
	store.loader = loader
	store.loaded = time.Time{}
	store.mu.Unlock()
}

func SetStatic(values map[string]string) {
	store.mu.Lock()
	store.loader = nil
	store.values = make(map[string]string, len(values))
	for k, v := range values {
		store.values[k] = v
	}
	store.loaded = time.Now()
	store.mu.Unlock()
}

func SetStaticTyped(values map[string]any) {
	flat := make(map[string]string, len(values))
	for key, value := range values {
		switch v := value.(type) {
		case string:
			flat[key] = v
		case bool:
			if v {
				flat[key] = "1"
			} else {
				flat[key] = "0"
			}
		case float64:
			flat[key] = strconv.FormatInt(int64(v), 10)
		case int:
			flat[key] = strconv.Itoa(v)
		case int64:
			flat[key] = strconv.FormatInt(v, 10)
		case []any:
			parts := make([]string, 0, len(v))
			for _, item := range v {
				if n, ok := item.(float64); ok {
					parts = append(parts, strconv.FormatInt(int64(n), 10))
				}
			}
			flat[key] = strings.Join(parts, ",")
		}
	}
	SetStatic(flat)
}

func Invalidate() {
	store.mu.Lock()
	store.loaded = time.Time{}
	store.mu.Unlock()
}

func (st *snapshot) get(key string) string {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.loader != nil && time.Since(st.loaded) >= cacheTTL {
		st.refreshLocked()
	}
	return st.values[key]
}

func (st *snapshot) all() map[string]string {
	st.mu.Lock()
	defer st.mu.Unlock()
	out := map[string]string{}
	if st.loader != nil && time.Since(st.loaded) >= cacheTTL {
		st.refreshLocked()
	}
	for k, v := range st.values {
		out[k] = v
	}
	return out
}

func (st *snapshot) refreshLocked() {
	ctx, cancel := context.WithTimeout(context.Background(), loadTimeout)
	defer cancel()
	keys := make([]string, 0, len(registry))
	for _, s := range registry {
		keys = append(keys, s.Key)
	}
	next, err := st.loader(ctx, keys)
	if err != nil {
		st.loaded = time.Now().Add(retryAfter - cacheTTL)
		return
	}
	st.values = next
	st.loaded = time.Now()
}

func RawToString(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var n json.Number
	if json.Unmarshal(raw, &n) == nil {
		return n.String()
	}
	var b bool
	if json.Unmarshal(raw, &b) == nil {
		if b {
			return "1"
		}
		return "0"
	}
	return strings.Trim(string(raw), `"`)
}

func (s *Setting) stored() string {
	return strings.TrimSpace(store.get(s.Key))
}

func (s *Setting) effective(stored string) string {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return s.Default
	}
	norm, err := s.Normalize(stored)
	if err != nil {
		return s.Default
	}
	return norm
}

func (s *Setting) Raw() string {
	return s.effective(s.stored())
}

func (s *Setting) Int() int64 {
	n, err := strconv.ParseInt(s.Raw(), 10, 64)
	if err != nil {
		n, _ = strconv.ParseInt(s.Default, 10, 64)
	}
	return n
}

func (s *Setting) Bool() bool {
	return s.Raw() == "1"
}

func (s *Setting) Str() string {
	return s.Raw()
}

func (s *Setting) Ints() []int64 {
	return parseList(s.Raw())
}

func (s *Setting) Duration() time.Duration {
	return time.Duration(s.Int()) * unitDuration(s.Unit)
}

func unitDuration(unit string) time.Duration {
	switch unit {
	case UnitMs:
		return time.Millisecond
	case UnitSec:
		return time.Second
	case UnitMin:
		return time.Minute
	case UnitHour:
		return time.Hour
	case UnitDay:
		return 24 * time.Hour
	}
	return 1
}

func (s *Setting) Bytes() int64 {
	return s.Int() << 20
}

func parseList(raw string) []int64 {
	out := []int64{}
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' || r == ';' }) {
		if n, err := strconv.ParseInt(part, 10, 64); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func (s *Setting) Normalize(input string) (string, error) {
	input = strings.TrimSpace(input)
	switch s.Kind {
	case KindInt:
		n, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return "", fmt.Errorf("значение должно быть числом")
		}
		if n < s.Min || n > s.Max {
			return "", fmt.Errorf("допустимо от %d до %d", s.Min, s.Max)
		}
		return strconv.FormatInt(n, 10), nil
	case KindBool:
		switch strings.ToLower(input) {
		case "1", "true", "on", "yes":
			return "1", nil
		case "0", "false", "off", "no":
			return "0", nil
		}
		return "", fmt.Errorf("ожидается да или нет")
	case KindEnum:
		for _, o := range s.Options {
			if o == input {
				return input, nil
			}
		}
		return "", fmt.Errorf("недопустимый вариант")
	case KindMulti:
		chosen := map[string]bool{}
		for _, part := range strings.FieldsFunc(input, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
			part = strings.TrimSpace(part)
			if !slices.Contains(s.Options, part) {
				return "", fmt.Errorf("недопустимый вариант")
			}
			chosen[part] = true
		}
		out := make([]string, 0, len(chosen))
		for _, o := range s.Options {
			if chosen[o] {
				out = append(out, o)
			}
		}
		if len(out) == 0 {
			return "", fmt.Errorf("выберите хотя бы один вариант")
		}
		return strings.Join(out, ","), nil
	case KindString:
		limit := int(s.Max)
		if limit <= 0 || limit > maxStringSize {
			limit = maxStringSize
		}
		if len([]rune(input)) > limit {
			return "", fmt.Errorf("не длиннее %d символов", limit)
		}
		return input, nil
	case KindIntList:
		list := parseList(input)
		if len(list) == 0 || len(list) > maxListItems {
			return "", fmt.Errorf("нужен список чисел через запятую, не более %d", maxListItems)
		}
		parts := make([]string, 0, len(list))
		for _, n := range list {
			if n < s.Min || n > s.Max {
				return "", fmt.Errorf("каждое значение от %d до %d", s.Min, s.Max)
			}
			parts = append(parts, strconv.FormatInt(n, 10))
		}
		return strings.Join(parts, ","), nil
	}
	return "", fmt.Errorf("неизвестный тип")
}

func Effective() map[string]string {
	stored := store.all()
	out := make(map[string]string, len(registry))
	for _, s := range registry {
		out[s.Key] = s.effective(stored[s.Key])
	}
	return out
}

func Stored() map[string]string {
	return store.all()
}

func (s *Setting) Typed(raw string) any {
	switch s.Kind {
	case KindInt:
		n, _ := strconv.ParseInt(raw, 10, 64)
		return n
	case KindBool:
		return raw == "1"
	case KindIntList:
		return parseList(raw)
	case KindMulti:
		parts := []string{}
		for _, part := range strings.Split(raw, ",") {
			if part = strings.TrimSpace(part); part != "" {
				parts = append(parts, part)
			}
		}
		return parts
	}
	return raw
}

func PublicValues() map[string]any {
	eff := Effective()
	out := map[string]any{}
	for _, s := range registry {
		if s.Public {
			out[s.Key] = s.Typed(eff[s.Key])
		}
	}
	return out
}

func AgentValues() map[string]any {
	eff := Effective()
	out := map[string]any{}
	for _, s := range registry {
		if s.Agent {
			out[s.Key] = s.Typed(eff[s.Key])
		}
	}
	return out
}

func Validate(next map[string]string) error {
	eff := Effective()
	for k, v := range next {
		eff[k] = v
	}
	for _, c := range crossChecks {
		if err := c(eff); err != nil {
			return err
		}
	}
	return nil
}

var crossChecks []func(map[string]string) error

func addCheck(fn func(map[string]string) error) {
	crossChecks = append(crossChecks, fn)
}
