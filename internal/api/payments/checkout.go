package payments

import (
	"context"
	"fmt"
	"strings"

	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/checkout/session"
)

func createStripeCheckout(secretKey string, in CheckoutInput) (string, string, error) {
	stripe.Key = secretKey
	currency := strings.ToLower(in.Currency)
	if currency == "" {
		currency = "rub"
	}
	amountCents := int64(in.Amount * 100)
	params := &stripe.CheckoutSessionParams{
		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL: stripe.String(in.ReturnURL),
		CancelURL:  stripe.String(in.FailURL),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Quantity: stripe.Int64(1),
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String(currency),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(in.Description),
					},
					UnitAmount: stripe.Int64(amountCents),
				},
			},
		},
	}
	params.AddMetadata("payment_id", in.PaymentID)
	params.AddMetadata("vortanix_payment_id", in.PaymentID)
	sess, err := session.New(params)
	if err != nil {
		return "", "", err
	}
	if sess.URL == "" {
		return "", "", fmt.Errorf("stripe: empty checkout url")
	}
	return sess.URL, sess.ID, nil
}

func CreateCheckout(ctx context.Context, providerCode string, cfg map[string]any, in CheckoutInput) (redirectURL, providerPaymentID string, err error) {
	p, err := Get(providerCode)
	if err != nil {
		return "", "", err
	}
	return p.CreateCheckout(ctx, cfg, in)
}
