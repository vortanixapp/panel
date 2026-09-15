package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	whmcsmodule "github.com/vortanixapp/panel/integrations/whmcs"
	"github.com/vortanixapp/panel/pkg/buildinfo"
)

const (
	whmcsURLSetting        = "whmcs.url"
	whmcsOrderURLSetting   = "whmcs.order_url"
	whmcsOrdersOnlySetting = "whmcs.orders_only"
	whmcsKeyScope          = "admin.whmcs.write"
)

func (h *Handler) AdminWHMCSSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var total, active, suspended, keys int
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COUNT(*) FILTER (WHERE status <> 'terminated'),
		       COUNT(*) FILTER (WHERE status = 'active'),
		       COUNT(*) FILTER (WHERE status = 'suspended')
		FROM core.whmcs_services
	`).Scan(&total, &active, &suspended)
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COUNT(*) FROM core.api_keys
		WHERE revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > now())
		  AND scopes ? $1
	`, whmcsKeyScope).Scan(&keys)

	writeJSON(w, http.StatusOK, map[string]any{
		"url":            h.tenantSettingString(ctx, whmcsURLSetting),
		"order_url":      h.tenantSettingString(ctx, whmcsOrderURLSetting),
		"orders_only":    truthySetting(h.tenantSettingString(ctx, whmcsOrdersOnlySetting)),
		"panel_url":      h.frontendURL,
		"scope":          whmcsKeyScope,
		"active_keys":    keys,
		"module_version": buildinfo.Current(),
		"services":       map[string]int{"total": total, "active": active, "suspended": suspended},
	})
}

func (h *Handler) AdminWHMCSSettingsUpdate(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body struct {
		URL        string `json:"url"`
		OrderURL   string `json:"order_url"`
		OrdersOnly bool   `json:"orders_only"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	base, err := normalizeHTTPURL(body.URL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "адрес WHMCS должен начинаться с http:// или https://")
		return
	}
	order, err := normalizeHTTPURL(body.OrderURL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "ссылка на заказ должна начинаться с http:// или https://")
		return
	}
	if body.OrdersOnly && whmcsOrderURL(base, order) == "" {
		writeError(w, http.StatusBadRequest, "чтобы принимать заказы только в WHMCS, укажите адрес WHMCS")
		return
	}

	ctx := r.Context()
	h.setTenantSettingString(ctx, whmcsURLSetting, base)
	h.setTenantSettingString(ctx, whmcsOrderURLSetting, order)
	h.setTenantSettingString(ctx, whmcsOrdersOnlySetting, strconv.FormatBool(body.OrdersOnly))
	audit(ctx, h.dbOf(ctx), claims.UserID, "settings.update", "whmcs", map[string]any{
		"url": base, "order_url": order, "orders_only": body.OrdersOnly,
	})
	h.AdminWHMCSSettings(w, r)
}

func (h *Handler) AdminWHMCSModule(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	err := fs.WalkDir(whmcsmodule.Module, "modules", func(name string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() {
			return walkErr
		}
		data, err := whmcsmodule.Module.ReadFile(name)
		if err != nil {
			return err
		}
		f, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate, Modified: time.Now()})
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		return err
	})
	if err == nil {
		err = zw.Close()
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось собрать архив модуля")
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="vortanix-whmcs-module.zip"`)
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

func (h *Handler) WHMCSSSOExchange(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	token := strings.TrimSpace(body.Token)
	if token == "" {
		writeError(w, http.StatusBadRequest, "в ссылке нет ключа входа")
		return
	}

	ctx := r.Context()
	if blocked, reason := h.ipBlocked(ctx, clientIP(r)); blocked {
		msg := "доступ с этого адреса заблокирован"
		if reason != "" {
			msg += ": " + reason
		}
		writeError(w, http.StatusForbidden, msg)
		return
	}

	var userID, redirect string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		UPDATE core.whmcs_sso_tokens SET used_at = now()
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > now()
		RETURNING user_id::text, redirect
	`, sha256Hex(token)).Scan(&userID, &redirect); err != nil {
		writeError(w, http.StatusUnauthorized, "ссылка для входа устарела — откройте панель из WHMCS ещё раз")
		return
	}

	var email, role string
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT email, role FROM core.users WHERE id = $1 AND status = 'active'
	`, userID).Scan(&email, &role); err != nil {
		writeError(w, http.StatusUnauthorized, "учётная запись недоступна")
		return
	}
	if isStaffRole(role) {
		writeError(w, http.StatusForbidden, "вход под сотрудником панели из WHMCS запрещён")
		return
	}

	access, refresh, err := h.issueAuthTokens(r, userID, email, role, "", 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue tokens")
		return
	}
	h.recordLoginAttempt(ctx, r, userID, email, "", true)
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  access,
		"refresh_token": refresh,
		"redirect":      whmcsRedirect(redirect, ""),
		"user":          map[string]string{"id": userID, "email": email, "role": role},
	})
}

