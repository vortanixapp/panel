package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
)

type kbArticle struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Excerpt   string `json:"excerpt"`
	Body      string `json:"body,omitempty"`
	Category  string `json:"category"`
	Published bool   `json:"published"`
	Views     int    `json:"views"`
	Position  int    `json:"position"`
	UpdatedAt string `json:"updated_at"`
}

func (h *Handler) ListKBArticles(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	category := strings.TrimSpace(r.URL.Query().Get("category"))
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, slug, title, excerpt, category, published, views, position,
		       updated_at::text
		FROM core.kb_articles
		WHERE published
		  AND ($1 = '' OR category = $1)
		  AND ($2 = '' OR title ILIKE '%' || $2 || '%'
		                OR excerpt ILIKE '%' || $2 || '%'
		                OR body ILIKE '%' || $2 || '%')
		ORDER BY position, views DESC, title
		LIMIT 100
	`, category, query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось прочитать базу знаний")
		return
	}
	defer rows.Close()

	list := []kbArticle{}
	for rows.Next() {
		var a kbArticle
		if rows.Scan(&a.ID, &a.Slug, &a.Title, &a.Excerpt, &a.Category, &a.Published,
			&a.Views, &a.Position, &a.UpdatedAt) == nil {
			list = append(list, a)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"articles": list})
}

func (h *Handler) GetKBArticle(w http.ResponseWriter, r *http.Request) {
	_, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slug := chi.URLParam(r, "slug")
	var a kbArticle
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		UPDATE core.kb_articles
		SET views = views + 1
		WHERE slug = $1 AND published
		RETURNING id::text, slug, title, excerpt, body, category, published, views,
		          position, updated_at::text
	`, slug).Scan(&a.ID, &a.Slug, &a.Title, &a.Excerpt, &a.Body,
		&a.Category, &a.Published, &a.Views, &a.Position, &a.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "статья не найдена")
		return
	}

	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) AdminListKBArticles(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	if !requireAdmin(w, claims) {
		return
	}

	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, slug, title, excerpt, body, category, published, views,
		       position, updated_at::text
		FROM core.kb_articles
		ORDER BY position, title
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось прочитать базу знаний")
		return
	}
	defer rows.Close()

	list := []kbArticle{}
	for rows.Next() {
		var a kbArticle
		if rows.Scan(&a.ID, &a.Slug, &a.Title, &a.Excerpt, &a.Body, &a.Category,
			&a.Published, &a.Views, &a.Position, &a.UpdatedAt) == nil {
			list = append(list, a)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"articles": list})
}

var kbSlugStrip = regexp.MustCompile(`[^a-z0-9]+`)

func kbSlug(raw string) string {
	s := kbSlugStrip.ReplaceAllString(strings.ToLower(strings.TrimSpace(raw)), "-")
	return strings.Trim(s, "-")
}

func (h *Handler) AdminSaveKBArticle(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	if !requireAdmin(w, claims) {
		return
	}

	var body struct {
		ID        string `json:"id"`
		Slug      string `json:"slug"`
		Title     string `json:"title"`
		Excerpt   string `json:"excerpt"`
		Body      string `json:"body"`
		Category  string `json:"category"`
		Published *bool  `json:"published"`
		Position  int    `json:"position"`
	}
	if json.NewDecoder(r.Body).Decode(&body) != nil {
		writeError(w, http.StatusBadRequest, "неверный запрос")
		return
	}

	title := strings.TrimSpace(body.Title)
	if title == "" {
		writeError(w, http.StatusBadRequest, "заголовок обязателен")
		return
	}

	slug := kbSlug(body.Slug)
	if slug == "" {
		slug = kbSlug(title)
	}
	if slug == "" {
		writeError(w, http.StatusBadRequest, "задайте адрес статьи латиницей")
		return
	}

	category := strings.TrimSpace(body.Category)
	if category == "" {
		category = "other"
	}
	published := true
	if body.Published != nil {
		published = *body.Published
	}

	var id string
	var err error
	if strings.TrimSpace(body.ID) == "" {
		err = h.dbOf(r.Context()).QueryRow(r.Context(), `
			INSERT INTO core.kb_articles
			    ( slug, title, excerpt, body, category, published, position)
			VALUES ( $1, $2, $3, $4, $5, $6, $7)
			RETURNING id::text
		`, slug, title, body.Excerpt, body.Body, category, published,
			body.Position).Scan(&id)
	} else {
		err = h.dbOf(r.Context()).QueryRow(r.Context(), `
			UPDATE core.kb_articles
			SET slug = $2, title = $3, excerpt = $4, body = $5, category = $6,
			    published = $7, position = $8, updated_at = now()
			WHERE id = $1::uuid
			RETURNING id::text
		`, body.ID, slug, title, body.Excerpt, body.Body, category,
			published, body.Position).Scan(&id)
	}
	if err != nil || id == "" {
		writeError(w, http.StatusBadRequest, "не удалось сохранить статью — возможно, адрес занят")
		return
	}

	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "admin.kb.save", slug, nil)
	writeJSON(w, http.StatusOK, map[string]string{"id": id, "slug": slug})
}

func (h *Handler) AdminDeleteKBArticle(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	if !requireAdmin(w, claims) {
		return
	}

	id := chi.URLParam(r, "id")
	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `
		DELETE FROM core.kb_articles WHERE id = $1::uuid
	`, id)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "статья не найдена")
		return
	}

	audit(r.Context(), h.dbOf(r.Context()), claims.UserID, "admin.kb.delete", id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
