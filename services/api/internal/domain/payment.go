package domain

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotConfigured is returned when an adapter is used without its required
	// environment credentials (e.g. Paystack secret key is empty).
	ErrNotConfigured = errors.New("provider not configured")

	// ErrWebhookInvalid is returned when a provider webhook signature cannot be
	// verified or the payload is malformed.
	ErrWebhookInvalid = errors.New("webhook invalid")
)

// WebhookEventChargeSuccess is the only provider event type that may book an
// order; every other event is recorded and ignored.
const WebhookEventChargeSuccess = "charge.success"

// WebhookEvent is the normalized result of parsing a provider webhook payload.
type WebhookEvent struct {
	Type          string
	ProviderRef   string
	Status        string
	AmountPesewas int64
}

// PaymentEvent is a raw provider response or webhook kept for audit.
type PaymentEvent struct {
	ProviderRef string
	Provider    string
	Type        string
	Payload     []byte
	CreatedAt   time.Time
}

// PaymentEventRepository stores raw provider events for audit and debugging.
type PaymentEventRepository interface {
	RecordEvent(ctx context.Context, event PaymentEvent) error
}

// PaymentProvider is the outbound port for payment processing.
type PaymentProvider interface {
	// InitTransaction starts a checkout and returns the authorization URL and
	// the provider's reference for the transaction.
	InitTransaction(
		ctx context.Context,
		amountPesewas int64,
		email, reference, callbackURL string,
	) (authorizationURL string, providerRef string, err error)

	// VerifyWebhook validates the payload signature and returns the parsed event.
	VerifyWebhook(payload []byte, signature string) (*WebhookEvent, error)

	// VerifyTransaction asks the provider for the current status of a transaction.
	VerifyTransaction(ctx context.Context, providerRef string) (status string, err error)
}
