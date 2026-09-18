package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (h *Handler) GetBranding(w http.ResponseWriter, r *http.Request) {
	settings := h.loadTenantSettingStrings(r.Context())
	a := appearanceFromSettings(settings)

	title := firstNonEmpty(
		strings.TrimSpace(settings["brand.name"]),
		a.Name,
		strings.TrimSpace(settings["panel.name"]),
		"Vortanix",
	)
	name := strings.ToUpper(title)
	logo := strings.TrimSpace(settings["brand.logo_url"])
	if logo == "" {
		logo = h.brandingPublicURL(r, a.Logo)
	}
	primary := a.Accent.Panel
	if primary == "" {
		primary = defaultPanelAccent
	}

	payload := map[string]any{
		"whmcs":             whmcsPublicInfo(settings),
		"legal":             h.publicLegalInfo(r.Context(), settings),
		"brand_name":        name,
		"panel_name":        title,
		"logo_url":          logo,
		"logo_dark_url":     h.brandingPublicURL(r, a.LogoDark),
		"icon_url":          h.brandingPublicURL(r, a.Icon),
		"primary_color":     primary,
		"blocks":            a.Blocks,
		"user_menu_variant": a.UserMenu,
		"appearance": map[string]any{
			"accent":       a.Accent,
			"theme":        a.Theme,
			"theme_locked": a.ThemeLocked,
			"radius":       a.Radius,
			"font_panel":   a.FontPanel,
			"font_landing": a.FontLanding,
			"custom_css":   a.CustomCSS,
			"hero":         a.Hero,
			"links":        a.Links,
		},
	}
	writeJSON(w, http.StatusOK, payload)
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
