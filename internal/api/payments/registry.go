package payments

import "fmt"

var registry = map[string]Provider{
	"yookassa":      YooKassa{},
	"yoomoney":      YooMoney{},
	"tkassa":        TKassa{},
	"robokassa":     Robokassa{},
	"freekassa":     Freekassa{},
	"cloudpayments": CloudPayments{},
	"unitpay":       Unitpay{},
	"lava":          Lava{},
	"enot":          Enot{},
	"paymaster":     PayMaster{},
	"payeer":        Payeer{},
	"webmoney":      WebMoney{},
	"webpay":        WebPay{},
	"advcash":       AdvCash{},
	"perfectmoney":  PerfectMoney{},
	"stripe":        Stripe{},
	"paypal":        PayPal{},
	"mollie":        Mollie{},
	"paystack":      Paystack{},
	"flutterwave":   Flutterwave{},
	"razorpay":      Razorpay{},
	"payfast":       PayFast{},
	"square":        Square{},
	"authorizenet":  AuthorizeNet{},
	"paddle":        Paddle{},
	"nowpayments":   NowPayments{},
	"cryptocloud":   CryptoCloud{},
	"coinbase":      Coinbase{},
	"cryptocom":     CryptoCom{},
	"bank":          BankTransfer{},
}

func Get(code string) (Provider, error) {
	p, ok := registry[code]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", code)
	}
	return p, nil
}

func IsSupported(code string) bool {
	_, ok := registry[code]
	return ok
}

func Codes() []string {
	out := make([]string, 0, len(registry))
	for _, def := range AdminCatalog() {
		if _, ok := registry[def.Key]; ok {
			out = append(out, def.Key)
		}
	}
	return out
}

func Name(code string) string {
	if def, ok := Definition(code); ok {
		return def.Name
	}
	return code
}

func HasNotifications(code string) bool {
	p, ok := registry[code]
	if !ok {
		return false
	}
	_, notifies := p.(Notifier)
	return notifies
}
