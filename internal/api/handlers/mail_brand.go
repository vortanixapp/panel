package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/vortanixapp/panel/pkg/mailtpl"
)

func (h *Handler) mailBrand(ctx context.Context, r *http.Request) mailtpl.Brand {
	a := appearanceFromSettings(h.loadTenantSettingStrings(ctx))
	logo := h.brandingPublicURL(r, a.Logo)
	if logo != "" && !strings.HasPrefix(logo, "http://") && !strings.HasPrefix(logo, "https://") {
		if base := strings.TrimRight(h.frontendURL, "/"); base != "" {
			logo = base + "/" + strings.TrimLeft(logo, "/")
		}
	}
	return mailtpl.Brand{Name: a.Name, LogoURL: logo, Accent: a.Accent.Panel}
}
