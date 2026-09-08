package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
)

// База знаний.
//
// Половина обращений в поддержку повторяется дословно, а показать клиенту было
// нечего: статей в продукте не существовало. Экран базы знаний стоит перед
// формой создания тикета — если ответ уже написан, обращение не появится.

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

// ListKBArticles — список статей для клиента.
//
// Тело статьи в списке не отдаём: оно бывает длинным, а список показывает
// только заголовок и выжимку. За телом идёт отдельный запрос по slug.
func (h *Handler) ListKBArticles(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	category := strings.TrimSpace(r.URL.Query().Get("category"))
	query := strings.TrimSpace(r.URL.Query().Get("q"))

	// Поиск идёт и по заголовку, и по тексту: искать только по заголовку
	// значит не находить статью, названную иначе, чем спрашивает клиент.
	rows, err := h.dbOf(r.Context()).Query(r.Context(), `
		SELECT id::text, slug, title, excerpt, category, published, views, position,
		       updated_at::text
		FROM core.kb_articles
		WHERE tenant_id = $1
		  AND published
		  AND ($2 = '' OR category = $2)
		  AND ($3 = '' OR title ILIKE '%' || $3 || '%'
		                OR excerpt ILIKE '%' || $3 || '%'
		                OR body ILIKE '%' || $3 || '%')
		ORDER BY position, views DESC, title
		LIMIT 100
	`, claims.TenantID, category, query)
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

// GetKBArticle отдаёт статью целиком и считает просмотр.
func (h *Handler) GetKBArticle(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slug := chi.URLParam(r, "slug")
	var a kbArticle
	// Счётчик увеличиваем той же командой, что и читаем: отдельным UPDATE он
	// разошёлся бы с выдачей при одновременных запросах, а список «частых
	// вопросов» строится именно по нему.
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `
		UPDATE core.kb_articles
		SET views = views + 1
		WHERE tenant_id = $1 AND slug = $2 AND published
		RETURNING id::text, slug, title, excerpt, body, category, published, views,
		          position, updated_at::text
	`, claims.TenantID, slug).Scan(&a.ID, &a.Slug, &a.Title, &a.Excerpt, &a.Body,
		&a.Category, &a.Published, &a.Views, &a.Position, &a.UpdatedAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "статья не найдена")
		return
	}

	writeJSON(w, http.StatusOK, a)
}

// AdminListKBArticles отдаёт и неопубликованные: админ должен видеть черновики.
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
		WHERE tenant_id = $1
		ORDER BY position, title
	`, claims.TenantID)
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

// kbSlug приводит заголовок к адресу.
//
// Заголовки здесь русские, а транслитерации в проекте нет, поэтому от русского
// заголовка не остаётся ничего. В этом случае slug не выдумываем: пустой
// результат означает «пусть автор задаст адрес сам», и вызывающий вернёт
// понятную ошибку вместо статьи по адресу вида «---».
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
			    (tenant_id, slug, title, excerpt, body, category, published, position)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id::text
		`, claims.TenantID, slug, title, body.Excerpt, body.Body, category, published,
			body.Position).Scan(&id)
	} else {
		// Просмотры при правке не сбрасываем: они накоплены читателями, а не
		// автором, и обнулять их редактированием опечатки незачем.
		err = h.dbOf(r.Context()).QueryRow(r.Context(), `
			UPDATE core.kb_articles
			SET slug = $3, title = $4, excerpt = $5, body = $6, category = $7,
			    published = $8, position = $9, updated_at = now()
			WHERE tenant_id = $1 AND id = $2::uuid
			RETURNING id::text
		`, claims.TenantID, body.ID, slug, title, body.Excerpt, body.Body, category,
			published, body.Position).Scan(&id)
	}
	if err != nil || id == "" {
		writeError(w, http.StatusBadRequest, "не удалось сохранить статью — возможно, адрес занят")
		return
	}

	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "admin.kb.save", slug, nil)
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
		DELETE FROM core.kb_articles WHERE tenant_id = $1 AND id = $2::uuid
	`, claims.TenantID, id)
	if err != nil || tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "статья не найдена")
		return
	}

	audit(r.Context(), h.dbOf(r.Context()), claims.TenantID, claims.UserID, "admin.kb.delete", id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
