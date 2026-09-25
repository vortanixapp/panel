package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/pkg/sitetpl"
)

const (
	siteVersionsKept   = 50
	siteAssetSize      = 5 << 20
	siteNoteLimit      = 300
	sitePublicCacheTTL = 5 * time.Second
)

type siteCache struct {
	mu   sync.Mutex
	at   time.Time
	body []byte
}

func (c *siteCache) get() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.body == nil || time.Since(c.at) > sitePublicCacheTTL {
		return nil
	}
	return c.body
}

func (c *siteCache) set(body []byte) {
	c.mu.Lock()
	c.body, c.at = body, time.Now()
	c.mu.Unlock()
}

func (c *siteCache) reset() {
	c.mu.Lock()
	c.body = nil
	c.mu.Unlock()
}

var publicSite siteCache

type siteState struct {
	Draft          sitetpl.Document
	Published      sitetpl.Document
	Revision       int
	DraftUpdatedAt *time.Time
	DraftUpdatedBy string
	PublishedAt    *time.Time
	PublishedBy    string
}

const siteSelect = `
	SELECT t.draft, t.published, t.revision,
	       t.draft_updated_at, COALESCE(du.email, ''),
	       t.published_at, COALESCE(pu.email, '')
	FROM core.site_template t
	LEFT JOIN core.users du ON du.id = t.draft_updated_by
	LEFT JOIN core.users pu ON pu.id = t.published_by
	WHERE t.id = 1`

func scanSite(row pgx.Row) (siteState, error) {
	var s siteState
	var draft, published []byte
	if err := row.Scan(&draft, &published, &s.Revision, &s.DraftUpdatedAt, &s.DraftUpdatedBy, &s.PublishedAt, &s.PublishedBy); err != nil {
		return s, err
	}
	var err error
	if s.Draft, err = sitetpl.Parse(draft); err != nil {
		return s, err
	}
	if s.Published, err = sitetpl.Parse(published); err != nil {
		return s, err
	}
	s.Published = s.Published.WithoutTexts()
	return s, nil
}

func (s siteState) changed() bool {
	return s.Draft.TextCount() > 0 || !sitetpl.Same(s.Draft.WithoutTexts(), s.Published)
}

func (s siteState) status() map[string]any {
	return map[string]any{
		"revision":         s.Revision,
		"changed":          s.changed(),
		"pending_texts":    s.Draft.TextCount(),
		"draft_updated_at": s.DraftUpdatedAt,
		"draft_updated_by": s.DraftUpdatedBy,
		"published_at":     s.PublishedAt,
		"published_by":     s.PublishedBy,
	}
}

func (s siteState) payload() map[string]any {
	out := s.status()
	out["draft"] = s.Draft
	out["published"] = s.Published
	return out
}

func (h *Handler) loadSite(ctx context.Context) (siteState, error) {
	db := h.dbOf(ctx)
	if _, err := db.Exec(ctx, `INSERT INTO core.site_template (id) VALUES (1) ON CONFLICT (id) DO NOTHING`); err != nil {
		return siteState{}, err
	}
	return scanSite(db.QueryRow(ctx, siteSelect))
}

func lockSite(ctx context.Context, tx pgx.Tx) (siteState, error) {
	if _, err := tx.Exec(ctx, `INSERT INTO core.site_template (id) VALUES (1) ON CONFLICT (id) DO NOTHING`); err != nil {
		return siteState{}, err
	}
	return scanSite(tx.QueryRow(ctx, siteSelect+` FOR UPDATE OF t`))
}

func (h *Handler) siteLocales(ctx context.Context) map[string]bool {
	out := map[string]bool{}
	for _, l := range h.loadLanguages(ctx, false) {
		out[l.Code] = true
	}
	return out
}

func (h *Handler) allowSiteScripts(w http.ResponseWriter, r *http.Request, doc *sitetpl.Document) bool {
	wanted := false
	doc.EachBlock(func(b *sitetpl.Block) {
		if b.Type == "html" && b.Props != nil {
			if on, _ := b.Props["scripts"].(bool); on {
				wanted = true
			}
		}
	})
	if !wanted {
		return true
	}
	claims, ok := tenantClaims(r.Context())
	if _, isKey := apiKeyFromContext(r.Context()); ok && !isKey && claims.Role == "owner" {
		return true
	}
	writeCodedError(w, http.StatusForbidden, "owner_only",
		"Скрипты в блоке HTML может включить только владелец панели")
	return false
}

