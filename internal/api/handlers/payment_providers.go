package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/internal/api/payments"
)

const fxFeeKey = "payments.fx.fee_percent"

type paymentProviderRow struct {
	Exists     bool
	Enabled    bool
	Unreadable bool
	Config     map[string]any
}

func paymentFeeKey(code string) string {
	return "payments.providers." + code + ".fee_percent"
}

func (h *Handler) paymentBaseURL() string {
	return strings.TrimRight(envOr("PANEL_PUBLIC_URL", h.frontendURL), "/")
}

func (h *Handler) paymentNotifyURL(code string) string {
	return h.paymentBaseURL() + "/v1/webhooks/" + code
}

func (h *Handler) paymentReturnURL(paymentID string) string {
	return h.paymentBaseURL() + "/billing/topup/payment/" + paymentID
}

func (h *Handler) paymentCheckoutURL(paymentID string) string {
	return h.paymentBaseURL() + "/v1/pay/" + paymentID
}

func (h *Handler) decodeProviderConfig(raw []byte) (map[string]any, bool) {
	cfg := map[string]any{}
	if len(raw) == 0 {
		return cfg, true
	}
	plain, err := h.secrets.DecryptJSON(raw)
	if err != nil {
		return cfg, false
	}
	if len(plain) == 0 {
		return cfg, true
	}
	if err := json.Unmarshal(plain, &cfg); err != nil {
		return map[string]any{}, false
	}
	return cfg, true
}

func (h *Handler) loadPaymentProvider(ctx context.Context, code string) (paymentProviderRow, error) {
	var enabled bool
	var raw []byte
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT enabled, config FROM core.payment_providers WHERE provider = $1
	`, code).Scan(&enabled, &raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return paymentProviderRow{Config: map[string]any{}}, nil
	}
	if err != nil {
		return paymentProviderRow{}, err
	}
	cfg, ok := h.decodeProviderConfig(raw)
	return paymentProviderRow{Exists: true, Enabled: enabled, Unreadable: !ok, Config: cfg}, nil
}

func (h *Handler) loadPaymentProviders(ctx context.Context) map[string]paymentProviderRow {
	out := map[string]paymentProviderRow{}
	rows, err := h.dbOf(ctx).Query(ctx, `SELECT provider, enabled, config FROM core.payment_providers`)
	if err != nil {
		log.Printf("платёжные провайдеры не прочитаны: %v", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		var enabled bool
		var raw []byte
		if rows.Scan(&code, &enabled, &raw) != nil {
			continue
		}
		cfg, ok := h.decodeProviderConfig(raw)
		out[code] = paymentProviderRow{Exists: true, Enabled: enabled, Unreadable: !ok, Config: cfg}
	}
	return out
}

func paymentProviderReady(def payments.AdminProviderDef, row paymentProviderRow) bool {
	return row.Enabled && !row.Unreadable && payments.IsSupported(def.Key) && len(def.MissingFields(row.Config)) == 0
}

func providerValue(cfg map[string]any, key string) string {
	return strings.TrimSpace(anyString(cfg[key]))
}

func anyString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "1"
		}
		return "0"
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprint(t)
	}
}

func parsePercent(s string) float64 {
	v, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(s), ",", "."), 64)
	if err != nil {
		return 0
	}
	return payments.ClampPercent(v)
}

func percentFromAny(v any) (float64, error) {
	var f float64
	switch t := v.(type) {
	case float64:
		f = t
	case string:
		s := strings.ReplaceAll(strings.TrimSpace(t), ",", ".")
		if s == "" {
			return 0, nil
		}
		parsed, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, err
		}
		f = parsed
	default:
		return 0, fmt.Errorf("неподдерживаемое значение")
	}
	if math.IsNaN(f) || f < 0 || f >= 100 {
		return 0, fmt.Errorf("вне диапазона")
	}
	return f, nil
}

func defaultCurrencyFrom(settings map[string]string) string {
	cur := strings.ToUpper(strings.TrimSpace(settings["billing.default_currency"]))
	if slices.Contains(walletCurrencies, cur) {
		return cur
	}
	return walletCurrencies[0]
}

func (h *Handler) paymentSettingsView(settings map[string]string) map[string]any {
	return map[string]any{
		"default_currency": defaultCurrencyFrom(settings),
		"currencies":       walletCurrencies,
		"fx_fee_percent":   parsePercent(settings[fxFeeKey]),
	}
}

func (h *Handler) adminPaymentProviderView(def payments.AdminProviderDef, row paymentProviderRow, fee float64) map[string]any {
	config := map[string]string{}
	secrets := map[string]bool{}
	for _, f := range def.Fields {
		value := providerValue(row.Config, f.Key)
		if f.Type == "password" {
			secrets[f.Key] = value != ""
			continue
		}
		config[f.Key] = value
	}
	missing := def.MissingFields(row.Config)
	view := map[string]any{
		"key":           def.Key,
		"name":          def.Name,
		"enabled":       row.Enabled,
		"configured":    len(missing) == 0,
		"missing":       missing,
		"unreadable":    row.Unreadable,
		"fee_percent":   fee,
		"fields":        def.Fields,
		"config":        config,
		"secrets":       secrets,
		"manual":        def.Manual,
		"currency_mode": def.CurrencyMode(),
		"currency":      def.ChargeCurrency(row.Config, ""),
	}
	if payments.HasNotifications(def.Key) {
		view["webhook_url"] = h.paymentNotifyURL(def.Key)
	}
	return view
}

func (h *Handler) AdminPaymentProviders(w http.ResponseWriter, r *http.Request) {
	if _, ok := tenantClaims(r.Context()); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	rows := h.loadPaymentProviders(ctx)
	settings := h.loadTenantSettingStrings(ctx)
	list := make([]map[string]any, 0, len(payments.AdminCatalog()))
	for _, def := range payments.AdminCatalog() {
		list = append(list, h.adminPaymentProviderView(def, rows[def.Key], parsePercent(settings[paymentFeeKey(def.Key)])))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"providers": list,
		"settings":  h.paymentSettingsView(settings),
	})
}

func (h *Handler) AdminUpdatePaymentProvider(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	code := chi.URLParam(r, "code")
	def, found := payments.Definition(code)
	if !found {
		writeError(w, http.StatusNotFound, "Платёжный провайдер не найден")
		return
	}
	var body struct {
		Enabled    *bool          `json:"enabled"`
		FeePercent any            `json:"fee_percent"`
		Config     map[string]any `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	row, err := h.loadPaymentProvider(ctx, code)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}

	enabled := row.Enabled
	if body.Enabled != nil {
		enabled = *body.Enabled
	}

	var fee *float64
	if body.FeePercent != nil {
		parsed, err := percentFromAny(body.FeePercent)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Комиссия должна быть числом от 0 до 100")
			return
		}
		fee = &parsed
	}

	if row.Unreadable && body.Config == nil {
		if enabled {
			writeError(w, http.StatusConflict, "Сохранённые ключи провайдера не расшифровываются — проверьте SECRETS_KEY или введите ключи заново")
			return
		}
		if _, err := h.dbOf(ctx).Exec(ctx, `UPDATE core.payment_providers SET enabled = false WHERE provider = $1`, code); err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
	} else {
		merged := map[string]any{}
		if !row.Unreadable {
			for k, v := range row.Config {
				merged[k] = v
			}
		}
		for _, f := range def.Fields {
			raw, present := body.Config[f.Key]
			if !present {
				continue
			}
			value := strings.TrimSpace(anyString(raw))
			switch f.Type {
			case "password":
				if value != "" {
					merged[f.Key] = value
				}
			case "checkbox":
				if formTruthy(value) {
					merged[f.Key] = "1"
				} else {
					merged[f.Key] = "0"
				}
			default:
				if value == "" {
					delete(merged, f.Key)
				} else {
					merged[f.Key] = value
				}
			}
		}
		if enabled {
			if missing := def.MissingFields(merged); len(missing) > 0 {
				writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
					"error":   "Чтобы включить провайдер, заполните: " + strings.Join(missing, ", "),
					"missing": missing,
				})
				return
			}
		}
		plain, _ := json.Marshal(merged)
		sealed, err := h.secrets.EncryptJSON(plain)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Ключи провайдера не зашифрованы")
			return
		}
		if _, err := h.dbOf(ctx).Exec(ctx, `
			INSERT INTO core.payment_providers (provider, enabled, config)
			VALUES ($1, $2, $3::jsonb)
			ON CONFLICT (provider) DO UPDATE SET enabled = EXCLUDED.enabled, config = EXCLUDED.config
		`, code, enabled, sealed); err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		row = paymentProviderRow{Exists: true, Enabled: enabled, Config: merged}
	}

	if fee != nil {
		h.setTenantSettingString(ctx, paymentFeeKey(code), strconv.FormatFloat(*fee, 'f', -1, 64))
	}
	row.Enabled = enabled

	audit(ctx, h.dbOf(ctx), claims.UserID, "payment_provider.update", "payment_provider:"+code,
		map[string]any{"enabled": enabled})

	writeJSON(w, http.StatusOK, map[string]any{
		"provider": h.adminPaymentProviderView(def, row, parsePercent(h.tenantSettingString(ctx, paymentFeeKey(code)))),
	})
}

