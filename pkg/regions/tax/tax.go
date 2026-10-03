package tax

import (
	"math"

	"github.com/vortanixapp/panel/pkg/regions"
)

const (
	RegimeNone          = "none"
	RegimeOutsideScope  = "outside_scope"
	RegimeExempt        = "exempt"
	RegimeVATRU         = "vat_ru"
	RegimeVATEU         = "vat_eu"
	RegimeVATEurope     = "vat_europe"
	RegimeVATCIS        = "vat_cis"
	RegimeSalesTaxUS    = "sales_tax_us"
	RegimeReverseCharge = "reverse_charge"
)

type Rate struct {
	Rate  float64
	Label string
}

type RateLookup func(country, subdivision string) (Rate, bool)

type Seller struct {
	Country string
	RUVAT   float64
}

type Buyer struct {
	Country       string
	Subdivision   string
	Business      bool
	TaxID         string
	TaxIDVerified bool
	Exempt        bool
}

type Input struct {
	Seller  Seller
	Buyer   Buyer
	Enabled []string
	Gross   float64
	Rates   RateLookup
}

type Result struct {
	Regime        string
	Country       string
	Subdivision   string
	Rate          float64
	Label         string
	Net           float64
	Tax           float64
	Gross         float64
	ReverseCharge bool
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func carveOut(gross, rate float64) (net, tax float64) {
	if rate <= 0 {
		return round2(gross), 0
	}
	tax = round2(gross * rate / (100 + rate))
	return round2(gross - tax), tax
}

func Calculate(in Input) Result {
	country := regions.NormalizeCountry(in.Buyer.Country)
	if country == "" {
		country = regions.NormalizeCountry(in.Seller.Country)
	}
	res := Result{
		Regime:      RegimeNone,
		Country:     country,
		Subdivision: in.Buyer.Subdivision,
		Net:         round2(in.Gross),
		Gross:       round2(in.Gross),
	}

	if !regions.Enabled(in.Enabled, country) {
		res.Regime = RegimeOutsideScope
		return res
	}
	if in.Buyer.Exempt {
		res.Regime = RegimeExempt
		return res
	}

	sellerCountry := regions.NormalizeCountry(in.Seller.Country)
	region := regions.RegionOf(country)

	apply := func(regime string, rate Rate) Result {
		res.Regime = regime
		res.Rate = rate.Rate
		res.Label = rate.Label
		res.Net, res.Tax = carveOut(in.Gross, rate.Rate)
		return res
	}
	lookup := func(sub string) (Rate, bool) {
		if in.Rates == nil {
			return Rate{}, false
		}
		return in.Rates(country, sub)
	}

	if country == "RU" && sellerCountry == "RU" {
		if in.Seller.RUVAT > 0 {
			return apply(RegimeVATRU, Rate{Rate: in.Seller.RUVAT})
		}
		return res
	}

	switch region {
	case regions.US:
		if rate, ok := lookup(in.Buyer.Subdivision); ok && rate.Rate > 0 {
			return apply(RegimeSalesTaxUS, rate)
		}
	case regions.EU:
		if in.Buyer.Business && in.Buyer.TaxID != "" && in.Buyer.TaxIDVerified && country != sellerCountry {
			res.Regime = RegimeReverseCharge
			res.ReverseCharge = true
			return res
		}
		if rate, ok := lookup(""); ok && rate.Rate > 0 {
			return apply(RegimeVATEU, rate)
		}
	case regions.Europe:
		if rate, ok := lookup(""); ok && rate.Rate > 0 {
			return apply(RegimeVATEurope, rate)
		}
	case regions.CIS:
		if rate, ok := lookup(""); ok && rate.Rate > 0 {
			return apply(RegimeVATCIS, rate)
		}
	}
	return res
}
