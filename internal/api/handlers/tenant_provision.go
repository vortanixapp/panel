package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// ProvisionTenantDatabase заводит базу арендатора. Вызывает биллинг в момент
// установки панели: к активации в браузере база должна уже существовать, иначе
// лицензия привяжется не туда.
func (h *Handler) ProvisionTenantDatabase(w http.ResponseWriter, r *http.Request) {
	if !h.internalAuthorized(r) {
		writeError(w, http.StatusForbidden, "только для внутренних вызовов")
		return
	}
	if h.tenants == nil {
		writeError(w, http.StatusServiceUnavailable, "реестр баз не готов")
		return
	}

	var req struct {
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if slug == "" {
		writeError(w, http.StatusBadRequest, "не указан арендатор")
		return
	}

	creds, err := h.tenants.ProvisionWithUser(r.Context(), slug)
	if err != nil {
		log.Printf("база арендатора %s не заведена: %v", slug, err)
		writeError(w, http.StatusInternalServerError, "не удалось завести базу панели")
		return
	}

	// Пустой пароль здесь означает только одно: база у арендатора уже заведена
	// до конца, и пароль хранится у того, кто сохранил его в первый раз. После
	// оборвавшейся попытки пароль приходит новый — сохранять его снова.
	writeJSON(w, http.StatusOK, map[string]string{
		"slug":        slug,
		"db_host":     h.dbPublicHost,
		"db_port":     h.dbPublicPort,
		"db_name":     creds.DBName,
		"db_user":     creds.User,
		"db_password": creds.Password,
	})
}

func (h *Handler) internalAuthorized(r *http.Request) bool {
	if h.internalSecret == "" {
		return false
	}
	got := r.Header.Get("X-Internal-Secret")
	return subtle.ConstantTimeCompare([]byte(got), []byte(h.internalSecret)) == 1
}

// DropTenantDatabase удаляет базу арендатора. Зовёт биллинг при удалении
// лицензии: без ключа панель всё равно не работает, а данные держать незачем.
func (h *Handler) DropTenantDatabase(w http.ResponseWriter, r *http.Request) {
	if !h.internalAuthorized(r) {
		writeError(w, http.StatusForbidden, "только для внутренних вызовов")
		return
	}
	if h.tenants == nil {
		writeError(w, http.StatusServiceUnavailable, "реестр баз не готов")
		return
	}

	var req struct {
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	slug := strings.ToLower(strings.TrimSpace(req.Slug))
	if slug == "" {
		writeError(w, http.StatusBadRequest, "не указан арендатор")
		return
	}

	if err := h.tenants.Drop(r.Context(), slug); err != nil {
		log.Printf("база арендатора %s не удалена: %v", slug, err)
		writeError(w, http.StatusInternalServerError, "не удалось удалить базу панели")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"slug": slug, "status": "dropped"})
}
