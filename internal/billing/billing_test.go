package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"ecomteam/internal/domain"
)

func TestLemonSqueezyWebhookSignature(t *testing.T) {
	ls := NewLemonSqueezy("k", "1", "whsec")
	body := []byte(`{"meta":{"event_name":"subscription_created"}}`)

	mac := hmac.New(sha256.New, []byte("whsec"))
	mac.Write(body)
	good := hex.EncodeToString(mac.Sum(nil))

	if !ls.VerifyWebhook(body, good) {
		t.Error("valid signature should verify")
	}
	if ls.VerifyWebhook(body, "deadbeef") {
		t.Error("invalid signature must be rejected")
	}
	if ls.VerifyWebhook(body, good+"00") {
		t.Error("length-mismatched signature must be rejected")
	}
}

func TestLemonSqueezyParseEvent(t *testing.T) {
	ls := NewLemonSqueezy("k", "1", "whsec")
	body := []byte(`{
		"meta":{"event_name":"subscription_created","custom_data":{"user_id":"u-42"}},
		"data":{"id":"sub-9","attributes":{"status":"active","variant_id":12345}}
	}`)
	e, err := ls.ParseEvent(body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if e.EventName != "subscription_created" || e.UserID != "u-42" ||
		e.SubscriptionID != "sub-9" || e.Status != "active" || e.VariantID != "12345" {
		t.Fatalf("unexpected event: %+v", e)
	}
}

func TestMockCheckoutURL(t *testing.T) {
	m := NewMock("http://localhost:8080")
	url, err := m.Checkout(context.Background(), "u-1", "a@b.com", domain.PlanPro, "")
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if !strings.Contains(url, "user=u-1") || !strings.Contains(url, "plan=pro") {
		t.Fatalf("mock checkout url missing params: %s", url)
	}
	if !m.VerifyWebhook([]byte("x"), "anything") {
		t.Error("mock should accept any webhook")
	}
}
