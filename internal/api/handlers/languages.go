package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
)

const (
	defaultLocaleSetting  = "i18n.default_locale"
	languageNameLimit     = 64
	translationValueLimit = 10000
	translationBatchLimit = 20000
	translationBodyLimit  = 16 << 20
)

var (
	languageCodePattern   = regexp.MustCompile(`^[a-z]{2,3}(-[a-z0-9]{2,8})?$`)
	translationKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,200}$`)
	builtinLanguages      = []language{
		{Code: "ru", Name: "Русский", Base: "ru", Enabled: true, Builtin: true},
		{Code: "en", Name: "English", Base: "en", Enabled: true, Builtin: true},
	}
)

type language struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	Base       string `json:"base"`
	Enabled    bool   `json:"enabled"`
	Builtin    bool   `json:"builtin"`
	Translated int    `json:"translated"`
}

type publicLanguage struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Base string `json:"base"`
}

type languageInput struct {
	Code    string  `json:"code"`
	Name    *string `json:"name"`
	Base    *string `json:"base"`
	Enabled *bool   `json:"enabled"`
	Default bool    `json:"default"`
}

func isBuiltinLanguage(code string) bool {
	return code == "ru" || code == "en"
}

func normalizeLanguageCode(v string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(v)), "_", "-")
}

func findLanguage(list []language, code string) (language, bool) {
	for _, l := range list {
		if l.Code == code {
			return l, true
		}
	}
	return language{}, false
}

func defaultLanguage(list []language, stored string) string {
	if l, ok := findLanguage(list, stored); ok && l.Enabled {
		return l.Code
	}
	if l, ok := findLanguage(list, "ru"); ok && l.Enabled {
		return l.Code
	}
	for _, l := range list {
		if l.Enabled {
			return l.Code
		}
	}
	return "ru"
}

func languageNameError(name string) string {
	if name == "" {
		return "укажите название языка"
	}
	if utf8.RuneCountInString(name) > languageNameLimit {
		return "название языка не длиннее 64 символов"
	}
	return ""
}

func (h *Handler) loadLanguages(ctx context.Context, withCounts bool) []language {
	query := `SELECT code, name, base, enabled, 0 FROM core.languages ORDER BY created_at, code`
	if withCounts {
		query = `
			SELECT l.code, l.name, l.base, l.enabled,
			       (SELECT count(*) FROM core.translation_keys t WHERE t.locale = l.code AND btrim(t.value) <> '')
			FROM core.languages l
			ORDER BY l.created_at, l.code`
	}
	list := []language{}
	rows, err := h.dbOf(ctx).Query(ctx, query)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var l language
			if rows.Scan(&l.Code, &l.Name, &l.Base, &l.Enabled, &l.Translated) == nil {
				l.Builtin = isBuiltinLanguage(l.Code)
				list = append(list, l)
			}
		}
	}
	for _, b := range builtinLanguages {
		if _, ok := findLanguage(list, b.Code); !ok {
			list = append(list, b)
		}
	}
	return list
}

func (h *Handler) loadTranslations(ctx context.Context, code string) map[string]string {
	out := map[string]string{}
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT key, value FROM core.translation_keys
		WHERE locale = $1 AND btrim(value) <> ''
	`, code)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if rows.Scan(&k, &v) == nil {
			out[k] = v
		}
	}
	return out
}

func (h *Handler) PublicI18n(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	list := h.loadLanguages(ctx, false)
	def := defaultLanguage(list, h.tenantSettingString(ctx, defaultLocaleSetting))
	requested := normalizeLanguageCode(r.URL.Query().Get("locale"))
	current := def
	enabled := []publicLanguage{}
	for _, l := range list {
		if !l.Enabled {
			continue
		}
		enabled = append(enabled, publicLanguage{Code: l.Code, Name: l.Name, Base: l.Base})
		if l.Code == requested {
			current = l.Code
		}
	}
	base := "ru"
	if l, ok := findLanguage(list, current); ok {
		base = l.Base
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"default_locale": def,
		"languages":      enabled,
		"locale":         current,
		"base":           base,
		"messages":       h.loadTranslations(ctx, current),
	})
}