func (h *Handler) normalizeSite(ctx context.Context, w http.ResponseWriter, doc *sitetpl.Document) bool {
	err := sitetpl.Normalize(doc, sitetpl.Options{Locales: h.siteLocales(ctx)})
	if err == nil {
		return true
	}
	var invalid *sitetpl.Error
	if errors.As(err, &invalid) {
		writeError(w, http.StatusUnprocessableEntity, invalid.Error())
		return false
	}
	writeError(w, http.StatusInternalServerError, "Не удалось проверить шаблон")
	return false
}

func (h *Handler) PublicSite(w http.ResponseWriter, r *http.Request) {
	body := publicSite.get()
	if body == nil {
		ctx := r.Context()
		var raw []byte
		var publishedAt *time.Time
		doc := sitetpl.Document{}
		err := h.dbOf(ctx).QueryRow(ctx, `SELECT published, published_at FROM core.site_template WHERE id = 1`).Scan(&raw, &publishedAt)
		if err == nil {
			if parsed, perr := sitetpl.Parse(raw); perr == nil {
				doc = parsed
			}
		}
		body, _ = json.Marshal(map[string]any{
			"document":     doc.Public(),
			"published_at": publishedAt,
		})
		if err == nil || errors.Is(err, pgx.ErrNoRows) {
			publicSite.set(body)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(body)
}

func (h *Handler) AdminSiteMenu(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var raw []byte
	var menu any
	if h.dbOf(ctx).QueryRow(ctx, `SELECT published FROM core.site_template WHERE id = 1`).Scan(&raw) == nil {
		if doc, err := sitetpl.Parse(raw); err == nil {
			if m, ok := doc.Menus[sitetpl.AdminMenu]; ok {
				menu = m
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"menu": menu})
}

func (h *Handler) AdminTemplate(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		return
	}
	s, err := h.loadSite(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось загрузить шаблон")
		return
	}
	writeJSON(w, http.StatusOK, s.payload())
}

func (h *Handler) AdminTemplateSaveDraft(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, sitetpl.MaxDocumentBytes+(64<<10))
	var body struct {
		Document json.RawMessage `json:"document"`
		Revision int             `json:"revision"`
		Force    bool            `json:"force"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "Шаблон больше 1 МБ")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(body.Document) > sitetpl.MaxDocumentBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "Шаблон больше 1 МБ")
		return
	}
	doc, err := sitetpl.Parse(body.Document)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Шаблон повреждён")
		return
	}
	if !h.allowSiteScripts(w, r, &doc) {
		return
	}
	if !h.normalizeSite(ctx, w, &doc) {
		return
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить черновик")
		return
	}

	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить черновик")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := lockSite(ctx, tx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить черновик")
		return
	}
	if !body.Force && body.Revision != current.Revision {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":    "Черновик уже изменил " + staffName(current.DraftUpdatedBy) + ". Загрузите его версию или перезапишите своей",
			"template": current.payload(),
		})
		return
	}
	if _, err := tx.Exec(ctx, `
		UPDATE core.site_template
		SET draft = $1::jsonb, revision = revision + 1, draft_updated_at = now(), draft_updated_by = NULLIF($2, '')::uuid
		WHERE id = 1
	`, raw, claims.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить черновик")
		return
	}
	next, err := scanSite(tx.QueryRow(ctx, siteSelect))
	if err != nil || tx.Commit(ctx) != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить черновик")
		return
	}
	writeJSON(w, http.StatusOK, next.status())
}

func staffName(email string) string {
	if email == "" {
		return "другой сотрудник"
	}
	return email
}

func (h *Handler) AdminTemplatePublish(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	var body struct {
		Note     string `json:"note"`
		Revision int    `json:"revision"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	note := strings.TrimSpace(body.Note)
	if utf8.RuneCountInString(note) > siteNoteLimit {
		writeError(w, http.StatusUnprocessableEntity, "Заметка к версии не длиннее 300 символов")
		return
	}

	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось опубликовать шаблон")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	current, err := lockSite(ctx, tx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось опубликовать шаблон")
		return
	}
	if body.Revision != current.Revision {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":    "Черновик изменился после открытия редактора. Проверьте изменения и опубликуйте снова",
			"template": current.payload(),
		})
		return
	}
	if !current.changed() {
		writeError(w, http.StatusUnprocessableEntity, "Нет изменений для публикации")
		return
	}

	doc := current.Draft.WithoutTexts()
	changes, err := applySiteTexts(ctx, tx, current.Draft.Texts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить тексты")
		return
	}
	docRaw, _ := json.Marshal(doc)
	textsRaw, _ := json.Marshal(changes)
	if _, err := tx.Exec(ctx, `
		INSERT INTO core.site_template_versions (document, texts, note, created_by)
		VALUES ($1::jsonb, $2::jsonb, $3, NULLIF($4, '')::uuid)
	`, docRaw, textsRaw, note, claims.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось опубликовать шаблон")
		return
	}
	if _, err := tx.Exec(ctx, `
		UPDATE core.site_template
		SET draft = $1::jsonb, published = $1::jsonb, revision = revision + 1,
		    published_at = now(), published_by = NULLIF($2, '')::uuid,
		    draft_updated_at = now(), draft_updated_by = NULLIF($2, '')::uuid
		WHERE id = 1
	`, docRaw, claims.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось опубликовать шаблон")
		return
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM core.site_template_versions
		WHERE id NOT IN (SELECT id FROM core.site_template_versions ORDER BY id DESC LIMIT $1)
	`, siteVersionsKept); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось опубликовать шаблон")
		return
	}
	next, err := scanSite(tx.QueryRow(ctx, siteSelect))
	if err != nil || tx.Commit(ctx) != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось опубликовать шаблон")
		return
	}
	publicSite.reset()
	audit(ctx, h.dbOf(ctx), claims.UserID, "settings.template.publish", "template", map[string]any{
		"texts": changes.Count(),
		"note":  note,
	})
	writeJSON(w, http.StatusOK, next.payload())
}

func applySiteTexts(ctx context.Context, tx pgx.Tx, texts map[string]map[string]string) (sitetpl.Changes, error) {
	changes := sitetpl.Changes{}
	for locale, keys := range texts {
		if len(keys) == 0 {
			continue
		}
		current, err := currentPhrases(ctx, tx, locale, keys)
		if err != nil {
			return nil, err
		}
		var upKeys, upValues, dropKeys []string
		for key, to := range keys {
			if strings.TrimSpace(to) == "" {
				to = ""
			}
			from := current[key]
			if from == to {
				continue
			}
			changes.Set(locale, key, sitetpl.Change{From: from, To: to})
			if to == "" {
				dropKeys = append(dropKeys, key)
			} else {
				upKeys = append(upKeys, key)
				upValues = append(upValues, to)
			}
		}
		if len(dropKeys) > 0 {
			if _, err := tx.Exec(ctx, `DELETE FROM core.translation_keys WHERE locale = $1 AND key = ANY($2::text[])`, locale, dropKeys); err != nil {
				return nil, err
			}
		}
		if len(upKeys) > 0 {
			if _, err := tx.Exec(ctx, `
				INSERT INTO core.translation_keys (locale, key, value)
				SELECT $1, m.key, m.value FROM unnest($2::text[], $3::text[]) AS m(key, value)
				ON CONFLICT (locale, key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
			`, locale, upKeys, upValues); err != nil {
				return nil, err
			}
		}
	}
	return changes, nil
}

func currentPhrases(ctx context.Context, tx pgx.Tx, locale string, keys map[string]string) (map[string]string, error) {
	names := make([]string, 0, len(keys))
	for key := range keys {
		names = append(names, key)
	}
	rows, err := tx.Query(ctx, `
		SELECT key, value FROM core.translation_keys
		WHERE locale = $1 AND key = ANY($2::text[]) AND btrim(value) <> ''
	`, locale, names)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		out[key] = value
	}
	return out, rows.Err()
}

func (h *Handler) AdminTemplateDiscard(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	if _, err := h.dbOf(ctx).Exec(ctx, `
		UPDATE core.site_template
		SET draft = published, revision = revision + 1, draft_updated_at = now(), draft_updated_by = NULLIF($1, '')::uuid
		WHERE id = 1
	`, claims.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сбросить черновик")
		return
	}
	s, err := h.loadSite(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось загрузить шаблон")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "settings.template.discard", "template", nil)
	writeJSON(w, http.StatusOK, s.payload())
}

func (h *Handler) AdminTemplateVersions(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		return
	}
	ctx := r.Context()
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT v.id, v.created_at, COALESCE(u.email, ''), v.note, v.texts
		FROM core.site_template_versions v
		LEFT JOIN core.users u ON u.id = v.created_by
		ORDER BY v.id DESC
		LIMIT $1
	`, siteVersionsKept)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось загрузить версии")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id int64
		var createdAt time.Time
		var author, note string
		var textsRaw []byte
		if rows.Scan(&id, &createdAt, &author, &note, &textsRaw) != nil {
			continue
		}
		var changes sitetpl.Changes
		_ = json.Unmarshal(textsRaw, &changes)
		items = append(items, map[string]any{
			"id":         id,
			"created_at": createdAt,
			"author":     author,
			"note":       note,
			"texts":      changes.Count(),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) AdminTemplateRestore(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusNotFound, "Версия не найдена")
		return
	}

	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось вернуть версию")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := lockSite(ctx, tx); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось вернуть версию")
		return
	}
	var docRaw []byte
	if err := tx.QueryRow(ctx, `SELECT document FROM core.site_template_versions WHERE id = $1`, id).Scan(&docRaw); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "Версия не найдена")
			return
		}
		writeError(w, http.StatusInternalServerError, "Не удалось вернуть версию")
		return
	}
	doc, err := sitetpl.Parse(docRaw)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Версия повреждена")
		return
	}
	newer, err := newerTextChanges(ctx, tx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось вернуть версию")
		return
	}
	doc.Texts, err = pendingReverts(ctx, tx, sitetpl.Reverts(newer))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось вернуть версию")
		return
	}
	if !h.allowSiteScripts(w, r, &doc) {
		return
	}
	if !h.normalizeSite(ctx, w, &doc) {
		return
	}
	raw, _ := json.Marshal(doc)
	if _, err := tx.Exec(ctx, `
		UPDATE core.site_template
		SET draft = $1::jsonb, revision = revision + 1, draft_updated_at = now(), draft_updated_by = NULLIF($2, '')::uuid
		WHERE id = 1
	`, raw, claims.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось вернуть версию")
		return
	}
	next, err := scanSite(tx.QueryRow(ctx, siteSelect))
	if err != nil || tx.Commit(ctx) != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось вернуть версию")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "settings.template.restore", "template", map[string]any{"version": id})
	writeJSON(w, http.StatusOK, next.payload())
}

