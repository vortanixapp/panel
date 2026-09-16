package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"
)

func (h *Handler) SavedAccounts(w http.ResponseWriter, r *http.Request) {
	list := h.savedLogins(r)
	sort.Slice(list, func(i, j int) bool { return list[i].Used > list[j].Used })

	current := ""
	if claims, ok := tenantClaims(r.Context()); ok {
		current = claims.UserID
	}

	out := make([]map[string]any, 0, len(list))
	for _, item := range list {
		if item.ID == "" {
			continue
		}
		out = append(out, map[string]any{
			"id":         item.ID,
			"email":      item.Email,
			"role":       item.Role,
			"is_current": item.ID == current,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"accounts": out, "current_id": current})
}

func (h *Handler) SwitchAccount(w http.ResponseWriter, r *http.Request) {
	if !csrfValid(r) {
		writeCodedError(w, http.StatusForbidden, "csrf_failed",
			"Запрос отклонён: не совпал защитный ключ формы. Обновите страницу и повторите")
		return
	}
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	target := strings.TrimSpace(body.UserID)
	if target == "" {
		writeError(w, http.StatusBadRequest, "Не указан аккаунт")
		return
	}

	var saved *savedLogin
	for _, item := range h.savedLogins(r) {
		if item.ID == target {
			copied := item
			saved = &copied
			break
		}
	}
	if saved == nil {
		writeCodedError(w, http.StatusNotFound, "account_unknown",
			"Этот аккаунт не сохранён в браузере — войдите в него заново")
		return
	}

	userID, sessionID, expiresAt, err := h.tokens.ParseRefreshDetails(saved.Refresh)
	if err != nil || userID != target || sessionID == "" || expiresAt.Before(time.Now()) {
		h.forgetLogin(w, r, target)
		writeCodedError(w, http.StatusUnauthorized, "session_expired",
			"Сеанс этого аккаунта закрыт — войдите в него заново")
		return
	}

	ctx := r.Context()
	var alive bool
	if err := h.dbOf(ctx).QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM core.user_sessions WHERE id = $1 AND user_id = $2)`,
		sessionID, userID,
	).Scan(&alive); err != nil || !alive {
		h.forgetLogin(w, r, target)
		writeCodedError(w, http.StatusUnauthorized, "session_expired",
			"Сеанс этого аккаунта закрыт — войдите в него заново")
		return
	}

	var email, role string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT email, role FROM core.users WHERE id = $1 AND status = 'active'
	`, userID).Scan(&email, &role); err != nil {
		h.forgetLogin(w, r, target)
		writeCodedError(w, http.StatusUnauthorized, "account_unavailable",
			"Учётная запись недоступна")
		return
	}

	access, refresh, err := h.issueAuthTokens(r, userID, email, role, sessionID, rememberRefreshTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}
	h.startSession(w, r, userID, email, role, access, refresh, rememberRefreshTTL)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"user": map[string]string{"id": userID, "email": email, "role": role},
	})
}

func (h *Handler) ForgetAccount(w http.ResponseWriter, r *http.Request) {
	if !csrfValid(r) {
		writeCodedError(w, http.StatusForbidden, "csrf_failed",
			"Запрос отклонён: не совпал защитный ключ формы. Обновите страницу и повторите")
		return
	}
	var body struct {
		UserID string `json:"user_id"`
		All    bool   `json:"all"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.All {
		h.writeSavedLogins(w, r, nil, h.tokens.DefaultRefreshTTL())
		h.clearAuthCookies(w, r)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	h.forgetLogin(w, r, strings.TrimSpace(body.UserID))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
