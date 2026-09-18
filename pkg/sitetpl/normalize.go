package sitetpl

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

const (
	MaxDocumentBytes = 1 << 20
	maxBlocks        = 200
	maxCustomPages   = 100
	maxPages         = 200
	maxMenuItems     = 300
	maxMenuDepth     = 3
	maxRoles         = 20
	maxLocales       = 20
	maxShort         = 300
	maxLong          = 5000
	maxHTML          = 64 << 10
	maxHidden        = 100
	maxTexts         = 5000
	maxTextValue     = 10000
	maxURL           = 2000
)

var (
	idPattern      = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,64}$`)
	slugPattern    = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,58}[a-z0-9])?$`)
	iconPattern    = regexp.MustCompile(`^[a-z0-9-]{1,40}$`)
	localePattern  = regexp.MustCompile(`^[a-z]{2,3}(-[a-z0-9]{2,8})?$`)
	keyPattern     = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,200}$`)
	rolePattern    = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,64}$`)
	pagePattern    = regexp.MustCompile(`^/[A-Za-z0-9_\-\[\]/.]{0,119}$`)
	sectionPattern = regexp.MustCompile(`^[a-z0-9._-]{1,64}$`)
	assetPattern   = regexp.MustCompile(`^branding/[A-Za-z0-9._-]{1,120}$`)
)

var (
	menuNames     = map[string]bool{"site_header": true, "site_footer": true, "user_sidebar": true, AdminMenu: true}
	itemKinds     = map[string]bool{"link": true, "group": true}
	itemAudiences = map[string]bool{"guests": true, "users": true, "staff": true}
	pageAudiences = map[string]bool{"guests": true, "users": true}
	videoHosts    = map[string]bool{
		"youtube.com": true, "www.youtube.com": true, "m.youtube.com": true, "youtu.be": true,
		"youtube-nocookie.com": true, "www.youtube-nocookie.com": true,
		"vk.com": true, "vkvideo.ru": true, "rutube.ru": true,
	}
	serverTextPrefixes = []string{"mail.", "notify."}
)

type Error struct {
	msg string
}

func (e *Error) Error() string {
	return e.msg
}

func fail(format string, args ...any) error {
	return &Error{msg: fmt.Sprintf(format, args...)}
}

const urlRule = "Ссылка «%s» недопустима: подходят https://, http://, mailto:, tel:, /путь и #якорь"

type Options struct {
	Locales map[string]bool
}

func Normalize(doc *Document, opt Options) error {
	if err := normalizeMenus(doc); err != nil {
		return err
	}
	if err := normalizePages(doc); err != nil {
		return err
	}
	if err := normalizeCustomPages(doc); err != nil {
		return err
	}
	return normalizeTexts(doc, opt)
}

func SafeURL(v string) bool {
	if v == "" {
		return true
	}
	if len(v) > maxURL || strings.ContainsAny(v, "\\\x00\r\n\t <>\"'`") {
		return false
	}
	if strings.HasPrefix(v, "#") {
		return len(v) > 1
	}
	if strings.HasPrefix(v, "/") {
		return !strings.HasPrefix(v, "//")
	}
	u, err := url.Parse(v)
	if err != nil {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		return u.Host != "" && u.User == nil
	case "mailto", "tel":
		return u.Opaque != ""
	}
	return false
}

func safeImage(v string) bool {
	if assetPattern.MatchString(v) {
		return !strings.Contains(v, "..")
	}
	if !SafeURL(v) {
		return false
	}
	lower := strings.ToLower(v)
	return strings.HasPrefix(v, "/") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://")
}

func safeVideo(v string) bool {
	if !SafeURL(v) {
		return false
	}
	u, err := url.Parse(v)
	if err != nil || !strings.EqualFold(u.Scheme, "https") {
		return false
	}
	return videoHosts[strings.ToLower(u.Hostname())]
}

