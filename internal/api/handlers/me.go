package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/vortanixapp/panel/pkg/notify"
	"golang.org/x/crypto/bcrypt"
)

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		writeError(w, http.StatusBadRequest, "current_password and new_password are required")
		return
	}
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	ctx := r.Context()
	var hash string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT password_hash FROM core.users
		WHERE id = $1 AND status = 'active'
	`, claims.UserID).Scan(&hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.CurrentPassword)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid current password")
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	_, err = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.users SET password_hash = $1
		WHERE id = $2
	`, string(newHash), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	closed := int64(0)
	if tag, err := h.dbOf(ctx).Exec(ctx, `
		DELETE FROM core.user_sessions
		WHERE user_id = $1 AND id != $2
	`, claims.UserID, claims.SessionID); err != nil {
		log.Printf("смена пароля %s: чужие сеансы не закрыты: %v", claims.UserID, err)
	} else {
		closed = tag.RowsAffected()
	}

	audit(ctx, h.dbOf(ctx), claims.UserID, "user.password_change", "user:"+claims.UserID,
		map[string]any{"sessions_closed": closed})
	h.notifyUser(ctx, claims.UserID, notify.Event{
		Kind:   notify.KindPasswordChange,
		Title:  "Пароль изменён",
		Body:   "Пароль вашей учётной записи только что изменён, остальные сеансы завершены. Если это были не вы, восстановите доступ через сброс пароля.",
		Action: h.panelAction("Проверить сеансы", "/settings"),
	})

	writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "sessions_closed": closed})
}