func (h *Handler) AdminUpdatePaymentSettings(w http.ResponseWriter, r *http.Request) {
	claims, ok := tenantClaims(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	ctx := r.Context()
	var body struct {
		DefaultCurrency *string `json:"default_currency"`
		FXFeePercent    any     `json:"fx_fee_percent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.DefaultCurrency != nil {
		cur := strings.ToUpper(strings.TrimSpace(*body.DefaultCurrency))
		if !slices.Contains(walletCurrencies, cur) {
			writeError(w, http.StatusBadRequest, "Неизвестная валюта")
			return
		}
		h.setTenantSettingString(ctx, "billing.default_currency", cur)
	}
	if body.FXFeePercent != nil {
		fee, err := percentFromAny(body.FXFeePercent)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Комиссия за конвертацию должна быть числом от 0 до 100")
			return
		}
		h.setTenantSettingString(ctx, fxFeeKey, strconv.FormatFloat(fee, 'f', -1, 64))
	}
	audit(ctx, h.dbOf(ctx), claims.UserID, "payment_settings.update", "payment_settings", nil)
	writeJSON(w, http.StatusOK, map[string]any{"settings": h.paymentSettingsView(h.loadTenantSettingStrings(ctx))})
}

func (h *Handler) paymentReadinessCounts(ctx context.Context) (enabled, withNotifications int) {
	rows := h.loadPaymentProviders(ctx)
	for _, def := range payments.AdminCatalog() {
		if !paymentProviderReady(def, rows[def.Key]) {
			continue
		}
		enabled++
		if def.Manual || payments.HasNotifications(def.Key) {
			withNotifications++
		}
	}
	return enabled, withNotifications
}
