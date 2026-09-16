package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/internal/api/payments"
)

const legalBodyLimit = 200_000

var (
	legalKinds           = []string{"offer", "privacy", "consent", "cookies"}
	legalAcceptanceKinds = []string{"offer", "privacy", "consent"}
	legalTitles          = map[string]string{
		"offer":   "Публичная оферта",
		"privacy": "Политика обработки персональных данных",
		"consent": "Согласие на обработку персональных данных",
		"cookies": "Политика использования файлов cookie",
	}
)

type legalDocument struct {
	Kind               string    `json:"kind"`
	Version            int       `json:"version"`
	Title              string    `json:"title"`
	Body               string    `json:"body,omitempty"`
	RequiresAcceptance bool      `json:"requires_acceptance"`
	PublishedAt        time.Time `json:"published_at"`
	PublishedBy        string    `json:"published_by,omitempty"`
}

type legalConsent struct {
	Kind    string    `json:"kind"`
	Version int       `json:"version"`
	Title   string    `json:"title"`
	Action  string    `json:"action"`
	At      time.Time `json:"at"`
}

func (h *Handler) latestLegalDocuments(ctx context.Context, withBody bool) (map[string]legalDocument, error) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT DISTINCT ON (kind) kind, version, title, CASE WHEN $1::boolean THEN body ELSE '' END,
		       requires_acceptance, published_at
		FROM core.legal_documents
		ORDER BY kind, version DESC
	`, withBody)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]legalDocument{}
	for rows.Next() {
		var d legalDocument
		if err := rows.Scan(&d.Kind, &d.Version, &d.Title, &d.Body, &d.RequiresAcceptance, &d.PublishedAt); err != nil {
			return nil, err
		}
		out[d.Kind] = d
	}
	return out, rows.Err()
}

func (h *Handler) publicLegalInfo(ctx context.Context, settings map[string]string) map[string]any {
	profile := accountingProfileFrom(settings)
	docs, err := h.latestLegalDocuments(ctx, false)
	if err != nil {
		log.Printf("юридические документы не загружены: %v", err)
	}
	list := []legalDocument{}
	for _, kind := range legalKinds {
		if d, ok := docs[kind]; ok {
			list = append(list, d)
		}
	}
	_, offer := docs["offer"]
	_, privacy := docs["privacy"]
	_, consent := docs["consent"]
	banner := strings.TrimSpace(settings[cookieBannerSetting])
	return map[string]any{
		"documents": list,
		"company": map[string]string{
			"name":      firstNonEmpty(profile.Name, profile.FullName),
			"full_name": profile.FullName,
			"inn":       profile.INN,
			"ogrn":      profile.OGRN,
			"address":   profile.Address,
			"email":     profile.Email,
			"phone":     profile.Phone,
		},
		"cookie_banner": banner == "" || truthySetting(banner),
		"registration":  map[string]bool{"terms": offer || privacy, "personal_data": consent},
	}
}

func (h *Handler) PublicLegal(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	writeJSON(w, http.StatusOK, h.publicLegalInfo(ctx, h.loadTenantSettingStrings(ctx)))
}

func (h *Handler) PublicLegalDocument(w http.ResponseWriter, r *http.Request) {
	kind := chi.URLParam(r, "kind")
	if !slices.Contains(legalKinds, kind) {
		writeError(w, http.StatusNotFound, "Документ не найден")
		return
	}
	version, _ := strconv.Atoi(r.URL.Query().Get("version"))
	doc, err := h.legalDocument(r.Context(), kind, version)
	if err != nil {
		writeError(w, http.StatusNotFound, "Документ ещё не опубликован")
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *Handler) legalDocument(ctx context.Context, kind string, version int) (legalDocument, error) {
	var d legalDocument
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT d.kind, d.version, d.title, d.body, d.requires_acceptance, d.published_at, COALESCE(u.email, '')
		FROM core.legal_documents d
		LEFT JOIN core.users u ON u.id = d.published_by
		WHERE d.kind = $1 AND ($2 = 0 OR d.version = $2)
		ORDER BY d.version DESC
		LIMIT 1
	`, kind, version).Scan(&d.Kind, &d.Version, &d.Title, &d.Body, &d.RequiresAcceptance, &d.PublishedAt, &d.PublishedBy)
	return d, err
}

