package payments

import "context"

type BankTransfer struct{}

func (BankTransfer) Code() string { return "bank" }

func (BankTransfer) CreateCheckout(_ context.Context, cfg map[string]any, in CheckoutInput) (Checkout, error) {
	if strCfg(cfg, "account_number") == "" || strCfg(cfg, "account_holder") == "" {
		return Checkout{}, notConfigured("bank")
	}
	return Checkout{Manual: true, ProviderPaymentID: invoiceRef(in)}, nil
}

func BankInstructions(cfg map[string]any) map[string]string {
	out := map[string]string{}
	for _, key := range []string{"bank_name", "account_holder", "account_number", "swift_bic", "instructions"} {
		if v := strCfg(cfg, key); v != "" {
			out[key] = v
		}
	}
	return out
}
