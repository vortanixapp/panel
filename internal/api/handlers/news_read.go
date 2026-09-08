package handlers

import "net/http"

// MarkNewsRead отмечает всю ленту прочитанной.
//
// Отметка одна на пользователя, а не строка на каждую новость: лента показывает
// «новое с прошлого захода», и для этого достаточно сравнить дату публикации с
// этой датой. Таблица связей дала бы то же самое, но росла бы как число
// пользователей на число новостей.
//
// Время берём у базы, а не у клиента: часы браузера могут отставать, и новость,
// вышедшая минуту назад, осталась бы непрочитанной навсегда.
func (h *Handler) MarkNewsRead(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if _, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.users SET news_read_at = now() WHERE id = $1 AND tenant_id = $2
	`, claims.UserID, claims.TenantID); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось отметить новости прочитанными")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