func (h *Handler) pendingLegalDocuments(ctx context.Context, userID string) ([]legalDocument, error) {
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT d.kind, d.version, d.title
		FROM (
			SELECT DISTINCT ON (kind) kind, version, title
			FROM core.legal_documents
			WHERE requires_acceptance AND kind = ANY($2::text[])
			ORDER BY kind, version DESC
		) d
		LEFT JOIN LATERAL (
			SELECT c.version, c.action FROM core.legal_consents c
			WHERE c.user_id = $1 AND c.kind = d.kind
			ORDER BY c.created_at DESC, (c.action = 'withdrawn') DESC
			LIMIT 1
		) c ON true
		WHERE c.version IS NULL OR c.action <> 'accepted' OR c.version < d.version
		ORDER BY array_position($2::text[], d.kind)
	`, userID, legalAcceptanceKinds)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []legalDocument{}
	for rows.Next() {
		var d legalDocument
		if err := rows.Scan(&d.Kind, &d.Version, &d.Title); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (h *Handler) recordConsents(ctx context.Context, r *http.Request, userID string, kinds []string, source string) error {
	agent := r.Header.Get("User-Agent")
	if utf8.RuneCountInString(agent) > 300 {
		agent = string([]rune(agent)[:300])
	}
	_, err := h.dbOf(ctx).Exec(ctx, `
		INSERT INTO core.legal_consents (user_id, kind, version, action, source, ip, user_agent)
		SELECT $1::uuid, d.kind, d.version, 'accepted', $3::text, $4::text, $5::text
		FROM (
			SELECT DISTINCT ON (kind) kind, version FROM core.legal_documents
			WHERE kind = ANY($2::text[])
			ORDER BY kind, version DESC
		) d
	`, userID, kinds, source, clientIP(r), agent)
	return err
}

func (h *Handler) legalRegistrationKinds(ctx context.Context) []string {
	out := []string{}
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT DISTINCT kind FROM core.legal_documents WHERE kind = ANY($1::text[])
	`, legalAcceptanceKinds)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var kind string
		if rows.Scan(&kind) == nil {
			out = append(out, kind)
		}
	}
	return out
}

func (h *Handler) accountLegalPayload(ctx context.Context, userID, role string) (map[string]any, error) {
	pending := []legalDocument{}
	if !isStaffRole(role) {
		list, err := h.pendingLegalDocuments(ctx, userID)
		if err != nil {
			return nil, err
		}
		pending = list
	}
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT DISTINCT ON (c.kind) c.kind, c.version, COALESCE(d.title, ''), c.action, c.created_at
		FROM core.legal_consents c
		LEFT JOIN core.legal_documents d ON d.kind = c.kind AND d.version = c.version
		WHERE c.user_id = $1
		ORDER BY c.kind, c.created_at DESC, (c.action = 'withdrawn') DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	consents := []legalConsent{}
	for rows.Next() {
		var c legalConsent
		if err := rows.Scan(&c.Kind, &c.Version, &c.Title, &c.Action, &c.At); err != nil {
			return nil, err
		}
		consents = append(consents, c)
	}
	return map[string]any{"pending": pending, "consents": consents}, rows.Err()
}

