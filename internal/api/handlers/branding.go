package handlers

import (
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
		"user_menu_variant": a.UserMenu,
		"appearance": map[string]any{
			"accent":       a.Accent,
			"theme":        a.Theme,
			"theme_locked": a.ThemeLocked,
			"radius":       a.Radius,
			"font_panel":   a.FontPanel,
			"font_landing": a.FontLanding,
			"custom_css":   a.CustomCSS,
			"links":        a.Links,
		},
	}
	writeJSON(w, http.StatusOK, payload)
}