func (h *Handler) writeAdminLanguages(ctx context.Context, w http.ResponseWriter, status int) {
	list := h.loadLanguages(ctx, true)
	writeJSON(w, status, map[string]any{
		"default_locale": defaultLanguage(list, h.tenantSettingString(ctx, defaultLocaleSetting)),
		"languages":      list,
	})
}

func (h *Handler) AdminLanguages(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		return
	}
	h.writeAdminLanguages(r.Context(), w, http.StatusOK)
}

func (h *Handler) AdminLanguageCreate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	var body languageInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	code := normalizeLanguageCode(body.Code)
	if !languageCodePattern.MatchString(code) {
		writeError(w, http.StatusUnprocessableEntity, "код языка — две-три латинские буквы, можно с регионом: uk, kk, pt-br")
		return
	}
	if isBuiltinLanguage(code) {
		writeError(w, http.StatusConflict, "язык с таким кодом уже есть")
		return
	}
	name := ""
	if body.Name != nil {
		name = strings.TrimSpace(*body.Name)
	}
	if msg := languageNameError(name); msg != "" {
		writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}
	base := ""
	if body.Base != nil {
		base = *body.Base
	}
	if !isBuiltinLanguage(base) {
		writeError(w, http.StatusUnprocessableEntity, "недостающие фразы можно брать только из русского или английского")
		return
	}
	enabled := body.Enabled == nil || *body.Enabled
	tag, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.languages (code, name, base, enabled) VALUES ($1, $2, $3, $4)
		ON CONFLICT (code) DO NOTHING
	`, code, name, base, enabled)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось добавить язык")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusConflict, "язык с таким кодом уже есть")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "language.create", "language:"+code, map[string]any{
		"name": name,
		"base": base,
	})
	h.writeAdminLanguages(ctx, w, http.StatusCreated)
}

func (h *Handler) AdminLanguageUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	code := normalizeLanguageCode(chi.URLParam(r, "code"))
	list := h.loadLanguages(ctx, false)
	lang, found := findLanguage(list, code)
	if !found {
		writeError(w, http.StatusNotFound, "язык не найден")
		return
	}
	var body languageInput
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.Name != nil {
		lang.Name = strings.TrimSpace(*body.Name)
		if msg := languageNameError(lang.Name); msg != "" {
			writeError(w, http.StatusUnprocessableEntity, msg)
			return
		}
	}
	if body.Base != nil && *body.Base != lang.Base {
		if lang.Builtin {
			writeError(w, http.StatusUnprocessableEntity, "у встроенного языка нельзя сменить основу")
			return
		}
		if !isBuiltinLanguage(*body.Base) {
			writeError(w, http.StatusUnprocessableEntity, "недостающие фразы можно брать только из русского или английского")
			return
		}
		lang.Base = *body.Base
	}
	if body.Enabled != nil {
		lang.Enabled = *body.Enabled
	}
	isDefault := body.Default || defaultLanguage(list, h.tenantSettingString(ctx, defaultLocaleSetting)) == code
	if isDefault && !lang.Enabled {
		writeError(w, http.StatusUnprocessableEntity, "язык по умолчанию нельзя выключить — сначала назначьте другой")
		return
	}
	if _, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.languages (code, name, base, enabled) VALUES ($1, $2, $3, $4)
		ON CONFLICT (code) DO UPDATE
		SET name = EXCLUDED.name, base = EXCLUDED.base, enabled = EXCLUDED.enabled, updated_at = now()
	`, lang.Code, lang.Name, lang.Base, lang.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить язык")
		return
	}
	if body.Default {
		h.setTenantSettingString(ctx, defaultLocaleSetting, code)
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "language.update", "language:"+code, map[string]any{
		"name":    lang.Name,
		"base":    lang.Base,
		"enabled": lang.Enabled,
		"default": body.Default,
	})
	h.writeAdminLanguages(ctx, w, http.StatusOK)
}