func normalizeHTTPURL(raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return "", nil
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errors.New("invalid url")
	}
	return raw, nil
}

func whmcsOrderURL(base, order string) string {
	if order = strings.TrimSpace(order); order != "" {
		return order
	}
	if base = strings.TrimRight(strings.TrimSpace(base), "/"); base != "" {
		return base + "/cart.php"
	}
	return ""
}

func whmcsPublicInfo(settings map[string]string) map[string]any {
	base := strings.TrimRight(strings.TrimSpace(settings[whmcsURLSetting]), "/")
	order := whmcsOrderURL(base, settings[whmcsOrderURLSetting])
	if order == "" {
		return nil
	}
	info := map[string]any{
		"order_url":       order,
		"client_area_url": "",
		"orders_only":     truthySetting(settings[whmcsOrdersOnlySetting]),
	}
	if base != "" {
		info["client_area_url"] = base + "/clientarea.php"
	}
	return info
}

func (h *Handler) whmcsOrdersOnly(ctx context.Context) bool {
	if !truthySetting(h.tenantSettingString(ctx, whmcsOrdersOnlySetting)) {
		return false
	}
	return whmcsOrderURL(h.tenantSettingString(ctx, whmcsURLSetting), h.tenantSettingString(ctx, whmcsOrderURLSetting)) != ""
}

func (h *Handler) whmcsServiceURL(ctx context.Context, serviceID int64) string {
	base := strings.TrimRight(strings.TrimSpace(h.tenantSettingString(ctx, whmcsURLSetting)), "/")
	if base == "" {
		return ""
	}
	return base + "/clientarea.php?action=productdetails&id=" + strconv.FormatInt(serviceID, 10)
}

func (h *Handler) refuseWHMCSBilled(ctx context.Context, w http.ResponseWriter, serverID string) bool {
	var source string
	if h.dbOf(ctx).QueryRow(ctx, `
		SELECT billing_source FROM core.servers WHERE id = $1
	`, serverID).Scan(&source) != nil || source != "whmcs" {
		return false
	}
	writeCodedError(w, http.StatusConflict, "billed_in_whmcs",
		"оплата этого сервера ведётся в WHMCS — продление, смена тарифа и удаление выполняются там")
	return true
}

func (h *Handler) attachWHMCSBilling(ctx context.Context, serverID string, item map[string]any) {
	item["billing_source"] = "panel"
	var serviceID int64
	var status, reason string
	var due *time.Time
	if err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT w.service_id, w.status, COALESCE(w.suspend_reason, ''), w.next_due_date
		FROM core.servers s
		JOIN core.whmcs_services w ON w.server_id = s.id
		WHERE s.id = $1 AND s.billing_source = 'whmcs'
	`, serverID).Scan(&serviceID, &status, &reason, &due); err != nil {
		return
	}
	item["billing_source"] = "whmcs"
	item["whmcs"] = map[string]any{
		"service_id":     serviceID,
		"status":         status,
		"suspend_reason": nilIfEmpty(reason),
		"next_due_date":  dateOrNil(due),
		"manage_url":     nilIfEmpty(h.whmcsServiceURL(ctx, serviceID)),
	}
}
