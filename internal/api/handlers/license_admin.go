package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (h *Handler) LicenseState(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	st := h.licenseFor(r.Context())

	servers, nodes, admins := h.tenantCounts(r, claims.TenantID)

	writeJSON(w, http.StatusOK, map[string]any{
		"state":              string(st.Mode),
		"license_status":     st.LicenseStatus,
		"plan":               st.Plan,
		"key_hint":           st.KeyHint,
		"legacy":             st.Legacy,
		"activated":          st.Activated,
		"limits":             st.Limits,
		"usage":              map[string]int{"servers": servers, "nodes": nodes, "admins": admins},
		"license_expires_at": st.LicenseExpiresAt,
		"grace_until":        nullableTime(st.GraceUntil),
		"last_verified_at":   nullableTime(st.LastVerifiedAt),
		"message":            st.Reason,
	})
}

func (h *Handler) AdminLicense(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	if !requireAdmin(w, claims) {
		return
	}

	ctx := r.Context()
	st := h.licenseFor(r.Context())

	var slug, name string
	_ = h.dbOf(ctx).QueryRow(ctx, `SELECT slug, name FROM core.tenants WHERE id = $1`, claims.TenantID).Scan(&slug, &name)
	servers, nodes, admins := h.tenantCounts(r, claims.TenantID)

	writeJSON(w, http.StatusOK, map[string]any{
		"tenant_slug":        slug,
		"tenant_name":        name,
		"plan":               st.Plan,
		"installation_id":    st.InstallationID,
		"state":              string(st.Mode),
		"license_status":     st.LicenseStatus,
		"key_hint":           st.KeyHint,
		"legacy":             st.Legacy,
		"activated":          st.Activated,
		"revision":           st.Revision,
		"limits":             st.Limits,
		"effective_limits":   st.Limits,
		"api_write_rpm":      st.Limits.APIRPM,
		"license_expires_at": st.LicenseExpiresAt,
		"token_expires_at":   nullableTime(st.TokenExpiresAt),
		"grace_until":        nullableTime(st.GraceUntil),
		"last_verified_at":   nullableTime(st.LastVerifiedAt),
		"last_error":         st.LastError,
		"message":            st.Reason,
		"usage": map[string]int{
			"nodes":   nodes,
			"servers": servers,
			"admins":  admins,
		},
	})
}

type bindLicenseRequest struct {
	LicenseKey string `json:"license_key"`
}

func (h *Handler) AdminBindLicenseKey(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	if !requireAdmin(w, claims) {
		return
	}
	if h.licenseOps == nil {
		writeError(w, http.StatusServiceUnavailable, "подсистема лицензий не готова")
		return
	}

	var req bindLicenseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.LicenseKey = strings.TrimSpace(req.LicenseKey)
	if req.LicenseKey == "" {
		writeError(w, http.StatusBadRequest, "license_key обязателен")
		return
	}

	domain := h.licenseFor(r.Context()).Domain
	if domain == "" {
		domain = r.Host
	}

	if err := h.licenseOps.Activate(r.Context(), req.LicenseKey, domain); err != nil {
		writeCodedError(w, http.StatusForbidden, "license_invalid", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"state": string(h.licenseFor(r.Context()).Mode)})
}

func (h *Handler) AdminLicenseRefresh(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	if !requireAdmin(w, claims) {
		return
	}
	if h.licenseOps == nil {
		writeError(w, http.StatusServiceUnavailable, "подсистема лицензий не готова")
		return
	}

	h.licenseOps.RefreshNow(r.Context())
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "проверка запущена"})
}

func (h *Handler) tenantCounts(r *http.Request, tenantID string) (servers, nodes, admins int) {
	_ = h.dbOf(r.Context()).QueryRow(r.Context(), `
		SELECT
			(SELECT COUNT(*) FROM core.servers WHERE tenant_id = $1),
			(SELECT COUNT(*) FROM core.nodes   WHERE tenant_id = $1),
			(SELECT COUNT(*) FROM core.users
			 WHERE tenant_id = $1 AND role IN ('owner','admin') AND status = 'active')
	`, tenantID).Scan(&servers, &nodes, &admins)
	return servers, nodes, admins
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t
}
