package payments

import "fmt"

var registry = map[string]Provider{
	"yookassa":    YooKassa{},
	"freekassa":   Freekassa{},
	"robokassa":   Robokassa{},
	"stripe":      Stripe{},
	"paypal":      PayPal{},
	"nowpayments": NowPayments{},
}

func Get(code string) (Provider, error) {
	p, ok := registry[code]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", code)
	}
	return p, nil
}

func Supported() []string {
	return []string{"yookassa", "freekassa", "robokassa", "stripe", "paypal", "nowpayments"}
}

func IsSupported(code string) bool {
	_, err := Get(code)
	return err == nil
}
