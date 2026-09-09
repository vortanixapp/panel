package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
)

var appearanceSettingKeys = []string{
	"app.site.default_template",
	"app.site.template.colors",
	"app.site.template.blocks",
	"app.site.template.user_menu_variant",
}

func (h *Handler) GetAdminAppearance(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	values := map[string]string{}
	for _, key := range appearanceSettingKeys {
		values[key] = h.tenantSettingString(ctx, claims.TenantID, key)
	}
	if values["app.site.template.user_menu_variant"] == "" {
		values["app.site.template.user_menu_variant"] = "default"
	}
	writeJSON(w, http.StatusOK, map[string]any{"values": values})
}

func (h *Handler) UpdateAdminAppearance(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()

	var body struct {
		DefaultTemplate         string `json:"default_template"`
		TemplateColors          string `json:"template_colors"`
		TemplateBlocks          string `json:"template_blocks"`
		TemplateUserMenuVariant string `json:"template_user_menu_variant"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	variant := strings.TrimSpace(body.TemplateUserMenuVariant)
	if variant == "" {
		variant = "default"
	}
	if variant != "default" && variant != "screenshot" {
		writeError(w, http.StatusBadRequest, "invalid template_user_menu_variant")
		return
	}

	h.setTenantSettingString(ctx, claims.TenantID, "app.site.default_template", strings.TrimSpace(body.DefaultTemplate))
	h.setTenantSettingString(ctx, claims.TenantID, "app.site.template.colors", body.TemplateColors)
	h.setTenantSettingString(ctx, claims.TenantID, "app.site.template.blocks", body.TemplateBlocks)
	h.setTenantSettingString(ctx, claims.TenantID, "app.site.template.user_menu_variant", variant)

	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": "Оформление сохранено"})
}
