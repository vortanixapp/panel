package regions

import (
	"sort"
	"strings"
)

const (
	CIS    = "cis"
	EU     = "eu"
	Europe = "europe"
	US     = "us"
	World  = "world"
)

var Order = []string{CIS, EU, Europe, US}

var euCountries = []string{
	"AT", "BE", "BG", "HR", "CY", "CZ", "DK", "EE", "FI", "FR", "DE", "GR", "HU", "IE",
	"IT", "LV", "LT", "LU", "MT", "NL", "PL", "PT", "RO", "SK", "SI", "ES", "SE",
}

var europeCountries = []string{
	"GB", "CH", "NO", "IS", "LI", "AL", "AD", "BA", "ME", "MK", "RS", "SM", "VA", "MC", "XK",
}

var cisCountries = []string{
	"RU", "BY", "KZ", "UA", "AM", "AZ", "GE", "KG", "MD", "TJ", "TM", "UZ",
}

var otherCountries = []string{
	"AF", "DZ", "AS", "AO", "AI", "AQ", "AG", "AR", "AW", "AU", "BS", "BH", "BD", "BB", "BZ", "BJ",
	"BM", "BT", "BO", "BQ", "BW", "BV", "BR", "IO", "BN", "BF", "BI", "CV", "KH", "CM", "CA", "KY",
	"CF", "TD", "CL", "CN", "CX", "CC", "CO", "KM", "CG", "CD", "CK", "CR", "CI", "CU", "CW", "DJ",
	"DM", "DO", "EC", "EG", "SV", "GQ", "ER", "SZ", "ET", "FK", "FO", "FJ", "GF", "PF", "TF", "GA",
	"GM", "GH", "GI", "GL", "GD", "GP", "GU", "GT", "GG", "GN", "GW", "GY", "HT", "HM", "HN", "HK",
	"IN", "ID", "IR", "IQ", "IM", "IL", "JM", "JP", "JE", "JO", "KE", "KI", "KP", "KR", "KW", "LA",
	"LB", "LS", "LR", "LY", "MO", "MG", "MW", "MY", "MV", "ML", "MH", "MQ", "MR", "MU", "YT", "MX",
	"FM", "MN", "MS", "MA", "MZ", "MM", "NA", "NR", "NP", "NC", "NZ", "NI", "NE", "NG", "NU", "NF",
	"MP", "OM", "PK", "PW", "PS", "PA", "PG", "PY", "PE", "PH", "PN", "PR", "QA", "RE", "RW", "BL",
	"SH", "KN", "LC", "MF", "PM", "VC", "WS", "ST", "SA", "SN", "SC", "SL", "SG", "SX", "SB", "SO",
	"ZA", "GS", "SS", "LK", "SD", "SR", "SJ", "SY", "TW", "TZ", "TH", "TL", "TG", "TK", "TO", "TT",
	"TN", "TR", "TC", "TV", "UG", "AE", "UM", "UY", "VU", "VE", "VN", "VG", "VI", "WF", "EH", "YE",
	"ZM", "ZW", "AX",
}

var countryRegion = buildCountryRegion()

func buildCountryRegion() map[string]string {
	out := map[string]string{"US": US}
	for _, c := range euCountries {
		out[c] = EU
	}
	for _, c := range europeCountries {
		out[c] = Europe
	}
	for _, c := range cisCountries {
		out[c] = CIS
	}
	for _, c := range otherCountries {
		out[c] = World
	}
	return out
}

func NormalizeCountry(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func ValidCountry(code string) bool {
	_, ok := countryRegion[NormalizeCountry(code)]
	return ok
}

func RegionOf(code string) string {
	if region, ok := countryRegion[NormalizeCountry(code)]; ok {
		return region
	}
	return ""
}

func Countries() []string {
	out := make([]string, 0, len(countryRegion))
	for code := range countryRegion {
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}

func InRegion(code, region string) bool {
	return RegionOf(code) == region
}

func IsEU(code string) bool {
	return RegionOf(code) == EU
}

func ValidRegion(region string) bool {
	for _, r := range Order {
		if r == region {
			return true
		}
	}
	return false
}

func ParseList(raw string) []string {
	seen := map[string]bool{}
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
		part = strings.ToLower(strings.TrimSpace(part))
		if ValidRegion(part) {
			seen[part] = true
		}
	}
	out := make([]string, 0, len(seen))
	for _, r := range Order {
		if seen[r] {
			out = append(out, r)
		}
	}
	return out
}

func JoinList(list []string) string {
	return strings.Join(ParseList(strings.Join(list, ",")), ",")
}

func Enabled(enabled []string, country string) bool {
	region := RegionOf(country)
	if region == "" {
		return false
	}
	if region == World {
		return len(enabled) > 0
	}
	for _, r := range enabled {
		if r == region {
			return true
		}
	}
	return false
}
