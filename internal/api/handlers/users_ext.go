package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) VerifyUserEmail(w http.ResponseWriter, r *http.Request) {

	claims, ok := tenantClaims(r.Context())

	if !ok {

		return

	}

	id := chi.URLParam(r, "id")

	tag, err := h.dbOf(r.Context()).Exec(r.Context(), `

		UPDATE core.users SET email_verified_at = COALESCE(email_verified_at, now())

		WHERE id = $1 AND tenant_id = $2

	`, id, claims.TenantID)

	if err != nil {

		writeError(w, http.StatusInternalServerError, "database error")

		return

	}

	if tag.RowsAffected() == 0 {

		writeError(w, http.StatusNotFound, "not found")

		return

	}

	writeJSON(w, http.StatusOK, map[string]any{

		"ok": true,

		"status": "verified",

		"message": "Email пользователя подтвержден.",
	})

}

func (h *Handler) ImpersonateUser(w http.ResponseWriter, r *http.Request) {

	claims, ok := tenantClaims(r.Context())

	if !ok {

		return

	}

	id := chi.URLParam(r, "id")

	var email, role string

	err := h.dbOf(r.Context()).QueryRow(r.Context(), `SELECT email, role FROM core.users WHERE id = $1 AND tenant_id = $2`, id, claims.TenantID).Scan(&email, &role)

	if err != nil {

		writeError(w, http.StatusNotFound, "not found")

		return

	}

	if isStaffRole(role) {

		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{

			"ok": false,

			"message": "Нельзя войти под администратором.",
		})

		return

	}

	access, refresh, err := h.issueAuthTokens(r, claims.TenantID, claims.TenantSlug, id, email, role, "", 0)

	if err != nil {

		writeError(w, http.StatusInternalServerError, "token error")

		return

	}

	writeJSON(w, http.StatusOK, map[string]any{

		"ok": true,

		"token": access,

		"access_token": access,

		"refresh_token": refresh,

		"impersonator_id": claims.UserID,

		"message": "Вы вошли как пользователь: " + email,
	})

}

func (h *Handler) userPermissions(ctx context.Context, tenantID, role string) map[string]bool {
	return h.rbacLoadRolePermissions(ctx, tenantID, rbacNormalizeRole(role))
}