func (h *Handler) AdminLanguageDelete(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	code := normalizeLanguageCode(chi.URLParam(r, "code"))
	if isBuiltinLanguage(code) {
		writeError(w, http.StatusUnprocessableEntity, "встроенный язык можно только выключить")
		return
	}
	list := h.loadLanguages(ctx, false)
	if _, found := findLanguage(list, code); !found {
		writeError(w, http.StatusNotFound, "язык не найден")
		return
	}
	def := defaultLanguage(list, h.tenantSettingString(ctx, defaultLocaleSetting))
	if def == code {
		writeError(w, http.StatusUnprocessableEntity, "это язык по умолчанию — сначала назначьте другой")
		return
	}
	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось удалить язык")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	steps := []struct {
		sql  string
		args []any
	}{
		{`DELETE FROM core.translation_keys WHERE locale = $1`, []any{code}},
		{`UPDATE core.user_profiles SET locale = $2 WHERE locale = $1`, []any{code, def}},
		{`DELETE FROM core.languages WHERE code = $1`, []any{code}},
	}
	for _, step := range steps {
		if _, err := tx.Exec(ctx, step.sql, step.args...); err != nil {
			writeError(w, http.StatusInternalServerError, "не удалось удалить язык")
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось удалить язык")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "language.delete", "language:"+code, nil)
	h.writeAdminLanguages(ctx, w, http.StatusOK)
}

func (h *Handler) AdminLanguageMessages(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		return
	}
	ctx := r.Context()
	code := normalizeLanguageCode(chi.URLParam(r, "code"))
	if _, found := findLanguage(h.loadLanguages(ctx, false), code); !found {
		writeError(w, http.StatusNotFound, "язык не найден")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"code":     code,
		"messages": h.loadTranslations(ctx, code),
	})
}

func (h *Handler) AdminLanguageMessagesSave(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	code := normalizeLanguageCode(chi.URLParam(r, "code"))
	if _, found := findLanguage(h.loadLanguages(ctx, false), code); !found {
		writeError(w, http.StatusNotFound, "язык не найден")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, translationBodyLimit)
	var body struct {
		Messages map[string]string `json:"messages"`
		Replace  bool              `json:"replace"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "фраз слишком много для одного сохранения")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(body.Messages) > translationBatchLimit {
		writeError(w, http.StatusUnprocessableEntity, "за один раз можно сохранить не больше 20000 фраз")
		return
	}

	keys := make([]string, 0, len(body.Messages))
	values := make([]string, 0, len(body.Messages))
	cleared := []string{}
	for key, value := range body.Messages {
		if !translationKeyPattern.MatchString(key) {
			writeError(w, http.StatusUnprocessableEntity, "в ключах фраз допустимы только латиница, цифры, точка, дефис и подчёркивание")
			return
		}
		if strings.ContainsRune(value, 0) {
			writeError(w, http.StatusUnprocessableEntity, "фраза "+key+" содержит недопустимые символы")
			return
		}
		if utf8.RuneCountInString(value) > translationValueLimit {
			writeError(w, http.StatusUnprocessableEntity, "фраза "+key+" длиннее 10000 символов")
			return
		}
		if strings.TrimSpace(value) == "" {
			cleared = append(cleared, key)
			continue
		}
		keys = append(keys, key)
		values = append(values, value)
	}

	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить фразы")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if body.Replace {
		_, err = tx.Exec(ctx, `DELETE FROM core.translation_keys WHERE locale = $1`, code)
	} else if len(cleared) > 0 {
		_, err = tx.Exec(ctx, `DELETE FROM core.translation_keys WHERE locale = $1 AND key = ANY($2::text[])`, code, cleared)
	}
	if err == nil && len(keys) > 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO core.translation_keys (locale, key, value)
			SELECT $1, m.key, m.value FROM unnest($2::text[], $3::text[]) AS m(key, value)
			ON CONFLICT (locale, key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
		`, code, keys, values)
	}
	if err == nil {
		err = tx.Commit(ctx)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить фразы")
		return
	}

	audit(ctx, h.dbOf(ctx), claims.UserID, "language.messages", "language:"+code, map[string]any{
		"saved":   len(keys),
		"cleared": len(cleared),
		"replace": body.Replace,
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"code":     code,
		"messages": h.loadTranslations(ctx, code),
	})
}
