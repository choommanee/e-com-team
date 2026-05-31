package billing

import (
	"context"
	"encoding/json"
	"fmt"

	"ecomteam/internal/domain"
)

// Mock is a no-network Provider used when BILLING_MODE=mock. Checkout returns a
// local URL that immediately "activates" the plan via the dev webhook endpoint.
type Mock struct {
	baseURL string
}

// NewMock builds a mock provider. baseURL is the app's public base URL.
func NewMock(baseURL string) *Mock { return &Mock{baseURL: baseURL} }

func (m *Mock) Checkout(_ context.Context, userID, _ string, plan domain.PlanID, _ string) (string, error) {
	// Points at the app's own mock-activation endpoint so the upgrade can be
	// demonstrated end-to-end without a real payment provider.
	return fmt.Sprintf("%s/dev/mock-activate?user=%s&plan=%s", m.baseURL, userID, plan), nil
}

func (m *Mock) PortalURL(_ context.Context, _ string) (string, error) {
	return m.baseURL + "/dashboard", nil
}

// VerifyWebhook accepts everything in mock mode.
func (m *Mock) VerifyWebhook(_ []byte, _ string) bool { return true }

func (m *Mock) ParseEvent(body []byte) (WebhookEvent, error) {
	var e WebhookEvent
	if err := json.Unmarshal(body, &e); err != nil {
		return WebhookEvent{}, err
	}
	return e, nil
}
