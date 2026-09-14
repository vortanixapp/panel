package jobs

import (
	"context"
	"strings"

	"github.com/vortanixapp/panel/pkg/mailtpl"
)

func (r *Runner) mailBrand(ctx context.Context) mailtpl.Brand {
	brand := mailtpl.Brand{
		Name:   r.updateSetting(ctx, "app.name"),
		Accent: r.updateSetting(ctx, "appearance.accent.panel"),
	}
	logo := strings.TrimSpace(r.updateSetting(ctx, "app.branding.logo"))
	name := strings.TrimPrefix(logo, "branding/")
	if r.panelURL != "" && strings.HasPrefix(logo, "branding/") && name != "" && !strings.ContainsAny(name, `/\`) {
		brand.LogoURL = strings.TrimRight(r.panelURL, "/") + "/v1/uploads/branding/" + name
	}
	return brand
}
