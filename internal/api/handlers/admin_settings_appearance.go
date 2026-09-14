package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
)

const (
	defaultPanelAccent  = "#6366f1"
	appearanceCSSLimit  = 64 << 10
	appearanceTextLimit = 300
	appearanceNameLimit = 60
	appearanceAssetSize = 5 << 20
)

var (
	appearanceHexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
	appearanceFonts    = []string{"default", "inter", "manrope", "rubik", "montserrat", "ibm-plex-sans"}
	appearanceRadii    = []string{"standard", "strict", "soft"}
	appearanceThemes   = []string{"dark", "light", "system"}
	appearanceMenus    = []string{"default", "screenshot"}
	landingBlockNames  = []string{"hero", "games", "steps", "hardware", "panel", "locations", "faq", "pricing", "cta"}
	landingHeroFields  = []string{"badge", "title", "subtitle", "cta_primary", "cta_secondary"}
	landingLinkFields  = []string{"telegram", "discord", "vk", "support", "email", "offer", "privacy"}
	appearanceAssets   = map[string]string{
		"logo":      "app.branding.logo",
		"logo_dark": "app.branding.logo_dark",
		"icon":      "app.branding.icon",
	}
)

type appearanceAccent struct {
	Panel       string `json:"panel"`
	PanelDark   string `json:"panel_dark"`
	Landing     string `json:"landing"`
	LandingDark string `json:"landing_dark"`
}

type appearanceSettings struct {
	Name        string            `json:"name"`
	Logo        string            `json:"logo"`
	LogoDark    string            `json:"logo_dark"`
	Icon        string            `json:"icon"`
	Accent      appearanceAccent  `json:"accent"`
	Theme       string            `json:"theme"`
	ThemeLocked bool              `json:"theme_locked"`
	UserMenu    string            `json:"user_menu"`
	Radius      string            `json:"radius"`
	FontPanel   string            `json:"font_panel"`
	FontLanding string            `json:"font_landing"`
	CustomCSS   string            `json:"custom_css"`
	Blocks      map[string]bool   `json:"blocks"`
	Hero        map[string]string `json:"hero"`
	Links       map[string]string `json:"links"`
}

func appearanceChoice(value string, allowed []string) (string, bool) {
	value = strings.TrimSpace(value)
	for _, a := range allowed {
		if value == a {
			return value, true
		}
	}
	return allowed[0], false
}

func appearanceHex(value string) string {
	value = strings.TrimSpace(value)
	if appearanceHexColor.MatchString(value) {
		return strings.ToLower(value)
	}
	return ""
}

func appearanceFromSettings(s map[string]string) appearanceSettings {
	theme, _ := appearanceChoice(s["appearance.theme"], appearanceThemes)
	menu, _ := appearanceChoice(s["app.site.template.user_menu_variant"], appearanceMenus)
	radius, _ := appearanceChoice(s["appearance.radius"], appearanceRadii)
	fontPanel, _ := appearanceChoice(s["appearance.font.panel"], appearanceFonts)
	fontLanding, _ := appearanceChoice(s["appearance.font.landing"], appearanceFonts)

	a := appearanceSettings{
		Name:     strings.TrimSpace(s["app.name"]),
		Logo:     strings.TrimSpace(s["app.branding.logo"]),
		LogoDark: strings.TrimSpace(s["app.branding.logo_dark"]),
		Icon:     strings.TrimSpace(s["app.branding.icon"]),
		Accent: appearanceAccent{
			Panel:       appearanceHex(s["appearance.accent.panel"]),
			PanelDark:   appearanceHex(s["appearance.accent.panel_dark"]),
			Landing:     appearanceHex(s["appearance.accent.landing"]),
			LandingDark: appearanceHex(s["appearance.accent.landing_dark"]),
		},
		Theme:       theme,
		ThemeLocked: s["appearance.theme_locked"] == "1",
		UserMenu:    menu,
		Radius:      radius,
		FontPanel:   fontPanel,
		FontLanding: fontLanding,
		CustomCSS:   s["appearance.custom_css"],
		Blocks:      map[string]bool{},
		Hero:        map[string]string{},
		Links:       map[string]string{},
	}
	stored := jsonObjectOfBools(s["app.site.template.blocks"])
	for _, name := range landingBlockNames {
		v, ok := stored[name]
		a.Blocks[name] = !ok || v
	}
	hero := jsonObjectOfStrings(s["appearance.landing.hero"])
	for _, f := range landingHeroFields {
		a.Hero[f] = hero[f]
	}
	for _, f := range landingLinkFields {
		a.Links[f] = strings.TrimSpace(s["app.links."+f])
	}
	return a
}

