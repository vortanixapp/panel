package jobs

import (
	"context"
	"encoding/json"
	"time"
)

const (
	SettingPanelFreeze      = "panel.freeze"
	SettingPanelFreezeSince = "panel.freeze_since"
)

func (r *Runner) setPanelFreeze(ctx context.Context, on bool) {
	value, _ := json.Marshal(on)
	_, _ = r.db.Exec(ctx, `
		INSERT INTO core.tenant_settings (key, value) VALUES ($1, $2::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
	`, SettingPanelFreeze, value)
	since, _ := json.Marshal(time.Now().UTC().Format(time.RFC3339))
	if !on {
		since, _ = json.Marshal("")
	}
	_, _ = r.db.Exec(ctx, `
		INSERT INTO core.tenant_settings (key, value) VALUES ($1, $2::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()
	`, SettingPanelFreezeSince, since)
}

func (r *Runner) panelFrozen(ctx context.Context) bool {
	var raw []byte
	if r.db.QueryRow(ctx, `SELECT value FROM core.tenant_settings WHERE key = $1`, SettingPanelFreeze).Scan(&raw) != nil {
		return false
	}
	var on bool
	_ = json.Unmarshal(raw, &on)
	return on
}
