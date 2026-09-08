package handlers

import (
	"encoding/json"

	"net/http"

	"strings"

	"github.com/go-chi/chi/v5"

	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())

	if !ok {
		return

	}

	var req struct {
		Name string `json:"name"`

		Email string `json:"email"`

		Password string `json:"password"`

		PasswordConfirmation string `json:"password_confirmation"`

		Role string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")

		return

	}

	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")

		return

	}

	if req.PasswordConfirmation != "" && req.PasswordConfirmation != req.Password {
		writeError(w, http.StatusBadRequest, "password confirmation mismatch")

		return

	}

	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")

		return

	}

	role := req.Role

	if role == "" {
		role = "user"

	}

	if role != "user" && role != "admin" && role != "support" {
		writeError(w, http.StatusBadRequest, "invalid role")

		return

	}

	if claims.Role != "owner" && (role == "admin") {
		writeError(w, http.StatusForbidden, "only owner can create admins")

		return

	}

	if role == "admin" && !h.checkAdminQuota(r.Context(), claims.TenantID, w) {
		return

	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")

		return

	}

	email := strings.ToLower(strings.TrimSpace(req.Email))

	var userID string

	err = h.dbOf(r.Context()).QueryRow(r.Context(), `

		INSERT INTO core.users (tenant_id, email, password_hash, role)

		VALUES ($1, $2, $3, $4)

		RETURNING id::text

	`, claims.TenantID, email, string(hash), role).Scan(&userID)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			writeError(w, http.StatusConflict, "email already registered")

			return

		}

		writeError(w, http.StatusInternalServerError, "failed to create user")

		return

	}

	if name := strings.TrimSpace(req.Name); name != "" {
		_, _ = h.dbOf(r.Context()).Exec(r.Context(), `

			INSERT INTO core.user_profiles (user_id, display_name, first_name, updated_at)

			VALUES ($1, $2, $2, now())

			ON CONFLICT (user_id) DO UPDATE SET display_name = EXCLUDED.display_name, first_name = EXCLUDED.first_name, updated_at = now()

		`, userID, name)

	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"ok": true,

		"id": userID,

		"email": email,

		"role": role,
	})

}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())

	if !ok {
		return

	}

	userID := chi.URLParam(r, "id")

	if userID == "" {
		writeError(w, http.StatusBadRequest, "id is required")

		return

	}

	if userID == claims.UserID {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"ok": false,

			"error": "Нельзя удалить собственный аккаунт.",
		})

		return

	}

	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `

		DELETE FROM core.users

		WHERE id = $1 AND tenant_id = $2 AND role != 'owner'

	`, userID, claims.TenantID)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")

		return

	}

	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "user not found")

		return

	}

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "deleted"})

}