func normalizeL(in L, limit int, html bool) (L, error) {
	if len(in) == 0 {
		return nil, nil
	}
	if len(in) > maxLocales {
		return nil, fail("Текст задан больше чем на %d языках", maxLocales)
	}
	out := L{}
	for locale, value := range in {
		if !localePattern.MatchString(locale) {
			return nil, fail("Неизвестный язык «%s»", locale)
		}
		if !html {
			value = strings.TrimSpace(value)
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		if strings.ContainsRune(value, 0) {
			return nil, fail("Текст содержит недопустимые символы")
		}
		if html {
			if len(value) > limit {
				return nil, fail("HTML больше %d КБ", limit>>10)
			}
		} else if utf8.RuneCountInString(value) > limit {
			return nil, fail("Текст «%s…» длиннее %d символов", truncate(value, 40), limit)
		}
		out[locale] = value
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

func normalizeMenus(doc *Document) error {
	out := map[string]Menu{}
	for name, menu := range doc.Menus {
		if !menuNames[name] {
			continue
		}
		count := 0
		items, err := normalizeItems(menu.Items, 1, &count, map[string]bool{})
		if err != nil {
			return err
		}
		if items == nil {
			items = []Item{}
		}
		out[name] = Menu{Items: items}
	}
	if len(out) == 0 {
		out = nil
	}
	doc.Menus = out
	return nil
}

func normalizeItems(items []Item, depth int, count *int, seen map[string]bool) ([]Item, error) {
	if len(items) == 0 {
		return nil, nil
	}
	if depth > maxMenuDepth {
		return nil, fail("Меню вложено глубже трёх уровней")
	}
	out := make([]Item, 0, len(items))
	for _, it := range items {
		*count++
		if *count > maxMenuItems {
			return nil, fail("В меню больше %d пунктов", maxMenuItems)
		}
		if !idPattern.MatchString(it.ID) {
			return nil, fail("У пункта меню некорректный идентификатор")
		}
		if seen[it.ID] {
			return nil, fail("Пункт меню «%s» повторяется", it.ID)
		}
		seen[it.ID] = true
		if it.Ref != "" && !idPattern.MatchString(it.Ref) {
			return nil, fail("У пункта меню некорректная ссылка на стандартный пункт")
		}
		if it.Kind == "" {
			it.Kind = "link"
		}
		if !itemKinds[it.Kind] {
			return nil, fail("Неизвестный вид пункта меню «%s»", it.Kind)
		}
		var err error
		if it.Label, err = normalizeL(it.Label, maxShort, false); err != nil {
			return nil, err
		}
		if it.Badge, err = normalizeL(it.Badge, 40, false); err != nil {
			return nil, err
		}
		it.Icon = strings.TrimSpace(it.Icon)
		if it.Icon != "" && !iconPattern.MatchString(it.Icon) {
			return nil, fail("Неизвестная иконка «%s»", it.Icon)
		}
		it.URL = strings.TrimSpace(it.URL)
		if !SafeURL(it.URL) {
			return nil, fail(urlRule, it.URL)
		}
		if it.Audience == "all" {
			it.Audience = ""
		}
		if it.Audience != "" && !itemAudiences[it.Audience] {
			return nil, fail("Неизвестная видимость пункта меню «%s»", it.Audience)
		}
		if it.Roles, err = normalizeRoles(it.Roles); err != nil {
			return nil, err
		}
		if it.Audience != "staff" {
			it.Roles = nil
		}
		if it.Items, err = normalizeItems(it.Items, depth+1, count, seen); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, nil
}

func normalizeRoles(roles []string) ([]string, error) {
	if len(roles) > maxRoles {
		return nil, fail("Слишком много ролей у пункта меню")
	}
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		role = strings.TrimSpace(role)
		if !rolePattern.MatchString(role) {
			return nil, fail("Некорректная роль «%s»", role)
		}
		if !slices.Contains(out, role) {
			out = append(out, role)
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

type zone int

const (
	zoneSite zone = iota
	zonePanel
	zoneSlot
)

func normalizeBlocks(blocks []Block, z zone, seen map[string]bool) ([]Block, error) {
	if len(blocks) > maxBlocks {
		return nil, fail("На странице больше %d блоков", maxBlocks)
	}
	out := make([]Block, 0, len(blocks))
	used := map[string]bool{}
	for _, b := range blocks {
		if !idPattern.MatchString(b.ID) {
			return nil, fail("У блока некорректный идентификатор")
		}
		if seen[b.ID] {
			return nil, fail("Блок «%s» повторяется на странице", b.ID)
		}
		if builtinSections[b.Type] {
			if z != zoneSite || used[b.Type] {
				continue
			}
			used[b.Type] = true
			seen[b.ID] = true
			out = append(out, Block{ID: b.ID, Type: b.Type, Hidden: b.Hidden})
			continue
		}
		schema, ok := blockSchemas[b.Type]
		if !ok {
			continue
		}
		props, err := normalizeProps(b.Props, schema)
		if err != nil {
			return nil, err
		}
		seen[b.ID] = true
		out = append(out, Block{ID: b.ID, Type: b.Type, Hidden: b.Hidden, Props: props})
	}
	return out, nil
}

func normalizeProps(in map[string]any, schema map[string]field) (map[string]any, error) {
	out := map[string]any{}
	for name, f := range schema {
		raw, ok := in[name]
		if !ok || raw == nil {
			continue
		}
		v, keep, err := normalizeValue(raw, f)
		if err != nil {
			return nil, err
		}
		if keep {
			out[name] = v
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func normalizeValue(raw any, f field) (any, bool, error) {
	switch f.kind {
	case kindText, kindLong, kindHTML:
		obj, ok := raw.(map[string]any)
		if !ok {
			return nil, false, nil
		}
		in := L{}
		for locale, v := range obj {
			if s, ok := v.(string); ok {
				in[locale] = s
			}
		}
		limit, html := maxShort, false
		switch f.kind {
		case kindLong:
			limit = maxLong
		case kindHTML:
			limit, html = maxHTML, true
		}
		l, err := normalizeL(in, limit, html)
		if err != nil {
			return nil, false, err
		}
		return l, l != nil, nil
	case kindURL, kindImage, kindVideo, kindIcon:
		s, ok := raw.(string)
		if !ok {
			return nil, false, nil
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return nil, false, nil
		}
		switch f.kind {
		case kindURL:
			if !SafeURL(s) {
				return nil, false, fail(urlRule, s)
			}
		case kindImage:
			if !safeImage(s) {
				return nil, false, fail("Адрес картинки «%s» недопустим", s)
			}
		case kindVideo:
			if !safeVideo(s) {
				return nil, false, fail("Видео подключается только с YouTube, VK и Rutube по https")
			}
		case kindIcon:
			if !iconPattern.MatchString(s) {
				return nil, false, fail("Неизвестная иконка «%s»", s)
			}
		}
		return s, true, nil
	case kindBool:
		b, ok := raw.(bool)
		return b, ok && b, nil
	case kindEnum:
		s, ok := raw.(string)
		if !ok || !slices.Contains(f.values, s) {
			return nil, false, nil
		}
		return s, true, nil
	case kindInt:
		n, ok := raw.(float64)
		if !ok || n != math.Trunc(n) || n < float64(f.min) || n > float64(f.max) {
			return nil, false, nil
		}
		return int(n), true, nil
	case kindList:
		arr, ok := raw.([]any)
		if !ok {
			return nil, false, nil
		}
		if len(arr) > f.max {
			return nil, false, fail("В списке больше %d элементов", f.max)
		}
		items := make([]map[string]any, 0, len(arr))
		for _, el := range arr {
			obj, ok := el.(map[string]any)
			if !ok {
				continue
			}
			props, err := normalizeProps(obj, f.item)
			if err != nil {
				return nil, false, err
			}
			if props == nil {
				props = map[string]any{}
			}
			items = append(items, props)
		}
		return items, len(items) > 0, nil
	}
	return nil, false, nil
}

func normalizeHidden(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if sectionPattern.MatchString(id) && !slices.Contains(out, id) && len(out) < maxHidden {
			out = append(out, id)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizePages(doc *Document) error {
	if len(doc.Pages) > maxPages {
		return fail("Настроено больше %d страниц", maxPages)
	}
	out := map[string]Page{}
	for key, page := range doc.Pages {
		if !pagePattern.MatchString(key) || strings.Contains(key, "..") {
			continue
		}
		seen := map[string]bool{}
		var next Page
		if key == "/" {
			if page.Blocks != nil {
				blocks, err := normalizeBlocks(*page.Blocks, zoneSite, seen)
				if err != nil {
					return err
				}
				next.Blocks = &blocks
			}
		} else {
			top, err := normalizeBlocks(page.Top, zoneSlot, seen)
			if err != nil {
				return err
			}
			bottom, err := normalizeBlocks(page.Bottom, zoneSlot, seen)
			if err != nil {
				return err
			}
			if len(top) > 0 {
				next.Top = top
			}
			if len(bottom) > 0 {
				next.Bottom = bottom
			}
			next.Hidden = normalizeHidden(page.Hidden)
		}
		if next.Blocks == nil && next.Top == nil && next.Bottom == nil && next.Hidden == nil {
			continue
		}
		out[key] = next
	}
	if len(out) == 0 {
		out = nil
	}
	doc.Pages = out
	return nil
}

func normalizeCustomPages(doc *Document) error {
	if len(doc.CustomPages) > maxCustomPages {
		return fail("Своих страниц больше %d", maxCustomPages)
	}
	ids := map[string]bool{}
	slugs := map[string]bool{}
	out := make([]CustomPage, 0, len(doc.CustomPages))
	for _, p := range doc.CustomPages {
		if !idPattern.MatchString(p.ID) || ids[p.ID] {
			return fail("У своей страницы некорректный или повторяющийся идентификатор")
		}
		ids[p.ID] = true
		p.Slug = strings.ToLower(strings.TrimSpace(p.Slug))
		if !slugPattern.MatchString(p.Slug) {
			return fail("Адрес страницы «%s»: латиница в нижнем регистре, цифры и дефис, до 60 символов", p.Slug)
		}
		if slugs[p.Slug] {
			return fail("Адрес /p/%s уже занят другой страницей", p.Slug)
		}
		slugs[p.Slug] = true
		var err error
		if p.Title, err = normalizeL(p.Title, maxShort, false); err != nil {
			return err
		}
		if p.Description, err = normalizeL(p.Description, maxShort, false); err != nil {
			return err
		}
		if p.Layout != "panel" {
			p.Layout = "site"
		}
		if p.Audience == "all" || !pageAudiences[p.Audience] {
			p.Audience = ""
		}
		z := zoneSite
		if p.Layout == "panel" {
			z = zonePanel
			p.Audience = "users"
		}
		blocks, err := normalizeBlocks(p.Blocks, z, map[string]bool{})
		if err != nil {
			return err
		}
		p.Blocks = blocks
		out = append(out, p)
	}
	if len(out) == 0 {
		out = nil
	}
	doc.CustomPages = out
	return nil
}

func normalizeTexts(doc *Document, opt Options) error {
	out := map[string]map[string]string{}
	for locale, keys := range doc.Texts {
		if !localePattern.MatchString(locale) || (opt.Locales != nil && !opt.Locales[locale]) {
			continue
		}
		if len(keys) > maxTexts {
			return fail("Изменённых фраз больше %d", maxTexts)
		}
		clean := map[string]string{}
		for key, value := range keys {
			if !keyPattern.MatchString(key) || serverText(key) {
				return fail("Фразу «%s» нельзя менять в редакторе шаблона", key)
			}
			if strings.ContainsRune(value, 0) {
				return fail("Фраза %s содержит недопустимые символы", key)
			}
			if utf8.RuneCountInString(value) > maxTextValue {
				return fail("Фраза %s длиннее %d символов", key, maxTextValue)
			}
			clean[key] = value
		}
		if len(clean) > 0 {
			out[locale] = clean
		}
	}
	if len(out) == 0 {
		out = nil
	}
	doc.Texts = out
	return nil
}

func serverText(key string) bool {
	for _, prefix := range serverTextPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}
	return false
}