func normalizeAppearance(a *appearanceSettings) string {
	a.Name = strings.TrimSpace(a.Name)
	if utf8.RuneCountInString(a.Name) > appearanceNameLimit {
		return "название не длиннее 60 символов"
	}
	for _, c := range []*string{&a.Accent.Panel, &a.Accent.PanelDark, &a.Accent.Landing, &a.Accent.LandingDark} {
		*c = strings.TrimSpace(*c)
		if *c != "" && !appearanceHexColor.MatchString(*c) {
			return "цвет задаётся в формате #RRGGBB"
		}
		*c = strings.ToLower(*c)
	}
	for _, c := range []struct {
		value   *string
		allowed []string
	}{
		{&a.Theme, appearanceThemes},
		{&a.UserMenu, appearanceMenus},
		{&a.Radius, appearanceRadii},
		{&a.FontPanel, appearanceFonts},
		{&a.FontLanding, appearanceFonts},
	} {
		v, ok := appearanceChoice(*c.value, c.allowed)
		if !ok {
			return "недопустимое значение настройки оформления"
		}
		*c.value = v
	}
	if len(a.CustomCSS) > appearanceCSSLimit {
		return "свой CSS не больше 64 КБ"
	}
	for _, f := range landingHeroFields {
		if utf8.RuneCountInString(strings.TrimSpace(a.Hero[f])) > appearanceTextLimit {
			return "тексты лендинга не длиннее 300 символов"
		}
	}
	for _, f := range landingLinkFields {
		v := strings.TrimSpace(a.Links[f])
		if v == "" {
			continue
		}
		if utf8.RuneCountInString(v) > appearanceTextLimit {
			return "ссылки не длиннее 300 символов"
		}
		if f == "email" {
			if !strings.Contains(v, "@") || strings.ContainsAny(v, " <>\"'") {
				return "укажите почту в виде name@example.com"
			}
			continue
		}
		if !strings.HasPrefix(v, "https://") && !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "/") {
			return "ссылки должны начинаться с https:// или с /"
		}
	}
	return ""
}

func (h *Handler) GetAdminAppearance(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"appearance": appearanceFromSettings(h.loadTenantSettingStrings(r.Context())),
	})
}

func (h *Handler) UpdateAdminAppearance(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	r.Body = http.MaxBytesReader(w, r.Body, appearanceCSSLimit+(64<<10))

	var body appearanceSettings
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if msg := normalizeAppearance(&body); msg != "" {
		writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}

	locked := "0"
	if body.ThemeLocked {
		locked = "1"
	}
	values := map[string]string{
		"app.name":                            body.Name,
		"appearance.accent.panel":             body.Accent.Panel,
		"appearance.accent.panel_dark":        body.Accent.PanelDark,
		"appearance.accent.landing":           body.Accent.Landing,
		"appearance.accent.landing_dark":      body.Accent.LandingDark,
		"appearance.theme":                    body.Theme,
		"appearance.theme_locked":             locked,
		"app.site.template.user_menu_variant": body.UserMenu,
		"appearance.radius":                   body.Radius,
		"appearance.font.panel":               body.FontPanel,
		"appearance.font.landing":             body.FontLanding,
		"appearance.custom_css":               body.CustomCSS,
	}

	blocks := map[string]bool{}
	for _, name := range landingBlockNames {
		v, ok := body.Blocks[name]
		blocks[name] = !ok || v
	}
	rawBlocks, _ := json.Marshal(blocks)
	values["app.site.template.blocks"] = string(rawBlocks)

	hero := map[string]string{}
	for _, f := range landingHeroFields {
		if v := strings.TrimSpace(body.Hero[f]); v != "" {
			hero[f] = v
		}
	}
	rawHero, _ := json.Marshal(hero)
	values["appearance.landing.hero"] = string(rawHero)

	for _, f := range landingLinkFields {
		values["app.links."+f] = strings.TrimSpace(body.Links[f])
	}
	for key, value := range values {
		h.setTenantSettingString(ctx, key, value)
	}

	audit(ctx, h.dbOf(ctx), claims.UserID, "settings.appearance", "appearance", nil)
	writeJSON(w, http.StatusOK, map[string]any{
		"appearance": appearanceFromSettings(h.loadTenantSettingStrings(ctx)),
	})
}

func (h *Handler) UploadAdminAppearanceAsset(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	kind := chi.URLParam(r, "kind")
	key, ok := appearanceAssets[kind]
	if !ok {
		writeError(w, http.StatusNotFound, "unknown asset")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, appearanceAssetSize+(1<<20))
	if err := r.ParseMultipartForm(appearanceAssetSize); err != nil {
		writeError(w, http.StatusBadRequest, "файл больше 5 МБ или повреждён")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "выберите файл")
		return
	}
	defer file.Close()

	base := strings.ReplaceAll(kind, "_", "-") + "-" + strconv.FormatInt(time.Now().Unix(), 36)
	path, err := h.saveBrandingFile(file, header.Filename, base)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "поддерживаются PNG, JPG, WEBP, SVG и ICO")
		return
	}
	previous := h.tenantSettingString(ctx, key)
	h.setTenantSettingString(ctx, key, path)
	h.removeBrandingFile(previous, path)

	audit(ctx, h.dbOf(ctx), claims.UserID, "settings.appearance.asset", kind, nil)
	writeJSON(w, http.StatusOK, map[string]any{
		"appearance": appearanceFromSettings(h.loadTenantSettingStrings(ctx)),
	})
}

func (h *Handler) DeleteAdminAppearanceAsset(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	kind := chi.URLParam(r, "kind")
	key, ok := appearanceAssets[kind]
	if !ok {
		writeError(w, http.StatusNotFound, "unknown asset")
		return
	}
	previous := h.tenantSettingString(ctx, key)
	h.setTenantSettingString(ctx, key, "")
	h.removeBrandingFile(previous, "")

	audit(ctx, h.dbOf(ctx), claims.UserID, "settings.appearance.asset_delete", kind, nil)
	writeJSON(w, http.StatusOK, map[string]any{
		"appearance": appearanceFromSettings(h.loadTenantSettingStrings(ctx)),
	})
}

func (h *Handler) removeBrandingFile(previous, keep string) {
	previous = strings.TrimSpace(previous)
	if previous == "" || previous == keep || !strings.HasPrefix(previous, "branding/") {
		return
	}
	name := strings.TrimPrefix(previous, "branding/")
	if name == "" || strings.Contains(name, "..") || strings.ContainsAny(name, `/\`) {
		return
	}
	_ = os.Remove(filepath.Join(h.uploadDir, "branding", name))
}
