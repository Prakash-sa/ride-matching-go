package payments

import (
	"context"
	"os"

	stripe "github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/paymentintent"
)

// StripeClient is a thin wrapper around stripe-go for PaymentIntent hold/capture/cancel flows.
type StripeClient struct{}

// NewStripeClient initializes the stripe client with the STRIPE_API_KEY env var.
func NewStripeClient() *StripeClient {
	stripe.Key = os.Getenv("STRIPE_API_KEY")
	return &StripeClient{}
}

// Hold creates a PaymentIntent with capture_method=manual to hold funds.
// It returns the PaymentIntent ID on success.
func (s *StripeClient) Hold(ctx context.Context, amount int64, currency, customerID string) (string, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(currency),
	}
	if customerID != "" {
		params.Customer = stripe.String(customerID)
	}
	params.CaptureMethod = stripe.String(string(stripe.PaymentIntentCaptureMethodManual))
	pi, err := paymentintent.New(params)
	if err != nil {
		return "", err
	}
	return pi.ID, nil
}

// Capture finalizes a previously-held PaymentIntent.
func (s *StripeClient) Capture(ctx context.Context, paymentIntentID string) error {
	_, err := paymentintent.Capture(paymentIntentID, nil)
	return err
}

// Cancel releases the hold on a PaymentIntent.
func (s *StripeClient) Cancel(ctx context.Context, paymentIntentID string) error {
	_, err := paymentintent.Cancel(paymentIntentID, nil)
	return err
}
