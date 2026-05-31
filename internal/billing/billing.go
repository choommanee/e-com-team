// Package billing abstracts subscription checkout and webhook handling behind a
// Provider interface, with a mock (no external calls) and a LemonSqueezy impl.
package billing

import (
	"context"

	"ecomteam/internal/domain"
)

// WebhookEvent is the normalized result of parsing a provider webhook.
type WebhookEvent struct {
	EventName      string        // e.g. subscription_created
	UserID         string        // resolved from custom data
	Plan           domain.PlanID // resolved from variant
	Status         string        // active, cancelled, expired
	SubscriptionID string
	VariantID      string
}

// Provider issues checkout/portal links and verifies+parses webhooks.
type Provider interface {
	// Checkout returns a hosted checkout URL for the given user + plan.
	Checkout(ctx context.Context, userID, email string, plan domain.PlanID, variantID string) (string, error)
	// PortalURL returns a customer self-service portal URL for a subscription.
	PortalURL(ctx context.Context, subscriptionID string) (string, error)
	// VerifyWebhook reports whether the signature matches the raw body.
	VerifyWebhook(body []byte, signature string) bool
	// ParseEvent extracts a normalized event from a verified webhook body.
	ParseEvent(body []byte) (WebhookEvent, error)
}