func (h *Handler) AccountLegal(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	payload, err := h.accountLegalPayload(r.Context(), claims.UserID, claims.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (h *Handler) AccountLegalAccept(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		Kinds []string `json:"kinds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	kinds := []string{}
	for _, kind := range body.Kinds {
		if slices.Contains(legalAcceptanceKinds, kind) && !slices.Contains(kinds, kind) {
			kinds = append(kinds, kind)
		}
	}
	if len(kinds) == 0 {
		writeError(w, http.StatusBadRequest, "Выберите документы, которые принимаете")
		return
	}
	ctx := r.Context()
	if err := h.recordConsents(ctx, r, claims.UserID, kinds, "panel"); err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось записать согласие")
		return
	}
	payload, err := h.accountLegalPayload(ctx, claims.UserID, claims.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (h *Handler) AdminLegalDocuments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	current, err := h.latestLegalDocuments(ctx, true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT d.kind, d.version, d.title, d.requires_acceptance, d.published_at, COALESCE(u.email, '')
		FROM core.legal_documents d
		LEFT JOIN core.users u ON u.id = d.published_by
		ORDER BY d.kind, d.version DESC
	`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	versions := map[string][]legalDocument{}
	for rows.Next() {
		var d legalDocument
		if rows.Scan(&d.Kind, &d.Version, &d.Title, &d.RequiresAcceptance, &d.PublishedAt, &d.PublishedBy) == nil {
			versions[d.Kind] = append(versions[d.Kind], d)
		}
	}
	out := []map[string]any{}
	for _, kind := range legalKinds {
		item := map[string]any{
			"kind":          kind,
			"default_title": legalTitles[kind],
			"acceptance":    slices.Contains(legalAcceptanceKinds, kind),
			"current":       nil,
			"versions":      versions[kind],
		}
		if versions[kind] == nil {
			item["versions"] = []legalDocument{}
		}
		if d, ok := current[kind]; ok {
			item["current"] = d
		}
		out = append(out, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"documents": out})
}

func (h *Handler) AdminLegalDocumentVersion(w http.ResponseWriter, r *http.Request) {
	kind := chi.URLParam(r, "kind")
	version, err := strconv.Atoi(chi.URLParam(r, "version"))
	if !slices.Contains(legalKinds, kind) || err != nil || version <= 0 {
		writeError(w, http.StatusNotFound, "Документ не найден")
		return
	}
	doc, err := h.legalDocument(r.Context(), kind, version)
	if err != nil {
		writeError(w, http.StatusNotFound, "Документ не найден")
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *Handler) AdminLegalPublish(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	kind := chi.URLParam(r, "kind")
	if !slices.Contains(legalKinds, kind) {
		writeError(w, http.StatusNotFound, "Документ не найден")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, legalBodyLimit+(64<<10))
	var body struct {
		Title              string `json:"title"`
		Body               string `json:"body"`
		RequiresAcceptance bool   `json:"requires_acceptance"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	title := firstNonEmpty(strings.TrimSpace(body.Title), legalTitles[kind])
	text := strings.TrimSpace(strings.ReplaceAll(body.Body, "\r\n", "\n"))
	switch {
	case text == "":
		writeError(w, http.StatusBadRequest, "Текст документа пуст")
		return
	case len(text) > legalBodyLimit:
		writeError(w, http.StatusBadRequest, "Текст документа длиннее 200 000 символов")
		return
	case utf8.RuneCountInString(title) > 200:
		writeError(w, http.StatusBadRequest, "Заголовок длиннее 200 символов")
		return
	}
	requires := body.RequiresAcceptance && kind != "cookies"
	ctx := r.Context()
	var version int
	err := h.dbOf(ctx).QueryRow(ctx, `
		INSERT INTO core.legal_documents (kind, version, title, body, requires_acceptance, published_by)
		SELECT $1::text, COALESCE(MAX(version), 0) + 1, $2::text, $3::text,
		       CASE WHEN $1::text = 'cookies' THEN false
		            WHEN COALESCE(MAX(version), 0) = 0 THEN true
		            ELSE $4::boolean END,
		       $5::uuid
		FROM core.legal_documents WHERE kind = $1::text
		RETURNING version
	`, kind, title, text, requires, claims.UserID).Scan(&version)
	if err != nil {
		writeError(w, http.StatusConflict, "Не удалось опубликовать документ, повторите попытку")
		return
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "legal.publish", "legal:"+kind, map[string]any{
		"version": version, "requires_acceptance": requires,
	})
	doc, err := h.legalDocument(ctx, kind, version)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (h *Handler) AdminLegalTemplate(w http.ResponseWriter, r *http.Request) {
	kind := chi.URLParam(r, "kind")
	if !slices.Contains(legalKinds, kind) {
		writeError(w, http.StatusNotFound, "Шаблон не найден")
		return
	}
	title, body := legalTemplate(kind, h.accountingProfile(r.Context()), time.Now())
	writeJSON(w, http.StatusOK, map[string]string{"title": title, "body": body})
}

func (h *Handler) AdminLegalConsents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	const perPage = 50
	var total int
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COUNT(*) FROM core.legal_consents c JOIN core.users u ON u.id = c.user_id
		WHERE ($1 = '' OR c.kind = $1) AND ($2 = '' OR u.email ILIKE '%' || $2 || '%' OR c.ip LIKE $2 || '%')
	`, kind, q).Scan(&total)
	rows, err := h.dbOf(ctx).Query(ctx, `
		SELECT c.id::text, c.kind, c.version, c.action, c.source, c.ip, c.user_agent, c.created_at, u.id::text, u.email
		FROM core.legal_consents c
		JOIN core.users u ON u.id = c.user_id
		WHERE ($1 = '' OR c.kind = $1) AND ($2 = '' OR u.email ILIKE '%' || $2 || '%' OR c.ip LIKE $2 || '%')
		ORDER BY c.created_at DESC
		LIMIT $3 OFFSET $4
	`, kind, q, perPage, (page-1)*perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	list := []map[string]any{}
	for rows.Next() {
		var id, k, action, source, ip, agent, userID, email string
		var version int
		var at time.Time
		if rows.Scan(&id, &k, &version, &action, &source, &ip, &agent, &at, &userID, &email) == nil {
			list = append(list, map[string]any{
				"id": id, "kind": k, "version": version, "action": action, "source": source,
				"ip": ip, "user_agent": agent, "created_at": at, "user_id": userID, "email": email,
			})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"consents": list, "total": total, "page": page, "per_page": perPage})
}

func (h *Handler) adminLegalSettingsPayload(ctx context.Context) map[string]any {
	settings := h.loadTenantSettingStrings(ctx)
	ident := identificationSettingsFrom(settings)
	enabled := map[string]bool{}
	if rows, err := h.dbOf(ctx).Query(ctx, `SELECT provider FROM core.payment_providers WHERE enabled`); err == nil {
		for rows.Next() {
			var code string
			if rows.Scan(&code) == nil {
				enabled[code] = true
			}
		}
		rows.Close()
	}
	providers := []map[string]any{}
	for _, code := range payments.Codes() {
		providers = append(providers, map[string]any{"code": code, "name": payments.Name(code), "enabled": enabled[code]})
	}
	banner := strings.TrimSpace(settings[cookieBannerSetting])
	return map[string]any{
		"identification_required":  ident.Required,
		"identification_providers": ident.Providers,
		"cookie_banner":            banner == "" || truthySetting(banner),
		"providers":                providers,
	}
}

func (h *Handler) AdminLegalSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.adminLegalSettingsPayload(r.Context()))
}

func (h *Handler) AdminLegalSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		IdentificationRequired  bool     `json:"identification_required"`
		IdentificationProviders []string `json:"identification_providers"`
		CookieBanner            bool     `json:"cookie_banner"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	codes := payments.Codes()
	providers := []string{}
	for _, code := range body.IdentificationProviders {
		code = strings.TrimSpace(code)
		if slices.Contains(codes, code) && !slices.Contains(providers, code) {
			providers = append(providers, code)
		}
	}
	raw, _ := json.Marshal(providers)
	ctx := r.Context()
	h.setTenantSettingString(ctx, identificationRequiredSetting, strconv.FormatBool(body.IdentificationRequired))
	h.setTenantSettingString(ctx, identificationProvidersSetting, string(raw))
	h.setTenantSettingString(ctx, cookieBannerSetting, strconv.FormatBool(body.CookieBanner))
	audit(ctx, h.dbOf(ctx), claims.UserID, "settings.update", "legal", map[string]any{
		"identification_required": body.IdentificationRequired, "identification_providers": providers,
		"cookie_banner": body.CookieBanner,
	})
	writeJSON(w, http.StatusOK, h.adminLegalSettingsPayload(ctx))
}

func (h *Handler) recordRegistrationConsents(ctx context.Context, r *http.Request, userID string, kinds []string) {
	if len(kinds) == 0 {
		return
	}
	if err := h.recordConsents(ctx, r, userID, kinds, "register"); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		log.Printf("согласия при регистрации %s не записаны: %v", userID, err)
	}
}
