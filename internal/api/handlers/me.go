package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/vortanixapp/panel/pkg/i18n"
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
	if len(req.NewPassword) < 8 {
		writeError(w, http.StatusBadRequest, "Пароль должен быть не короче 8 символов")
		return
	}
	if len(req.NewPassword) > 128 {
		writeError(w, http.StatusBadRequest, "Пароль должен быть не длиннее 128 символов")
		return
	}

	ctx := r.Context()
	var hash string
	var passwordSet bool
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT password_hash, password_set FROM core.users
		WHERE id = $1 AND status = 'active'
	`, claims.UserID).Scan(&hash, &passwordSet)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "Учётная запись не найдена")
			return
		}
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	if !h.checkAccountPassword(w, r, claims.UserID, passwordSet, hash, req.CurrentPassword) {
		return
	}
	if passwordSet && req.CurrentPassword == req.NewPassword {
		writeError(w, http.StatusBadRequest, "Новый пароль совпадает с текущим")
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	_, err = h.dbOf(ctx).Exec(ctx, `
		UPDATE core.users SET password_hash = $1, password_set = true
		WHERE id = $2
	`, string(newHash), claims.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Не удалось сохранить пароль")
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
		Title:  i18n.Key("notify.password_changed.title"),
		Body:   i18n.Key("notify.password_changed.body"),
		Action: h.panelAction("notify.action.sessions", "/settings?tab=sessions"),
	})

	writeJSON(w, http.StatusOK, map[string]any{"status": "updated", "sessions_closed": closed, "was_set": passwordSet})
}
