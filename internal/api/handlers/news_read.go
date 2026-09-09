package handlers

import "net/http"

func (h *Handler) MarkNewsRead(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if _, err := h.dbOf(r.Context()).Exec(r.Context(), `
		UPDATE core.users SET news_read_at = now() WHERE id = $1
	`, claims.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось отметить новости прочитанными")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
