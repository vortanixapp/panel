package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const (
	newsImageMaxBytes = 5 * 1024 * 1024
	newsTitleMaxLen   = 191
	newsSlugMaxLen    = 191
	newsExcerptMaxLen = 255
)

var slugTranslit = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch",
	'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

func newsSlugify(title string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case slugTranslit[r] != "":
			b.WriteString(slugTranslit[r])
			prevDash = false
		case r == 'ъ' || r == 'ь':
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteRune('-')
				prevDash = true
			}
		}
		if b.Len() >= newsSlugMaxLen {
			break
		}
	}
	return strings.Trim(b.String(), "-")
}

func newsSlugOrFallback(slug, title string, now time.Time) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		slug = newsSlugify(title)
	}
	if slug == "" {
		slug = "news-" + now.Format("20060102150405")
	}
	if len(slug) > newsSlugMaxLen {
		slug = slug[:newsSlugMaxLen]
	}
	return slug
}

type newsBody struct {
	Slug        *string `json:"slug"`
	Title       *string `json:"title"`
	Excerpt     *string `json:"excerpt"`
	Body        *string `json:"body"`
	PublishedAt *string `json:"published_at"`
	Active      *bool   `json:"active"`
	Tag         *string `json:"tag"`
	Pinned      *bool   `json:"pinned"`
}

func validateNews(title string, excerpt, slug string) error {
	t := strings.TrimSpace(title)
	if len([]rune(t)) < 2 {
		return fmt.Errorf("заголовок обязателен")
	}
	if len(t) > newsTitleMaxLen {
		return fmt.Errorf("заголовок не длиннее %d символов", newsTitleMaxLen)
	}
	if len(excerpt) > newsExcerptMaxLen {
		return fmt.Errorf("краткое описание не длиннее %d символов", newsExcerptMaxLen)
	}
	if len(slug) > newsSlugMaxLen {
		return fmt.Errorf("slug не длиннее %d символов", newsSlugMaxLen)
	}
	return nil
}

func parseNewsDate(v *string) (any, bool, error) {
	if v == nil {
		return nil, false, nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil, true, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02 15:04:05", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true, nil
		}
	}
	return nil, false, fmt.Errorf("некорректная дата публикации")
}

func (h *Handler) UpdateNews(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id := chi.URLParam(r, "id")
	var body newsBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	publishedAt, hasPublished, err := parseNewsDate(body.PublishedAt)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if body.Title != nil || body.Excerpt != nil || body.Slug != nil {
		title, excerpt, slug := "", "", ""
		if body.Title != nil {
			title = *body.Title
		}
		if body.Excerpt != nil {
			excerpt = *body.Excerpt
		}
		if body.Slug != nil {
			slug = *body.Slug
		}
		if body.Title != nil {
			if err := validateNews(title, excerpt, slug); err != nil {
				writeError(w, http.StatusUnprocessableEntity, err.Error())
				return
			}
		}
	}

	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.news SET
			slug         = COALESCE(NULLIF($2, ''), slug),
			title        = COALESCE($3, title),
			excerpt      = COALESCE($4, excerpt),
			body         = COALESCE($5, body),
			published_at = CASE WHEN $6::bool THEN $7::timestamptz ELSE published_at END,
			active       = COALESCE($8, active),
			tag          = COALESCE(NULLIF($9, ''), tag),
			pinned       = COALESCE($10, pinned),
			updated_at   = now()
		WHERE id = $1::uuid
	`, id,
		strings.TrimSpace(derefOrEmpty(body.Slug)), body.Title, body.Excerpt, body.Body,
		hasPublished, publishedAt, body.Active,
		strings.TrimSpace(derefOrEmpty(body.Tag)), body.Pinned)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "новость не найдена")
		return
	}
	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "news.update", "news:"+id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func derefOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func (h *Handler) UploadNewsImage(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	var slug string
	if err := h.dbOf(r.Context()).QueryRow(r.Context(),
		`SELECT slug FROM core.news WHERE id = $1::uuid`, id).Scan(&slug); err != nil {
		writeError(w, http.StatusNotFound, "новость не найдена")
		return
	}
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "не удалось разобрать форму")
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "файл изображения обязателен")
		return
	}
	defer file.Close()

	ext, allowed := catalogImageExt[header.Header.Get("Content-Type")]
	if !allowed {
		lower := strings.ToLower(header.Filename)
		switch {
		case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
			ext = ".jpg"
		case strings.HasSuffix(lower, ".png"):
			ext = ".png"
		case strings.HasSuffix(lower, ".webp"):
			ext = ".webp"
		case strings.HasSuffix(lower, ".gif"):
			ext = ".gif"
		default:
			writeError(w, http.StatusUnprocessableEntity, "изображение должно быть jpg, png, webp или gif")
			return
		}
	}
	rel, _, err := h.saveCatalogFile("news", id, "img_", ext, file, newsImageMaxBytes)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	var imageID string
	err = h.dbOf(r.Context()).QueryRow(r.Context(), `
		INSERT INTO core.news_images ( news_id, path, sort)
		VALUES ( $1::uuid, $2, COALESCE((SELECT MAX(sort) + 1 FROM core.news_images WHERE news_id = $1::uuid), 0))
		RETURNING id::text
	`, id, rel).Scan(&imageID)
	if err != nil {
		_ = h.deleteCatalogFile(rel)
		writeError(w, http.StatusInternalServerError, "не удалось сохранить изображение: "+err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":  imageID,
		"url": h.publicBaseURL(r) + "/v1/news/images/" + imageID,
	})
}

func (h *Handler) DeleteNewsImage(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	newsID := chi.URLParam(r, "id")
	imageID := chi.URLParam(r, "imageId")

	var rel string
	if err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		DELETE FROM core.news_images
		WHERE id = $1::uuid AND news_id = $2::uuid
		RETURNING path
	`, imageID, newsID).Scan(&rel); err != nil {
		writeError(w, http.StatusNotFound, "изображение не найдено")
		return
	}
	_ = h.deleteCatalogFile(rel)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) ServeNewsImage(w http.ResponseWriter, r *http.Request) {
	imageID := chi.URLParam(r, "id")
	ctx := r.Context()
	pool := h.db
	var rel string
	var active bool
	if err := pool.QueryRow(ctx, `
		SELECT i.path, n.active
		FROM core.news_images i
		JOIN core.news n ON n.id = i.news_id
		WHERE i.id = $1::uuid
	`, imageID).Scan(&rel, &active); err != nil || !active {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	abs, err := h.resolveCatalogPath(rel)
	if err != nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	info, statErr := os.Stat(abs)
	if statErr != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	switch strings.ToLower(filepath.Ext(abs)) {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, abs)
}

func (h *Handler) newsImages(r *http.Request, newsIDs []string) map[string][]map[string]string {
	out := map[string][]map[string]string{}
	if len(newsIDs) == 0 {
		return out
	}
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT news_id::text, id::text FROM core.news_images
		WHERE news_id = ANY($1::uuid[])
		ORDER BY news_id, sort, created_at
	`, newsIDs)
	if err != nil {
		return out
	}
	defer rows.Close()
	base := h.publicBaseURL(r)
	for rows.Next() {
		var newsID, imageID string
		if rows.Scan(&newsID, &imageID) != nil {
			continue
		}
		out[newsID] = append(out[newsID], map[string]string{
			"id":  imageID,
			"url": base + "/v1/news/images/" + imageID,
		})
	}
	return out
}
