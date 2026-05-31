package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ecomteam/internal/domain"
)

// LemonSqueezy implements Provider against the LemonSqueezy API.
type LemonSqueezy struct {
	apiKey        string
	storeID       string
	webhookSecret string
	baseURL       string
	http          *http.Client
}

// NewLemonSqueezy builds a LemonSqueezy provider.
func NewLemonSqueezy(apiKey, storeID, webhookSecret string) *LemonSqueezy {
	return &LemonSqueezy{
		apiKey:        apiKey,
		storeID:       storeID,
		webhookSecret: webhookSecret,
		baseURL:       "https://api.lemonsqueezy.com/v1",
		http:          &http.Client{Timeout: 30 * time.Second},
	}
}

// Checkout creates a hosted checkout for a variant and returns its URL.
func (l *LemonSqueezy) Checkout(ctx context.Context, userID, email string, _ domain.PlanID, variantID string) (string, error) {
	payload := map[string]any{
		"data": map[string]any{
			"type": "checkouts",
			"attributes": map[string]any{
				"checkout_data": map[string]any{
					"email":       email,
					"custom":      map[string]string{"user_id": userID},
				},
			},
			"relationships": map[string]any{
				"store": map[string]any{
					"data": map[string]string{"type": "stores", "id": l.storeID},
				},
				"variant": map[string]any{
					"data": map[string]string{"type": "variants", "id": variantID},
				},
			},
		},
	}
	var out struct {
		Data struct {
			Attributes struct {
				URL string `json:"url"`
			} `json:"attributes"`
		} `json:"data"`
		Errors []struct {
			Detail string `json:"detail"`
		} `json:"errors"`
	}
	if err := l.post(ctx, "/checkouts", payload, &out); err != nil {
		return "", err
	}
	if len(out.Errors) > 0 {
		return "", fmt.Errorf("lemonsqueezy checkout: %s", out.Errors[0].Detail)
	}
	if out.Data.Attributes.URL == "" {
		return "", fmt.Errorf("lemonsqueezy checkout: empty url")
	}
	return out.Data.Attributes.URL, nil
}

// PortalURL returns the customer portal for a subscription.
func (l *LemonSqueezy) PortalURL(ctx context.Context, subscriptionID string) (string, error) {
	var out struct {
		Data struct {
			Attributes struct {
				URLs struct {
					CustomerPortal string `json:"customer_portal"`
				} `json:"urls"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := l.get(ctx, "/subscriptions/"+subscriptionID, &out); err != nil {
		return "", err
	}
	if out.Data.Attributes.URLs.CustomerPortal == "" {
		return "", fmt.Errorf("lemonsqueezy portal: empty url")
	}
	return out.Data.Attributes.URLs.CustomerPortal, nil
}

// VerifyWebhook checks the HMAC-SHA256 hex signature against the body.
func (l *LemonSqueezy) VerifyWebhook(body []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(l.webhookSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ParseEvent extracts a normalized event. Plan is left empty; the handler maps
// VariantID → plan via the subscription catalog.
func (l *LemonSqueezy) ParseEvent(body []byte) (WebhookEvent, error) {
	var raw struct {
		Meta struct {
			EventName  string `json:"event_name"`
			CustomData struct {
				UserID string `json:"user_id"`
			} `json:"custom_data"`
		} `json:"meta"`
		Data struct {
			ID         string `json:"id"`
			Attributes struct {
				Status    string `json:"status"`
				VariantID int    `json:"variant_id"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return WebhookEvent{}, err
	}
	return WebhookEvent{
		EventName:      raw.Meta.EventName,
		UserID:         raw.Meta.CustomData.UserID,
		Status:         raw.Data.Attributes.Status,
		SubscriptionID: raw.Data.ID,
		VariantID:      fmt.Sprintf("%d", raw.Data.Attributes.VariantID),
	}, nil
}

func (l *LemonSqueezy) post(ctx context.Context, path string, body, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return l.do(ctx, http.MethodPost, path, raw, out)
}

func (l *LemonSqueezy) get(ctx context.Context, path string, out any) error {
	return l.do(ctx, http.MethodGet, path, nil, out)
}

func (l *LemonSqueezy) do(ctx context.Context, method, path string, body []byte, out any) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, l.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Authorization", "Bearer "+l.apiKey)

	resp, err := l.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		// Still try to decode an error envelope into out.
		_ = json.Unmarshal(data, out)
		if out == nil {
			return fmt.Errorf("lemonsqueezy %s: status %d: %s", path, resp.StatusCode, string(data))
		}
		return json.Unmarshal(data, out)
	}
	return json.Unmarshal(data, out)
}
