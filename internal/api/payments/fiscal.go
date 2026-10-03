package payments

import "sort"

type FiscalSystem struct {
	Country   string   `json:"country"`
	Name      string   `json:"name"`
	Available bool     `json:"available"`
	Providers []string `json:"providers"`
}

func receiptProviders() []string {
	out := []string{}
	for _, def := range buildCatalog() {
		for _, f := range def.Fields {
			if f.Key == "receipt" {
				out = append(out, def.Key)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

func FiscalSystems() []FiscalSystem {
	return []FiscalSystem{
		{Country: "RU", Name: "54-FZ", Available: true, Providers: receiptProviders()},
		{Country: "KZ", Name: "OFD / ISNA"},
		{Country: "BY", Name: "SKNO"},
		{Country: "UA", Name: "PRRO"},
		{Country: "PL", Name: "KSeF"},
		{Country: "IT", Name: "SDI"},
		{Country: "HU", Name: "NAV Online Invoice"},
		{Country: "RO", Name: "RO e-Factura"},
		{Country: "FR", Name: "PDP / Chorus Pro"},
		{Country: "DE", Name: "XRechnung / ZUGFeRD"},
	}
}
