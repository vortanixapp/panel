package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (h *Handler) GetBranding(w http.ResponseWriter, r *http.Request) {
	slug := r.Header.Get("X-Tenant-Slug")
	if slug == "" {
		slug = envOr("DEFAULT_TENANT_SLUG", "dev")
	}
	defaults := map[string]string{
		"brand_name":    "VORTANIX",
		"logo_url":      "",
		"primary_color": "#6366f1",
	}
	var tenantID string
	err := h.dbOf(r.Context()).QueryRow(r.Context(), `SELECT id::text FROM core.tenants WHERE slug = $1`, slug).Scan(&tenantID)
	if err != nil {
		writeJSON(w, http.StatusOK, defaults)
		return
	}

	ctx := r.Context()
	settings := map[string]string{}
	for _, k := range []string{
		"brand.name", "brand.logo_url", "brand.primary_color",
		"app.name", "app.branding.logo", "app.site.template.colors",
		"app.site.template.blocks", "app.site.template.user_menu_variant",
	} {
		settings[k] = h.tenantSettingString(ctx, tenantID, k)
	}

	result := map[string]string{
		"brand_name":    defaults["brand_name"],
		"logo_url":      defaults["logo_url"],
		"primary_color": defaults["primary_color"],
	}

	if v := strings.TrimSpace(settings["brand.name"]); v != "" {
		result["brand_name"] = strings.ToUpper(v)
	} else if v := strings.TrimSpace(settings["app.name"]); v != "" {
		result["brand_name"] = strings.ToUpper(v)
	}

	if v := strings.TrimSpace(settings["brand.logo_url"]); v != "" {
		result["logo_url"] = v
	} else if v := brandingLogoFromSetting(settings["app.branding.logo"]); v != "" {
		result["logo_url"] = h.brandingPublicURL(r, v)
	}

	if v := strings.TrimSpace(settings["brand.primary_color"]); v != "" {
		result["primary_color"] = v
	} else if v := primaryColorFromTemplate(settings["app.site.template.colors"]); v != "" {
		result["primary_color"] = v
	}

	variant := strings.TrimSpace(settings["app.site.template.user_menu_variant"])
	if variant != "screenshot" {
		variant = "default"
	}
	out := map[string]any{
		"brand_name":        result["brand_name"],
		"logo_url":          result["logo_url"],
		"primary_color":     result["primary_color"],
		"colors":            jsonObjectOfStrings(settings["app.site.template.colors"]),
		"blocks":            jsonObjectOfBools(settings["app.site.template.blocks"]),
		"user_menu_variant": variant,
	}

	writeJSON(w, http.StatusOK, out)
}

func jsonObjectOfStrings(raw string) map[string]string {
	out := map[string]string{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out
	}
	var parsed map[string]any
	if json.Unmarshal([]byte(raw), &parsed) != nil {
		return out
	}
	for k, v := range parsed {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			out[k] = strings.TrimSpace(s)
		}
	}
	return out
}

func jsonObjectOfBools(raw string) map[string]bool {
	out := map[string]bool{}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return out
	}
	var parsed map[string]any
	if json.Unmarshal([]byte(raw), &parsed) != nil {
		return out
	}
	for k, v := range parsed {
		switch val := v.(type) {
		case bool:
			out[k] = val
		case string:
			out[k] = val == "true" || val == "1" || val == "on"
		case float64:
			out[k] = val != 0
		}
	}
	return out
}

func brandingLogoFromSetting(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	return raw
}

func primaryColorFromTemplate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var parsed map[string]any
	if json.Unmarshal([]byte(raw), &parsed) != nil {
		return ""
	}
	if v, ok := parsed["primary"].(string); ok {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}
