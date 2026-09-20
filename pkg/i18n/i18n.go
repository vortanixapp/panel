package i18n

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	DefaultLocale = "ru"
	cacheTTL      = 30 * time.Second
)

var (
	serverPrefixes = []string{"mail.", "notify."}
	placeholder    = regexp.MustCompile(`\{(\w+)\}`)
	catalogs       = map[string]map[string]string{
		"ru": catalogRU,
		"en": catalogEN,
	}
)

type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Params map[string]any

type Msg struct {
	Key    string
	Params Params
	Raw    string
}

func Key(key string, params ...Params) Msg {
	m := Msg{Key: key}
	if len(params) > 0 {
		m.Params = params[0]
	}
	return m
}

func Raw(text string) Msg {
	return Msg{Raw: text}
}

type language struct {
	base    string
	enabled bool
}

type snapshot struct {
	loadedAt      time.Time
	languages     map[string]language
	defaultLocale string
	overrides     map[string]map[string]string
}

var (
	mu      sync.Mutex
	current *snapshot
)

func IsServerKey(key string) bool {
	for _, prefix := range serverPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}

func Catalog() map[string]map[string]string {
	out := make(map[string]map[string]string, len(catalogs))
	for base, messages := range catalogs {
		copied := make(map[string]string, len(messages))
		for k, v := range messages {
			copied[k] = v
		}
		out[base] = copied
	}
	return out
}

func Invalidate() {
	mu.Lock()
	defer mu.Unlock()
	if current != nil {
		current.loadedAt = time.Time{}
	}
}

func normalize(code string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(code)), "_", "-")
}

func newSnapshot() *snapshot {
	return &snapshot{
		loadedAt: time.Now(),
		languages: map[string]language{
			"ru": {base: "ru", enabled: true},
			"en": {base: "en", enabled: true},
		},
		defaultLocale: DefaultLocale,
		overrides:     map[string]map[string]string{},
	}
}

func load(ctx context.Context, db DB) *snapshot {
	mu.Lock()
	defer mu.Unlock()
	if current != nil && time.Since(current.loadedAt) < cacheTTL {
		return current
	}
	next := newSnapshot()
	if db == nil {
		return next
	}
	if err := next.fill(ctx, db); err != nil && current != nil {
		current.loadedAt = time.Now()
		return current
	}
	current = next
	return next
}

func (s *snapshot) fill(ctx context.Context, db DB) error {
	rows, err := db.Query(ctx, `SELECT code, base, enabled FROM core.languages`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var code, base string
		var enabled bool
		if rows.Scan(&code, &base, &enabled) == nil {
			s.languages[code] = language{base: base, enabled: enabled}
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	var stored string
	err = db.QueryRow(ctx, `
		SELECT COALESCE(value #>> '{}', '') FROM core.tenant_settings WHERE key = 'i18n.default_locale'
	`).Scan(&stored)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	s.defaultLocale = s.pickDefault(normalize(stored))

	rows, err = db.Query(ctx, `
		SELECT locale, key, value FROM core.translation_keys
		WHERE (key LIKE 'mail.%' OR key LIKE 'notify.%') AND btrim(value) <> ''
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var locale, key, value string
		if rows.Scan(&locale, &key, &value) != nil {
			continue
		}
		if s.overrides[locale] == nil {
			s.overrides[locale] = map[string]string{}
		}
		s.overrides[locale][key] = value
	}
	return rows.Err()
}

func (s *snapshot) pickDefault(stored string) string {
	if l, ok := s.languages[stored]; ok && l.enabled {
		return stored
	}
	if l, ok := s.languages[DefaultLocale]; ok && l.enabled {
		return DefaultLocale
	}
	codes := make([]string, 0, len(s.languages))
	for code, l := range s.languages {
		if l.enabled {
			codes = append(codes, code)
		}
	}
	sort.Strings(codes)
	if len(codes) > 0 {
		return codes[0]
	}
	return DefaultLocale
}

func (s *snapshot) localizer(locale string) Localizer {
	code := normalize(locale)
	if l, ok := s.languages[code]; !ok || !l.enabled {
		code = s.defaultLocale
	}
	base := DefaultLocale
	if l, ok := s.languages[code]; ok {
		if _, known := catalogs[l.base]; known {
			base = l.base
		}
	}
	return Localizer{locale: code, base: base, overrides: s.overrides[code]}
}

type Localizer struct {
	locale    string
	base      string
	overrides map[string]string
}

func For(ctx context.Context, db DB, locale string) Localizer {
	return load(ctx, db).localizer(locale)
}

func ForUser(ctx context.Context, db DB, userID string) Localizer {
	var locale string
	if userID != "" && db != nil {
		_ = db.QueryRow(ctx, `
			SELECT COALESCE(locale, '') FROM core.user_profiles WHERE user_id = $1
		`, userID).Scan(&locale)
	}
	return For(ctx, db, locale)
}

func (l Localizer) Locale() string {
	return l.locale
}

func (l Localizer) T(key string, params ...Params) string {
	return l.Text(Key(key, params...))
}

func (l Localizer) Text(m Msg) string {
	if m.Key == "" {
		return m.Raw
	}
	text, ok := l.overrides[m.Key]
	if !ok || strings.TrimSpace(text) == "" {
		if text, ok = catalogs[l.base][m.Key]; !ok {
			if text, ok = catalogs[DefaultLocale][m.Key]; !ok {
				return m.Key
			}
		}
	}
	if len(m.Params) == 0 {
		return text
	}
	return placeholder.ReplaceAllStringFunc(text, func(match string) string {
		value, ok := m.Params[match[1:len(match)-1]]
		if !ok {
			return match
		}
		return l.param(value)
	})
}

func (l Localizer) Paragraphs(parts ...Msg) string {
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if text := strings.TrimSpace(l.Text(part)); text != "" {
			out = append(out, text)
		}
	}
	return strings.Join(out, "\n\n")
}

func (l Localizer) param(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case Msg:
		return l.Text(v)
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func Format(text string, params Params) string {
	if len(params) == 0 {
		return text
	}
	return placeholder.ReplaceAllStringFunc(text, func(match string) string {
		value, ok := params[match[1:len(match)-1]]
		if !ok {
			return match
		}
		switch v := value.(type) {
		case string:
			return v
		case nil:
			return ""
		default:
			return fmt.Sprint(v)
		}
	})
}

func Placeholders(text string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, match := range placeholder.FindAllStringSubmatch(text, -1) {
		if name := match[1]; !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}