func newerTextChanges(ctx context.Context, tx pgx.Tx, id int64) ([]sitetpl.Changes, error) {
	rows, err := tx.Query(ctx, `SELECT texts FROM core.site_template_versions WHERE id > $1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []sitetpl.Changes
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var changes sitetpl.Changes
		if json.Unmarshal(raw, &changes) == nil && len(changes) > 0 {
			out = append(out, changes)
		}
	}
	return out, rows.Err()
}

func pendingReverts(ctx context.Context, tx pgx.Tx, reverts map[string]map[string]string) (map[string]map[string]string, error) {
	out := map[string]map[string]string{}
	for locale, keys := range reverts {
		current, err := currentPhrases(ctx, tx, locale, keys)
		if err != nil {
			return nil, err
		}
		for key, value := range keys {
			if current[key] == value {
				continue
			}
			if out[locale] == nil {
				out[locale] = map[string]string{}
			}
			out[locale][key] = value
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func (h *Handler) AdminTemplateUploadAsset(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, siteAssetSize+(1<<20))
	if err := r.ParseMultipartForm(siteAssetSize); err != nil {
		writeError(w, http.StatusBadRequest, "Файл больше 5 МБ или повреждён")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "Выберите файл")
		return
	}
	defer file.Close()

	suffix := make([]byte, 6)
	if _, err := rand.Read(suffix); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить файл")
		return
	}
	base := "template-" + hex.EncodeToString(suffix) + "-" + strconv.FormatInt(time.Now().Unix(), 36)
	path, err := h.saveBrandingFile(file, header.Filename, base)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "Поддерживаются PNG, JPG, WEBP, GIF, SVG и ICO")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "settings.template.asset", path, nil)
	writeJSON(w, http.StatusOK, map[string]any{
		"path": path,
		"url":  h.brandingPublicURL(r, path),
	})
}
