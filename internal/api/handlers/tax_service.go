package handlers

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/pkg/regions"
	"github.com/vortanixapp/panel/pkg/regions/tax"
	"github.com/vortanixapp/panel/pkg/settingsreg"
)

var regionSellerCountry = map[string]string{
	regions.CIS:    "RU",
	regions.EU:     "DE",
	regions.Europe: "GB",
	regions.US:     "US",
}

func defaultSellerCountry() string {
	enabled := settingsreg.AccountingRegionList()
	if len(enabled) == 0 || slices.Contains(enabled, regions.CIS) {
		return "RU"
	}
	if country, ok := regionSellerCountry[enabled[0]]; ok {
		return country
	}
	return "RU"
}

var taxableSources = map[string]bool{
	"server_rent":      true,
	"server_renew":     true,
	"server_tariff":    true,
	"server_resources": true,
	"hosting_rent":     true,
	"hosting_renew":    true,
}

func isTaxableSource(source string) bool {
	return taxableSources[source]
}

func normalizeSubdivision(country, region string) string {
	region = strings.ToUpper(strings.TrimSpace(region))
	if country == regions.US {
		region = strings.TrimPrefix(region, "US-")
		if len(region) == 2 {
			return region
		}
		return ""
	}
	return ""
}

func (h *Handler) sellerCountry(settings map[string]string) string {
	country := regions.NormalizeCountry(settings["company.country"])
	if regions.ValidCountry(country) {
		return country
	}
	return defaultSellerCountry()
}

func (h *Handler) lookupTaxRate(ctx context.Context) tax.RateLookup {
	return func(country, subdivision string) (tax.Rate, bool) {
		var rate float64
		var label string
		err := h.dbOf(ctx).QueryRow(ctx, `
			SELECT rate::float8, label FROM core.tax_rates
			WHERE country = $1 AND subdivision = $2 AND enabled
		`, country, subdivision).Scan(&rate, &label)
		if err != nil {
			return tax.Rate{}, false
		}
		return tax.Rate{Rate: rate, Label: label}, true
	}
}

type taxBuyerRow struct {
	country, region, payerType, taxID string
	verified, exempt                  bool
}

func (h *Handler) loadTaxBuyer(ctx context.Context, userID string) taxBuyerRow {
	var b taxBuyerRow
	_ = h.dbOf(ctx).QueryRow(ctx, `
		SELECT COALESCE(p.country, ''), COALESCE(p.region, ''),
		       COALESCE(b.payer_type, 'person'), COALESCE(b.tax_id, ''),
		       b.tax_id_verified_at IS NOT NULL, COALESCE(b.tax_exempt, false)
		FROM core.users u
		LEFT JOIN core.user_profiles p ON p.user_id = u.id
		LEFT JOIN core.user_billing_profiles b ON b.user_id = u.id
		WHERE u.id = $1
	`, userID).Scan(&b.country, &b.region, &b.payerType, &b.taxID, &b.verified, &b.exempt)
	return b
}

func (h *Handler) computeServiceTax(ctx context.Context, userID string, gross float64) tax.Result {
	settings := h.loadTenantSettingStrings(ctx)
	profile := accountingProfileFrom(settings)
	ruVAT := 0.0
	if n, err := strconv.ParseFloat(profile.VAT, 64); err == nil && n > 0 {
		ruVAT = n
	}
	buyer := h.loadTaxBuyer(ctx, userID)
	country := regions.NormalizeCountry(buyer.country)
	return tax.Calculate(tax.Input{
		Seller: tax.Seller{Country: h.sellerCountry(settings), RUVAT: ruVAT},
		Buyer: tax.Buyer{
			Country:       country,
			Subdivision:   normalizeSubdivision(country, buyer.region),
			Business:      buyer.payerType != "person",
			TaxID:         buyer.taxID,
			TaxIDVerified: buyer.verified,
			Exempt:        buyer.exempt,
		},
		Enabled: settingsreg.AccountingRegionList(),
		Gross:   gross,
		Rates:   h.lookupTaxRate(ctx),
	})
}

func (h *Handler) recordServiceTax(ctx context.Context, tx pgx.Tx, txID, userID, source string, gross float64) {
	if txID == "" || gross <= 0 || !isTaxableSource(source) {
		return
	}
	res := h.computeServiceTax(ctx, userID, gross)
	buyer := h.loadTaxBuyer(ctx, userID)
	sp, err := tx.Begin(ctx)
	if err != nil {
		return
	}
	_, err = sp.Exec(ctx, `
		INSERT INTO core.transaction_taxes
			(transaction_id, country, subdivision, regime, rate, net, tax, gross, reverse_charge, buyer_tax_id)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (transaction_id) DO NOTHING
	`, txID, res.Country, res.Subdivision, res.Regime, res.Rate, res.Net, res.Tax, res.Gross, res.ReverseCharge, buyer.taxID)
	if err != nil {
		_ = sp.Rollback(ctx)
		return
	}
	_ = sp.Commit(ctx)
}
