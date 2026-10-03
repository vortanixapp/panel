package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/vortanixapp/panel/pkg/settingsreg"
)

type registryItem struct {
	Key     string   `json:"key"`
	Group   string   `json:"group"`
	Section string   `json:"section"`
	Kind    string   `json:"kind"`
	Default string   `json:"default"`
	Min     int64    `json:"min"`
	Max     int64    `json:"max"`
	Unit    string   `json:"unit"`
	Options []string `json:"options,omitempty"`
	Value   string   `json:"value"`
	Custom  bool     `json:"custom"`
}

var patchSettingKeys = map[string]bool{
	"security.user_api_tokens":    true,
	"security.staff_2fa_required": true,
	"referral.enabled":            true,
	"referral.percent":            true,
	"referral.months":             true,
	"referral.min_payment":        true,
}

func patchKeyAllowed(key string) bool {
	if patchSettingKeys[key] {
		return true
	}
	_, ok := settingsreg.Lookup(key)
	return ok
}

func (h *Handler) GetAdminSettingsRegistry(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		return
	}
	settingsreg.Invalidate()
	effective := settingsreg.Effective()
	stored := settingsreg.Stored()
	items := make([]registryItem, 0)
	for _, s := range settingsreg.All() {
		items = append(items, registryItem{
			Key: s.Key, Group: s.Group, Section: s.Section, Kind: string(s.Kind),
			Default: s.Default, Min: s.Min, Max: s.Max, Unit: s.Unit, Options: s.Options,
			Value: effective[s.Key], Custom: strings.TrimSpace(stored[s.Key]) != "",
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) UpdateAdminSettingsRegistry(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		return
	}
	ctx := r.Context()
	var body struct {
		Values map[string]any `json:"values"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	next := map[string]string{}
	resets := []string{}
	fields := map[string]string{}
	for key, raw := range body.Values {
		s, known := settingsreg.Lookup(key)
		if !known {
			fields[key] = "неизвестная настройка"
			continue
		}
		text := registryInputString(raw)
		if text == "" {
			resets = append(resets, key)
			continue
		}
		norm, err := s.Normalize(text)
		if err != nil {
			fields[key] = err.Error()
			continue
		}
		next[key] = norm
	}
	if len(fields) == 0 {
		if err := settingsreg.Validate(registryProjected(next, resets)); err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": err.Error(), "fields": map[string]string{}})
			return
		}
	}
	if len(fields) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "Проверьте значения", "fields": fields})
		return
	}

	before := settingsreg.Effective()
	changed := map[string]any{}
	keys := make([]string, 0, len(next))
	for k := range next {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		h.setTenantSettingString(ctx, k, next[k])
		if before[k] != next[k] {
			changed[k] = map[string]string{"from": before[k], "to": next[k]}
		}
	}
	sort.Strings(resets)
	for _, k := range resets {
		h.resetRegistrySetting(ctx, k)
		def, _ := settingsreg.Lookup(k)
		if def != nil && before[k] != def.Default {
			changed[k] = map[string]string{"from": before[k], "to": def.Default}
		}
	}
	settingsreg.Invalidate()
	if len(changed) > 0 {
		audit(ctx, h.dbOf(ctx), claims.UserID, "settings.registry", "settings", map[string]any{"changed": changed})
	}
	h.GetAdminSettingsRegistry(w, r)
}

func (h *Handler) resetRegistrySetting(ctx context.Context, key string) {
	_, _ = h.dbOf(ctx).Exec(ctx, `DELETE FROM core.tenant_settings WHERE key = $1`, key)
}

func registryProjected(next map[string]string, resets []string) map[string]string {
	out := map[string]string{}
	for k, v := range next {
		out[k] = v
	}
	for _, k := range resets {
		if s, ok := settingsreg.Lookup(k); ok {
			out[k] = s.Default
		}
	}
	return out
}

func registryInputString(raw any) string {
	switch v := raw.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case bool:
		if v {
			return "1"
		}
		return "0"
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case []any:
		parts := make([]string, 0, len(v))
		for _, item := range v {
			if n, ok := item.(float64); ok {
				parts = append(parts, strconv.FormatInt(int64(n), 10))
			}
		}
		return strings.Join(parts, ",")
	}
	return ""
}
