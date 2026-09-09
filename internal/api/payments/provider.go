package payments

import "context"

type CheckoutInput struct {
	PaymentID   string
	Amount      float64
	Currency    string
	Description string
	ReturnURL   string
	FailURL     string
	MethodID    string
	Email       string
}

type Provider interface {
	Code() string
	CreateCheckout(ctx context.Context, cfg map[string]any, in CheckoutInput) (redirectURL, providerPaymentID string, err error)
}
