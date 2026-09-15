package payments

type Receipt struct {
	Email    string `json:"email"`
	Item     string `json:"item"`
	Taxation string `json:"taxation"`
	VAT      string `json:"vat"`
	Mode     string `json:"mode"`
}

var (
	atolTaxCodes = map[string][2]string{
		"0":  {"vat0", "vat0"},
		"5":  {"vat5", "vat105"},
		"7":  {"vat7", "vat107"},
		"10": {"vat10", "vat110"},
		"20": {"vat20", "vat120"},
		"22": {"vat22", "vat122"},
	}
	yookassaVATCodes = map[string][2]int{
		"none": {1, 1},
		"0":    {2, 2},
		"10":   {3, 5},
		"20":   {4, 6},
		"5":    {7, 9},
		"7":    {8, 10},
		"22":   {11, 12},
	}
	cloudPaymentsVATCodes = map[string][2]int{
		"0":  {0, 0},
		"5":  {5, 105},
		"7":  {7, 107},
		"10": {10, 110},
		"20": {20, 120},
		"22": {22, 122},
	}
	yookassaTaxSystems     = map[string]int{"osn": 1, "usn_income": 2, "usn_income_outcome": 3, "patent": 6}
	cloudPaymentsTaxations = map[string]int{"osn": 0, "usn_income": 1, "usn_income_outcome": 2, "patent": 5}
)

func ReceiptRequested(cfg map[string]any) bool {
	return boolCfg(cfg, "receipt")
}

func (r *Receipt) mode() string {
	if r != nil && (r.Mode == "full_prepayment" || r.Mode == "advance") {
		return r.Mode
	}
	return "full_payment"
}

func (r *Receipt) calculated() bool {
	return r.mode() != "full_payment"
}

func (r *Receipt) subject() string {
	if r.mode() == "advance" {
		return "payment"
	}
	return "service"
}

func (r *Receipt) item(fallback string) string {
	if r != nil && r.Item != "" {
		return truncate(r.Item, 128)
	}
	return truncate(fallback, 128)
}

func (r *Receipt) sno() string {
	if r == nil {
		return ""
	}
	if _, ok := yookassaTaxSystems[r.Taxation]; ok {
		return r.Taxation
	}
	return ""
}

func (r *Receipt) atolTax() string {
	if r == nil {
		return "none"
	}
	codes, ok := atolTaxCodes[r.VAT]
	if !ok {
		return "none"
	}
	if r.calculated() {
		return codes[1]
	}
	return codes[0]
}

func (r *Receipt) yookassaVAT(fallback int) int {
	if r != nil {
		if codes, ok := yookassaVATCodes[r.VAT]; ok {
			if r.calculated() {
				return codes[1]
			}
			return codes[0]
		}
	}
	if fallback <= 0 {
		return 1
	}
	return fallback
}

func (r *Receipt) cloudPaymentsVAT() any {
	if r == nil {
		return nil
	}
	codes, ok := cloudPaymentsVATCodes[r.VAT]
	if !ok {
		return nil
	}
	if r.calculated() {
		return codes[1]
	}
	return codes[0]
}

func (r *Receipt) cloudPaymentsMethod() int {
	switch r.mode() {
	case "full_prepayment":
		return 1
	case "advance":
		return 3
	}
	return 4
}

func (r *Receipt) cloudPaymentsObject() int {
	if r.subject() == "payment" {
		return 10
	}
	return 4
}
