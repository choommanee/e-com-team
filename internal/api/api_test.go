package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ecomteam/internal/affiliate"
	"ecomteam/internal/agents"
	"ecomteam/internal/auth"
	"ecomteam/internal/billing"
	"ecomteam/internal/config"
	"ecomteam/internal/domain"
	"ecomteam/internal/events"
	"ecomteam/internal/jobs"
	"ecomteam/internal/llm"
	"ecomteam/internal/promo"
	"ecomteam/internal/shopeeaff"
	"ecomteam/internal/store"
	"ecomteam/internal/subscription"
)

func newTestServer(t *testing.T) (*httptest.Server, context.CancelFunc) {
	t.Helper()
	cfg := config.Load()
	cfg.OutputDir = t.TempDir()
	cfg.BillingMode = "mock"

	st := store.NewMemory()
	bus := events.New()
	cat := subscription.NewCatalog("", "")
	q := subscription.NewQuota(st, cat)
	orch := agents.NewOrchestrator(llm.NewMock())
	pool := jobs.New(orch, st, bus, q, cfg.OutputDir, cfg.PublicBaseURL, 2)

	ctx, cancel := context.WithCancel(context.Background())
	go pool.Start(ctx)

	srv := New(Deps{
		Config: cfg, Store: st, Tokens: auth.NewTokenManager("test-secret", time.Hour),
		Bus: bus, Pool: pool, Quota: q, Catalog: cat, Billing: billing.NewMock(cfg.PublicBaseURL),
		Affiliate: affiliate.New(llm.NewMock()),
		Shopee:    shopeeaff.NewMock(),
		Promo:     mustPromoBuilder(t),
		Templates: nil,
	})
	ts := httptest.NewServer(srv.Handler(http.NotFoundHandler()))
	return ts, cancel
}

func mustPromoBuilder(t *testing.T) *promo.Builder {
	t.Helper()
	b, err := promo.NewBuilder()
	if err != nil {
		t.Fatalf("promo builder: %v", err)
	}
	return b
}

func postJSON(t *testing.T, url, token string, body any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request %s: %v", url, err)
	}
	return resp
}

func TestFullListingFlow(t *testing.T) {
	ts, cancel := newTestServer(t)
	defer cancel()
	defer ts.Close()

	// Register
	resp := postJSON(t, ts.URL+"/api/v1/auth/register", "", map[string]string{
		"email": "seller@example.com", "password": "secret123",
	})
	if resp.StatusCode != 200 {
		t.Fatalf("register status = %d", resp.StatusCode)
	}
	var auth authResponse
	_ = json.NewDecoder(resp.Body).Decode(&auth)
	resp.Body.Close()
	if auth.Token == "" {
		t.Fatal("expected token from register")
	}

	// Create listing
	resp = postJSON(t, ts.URL+"/api/v1/listings", auth.Token, map[string]string{
		"product_name": "ครีมกันแดด", "lang": "th",
	})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("create listing status = %d", resp.StatusCode)
	}
	var job domain.Job
	_ = json.NewDecoder(resp.Body).Decode(&job)
	resp.Body.Close()
	if job.ID == "" || job.Status != domain.JobPending {
		t.Fatalf("unexpected job: %+v", job)
	}

	// Poll until done.
	var final domain.Job
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest("GET", ts.URL+"/api/v1/listings/"+job.ID, nil)
		req.Header.Set("Authorization", "Bearer "+auth.Token)
		r, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("poll: %v", err)
		}
		_ = json.NewDecoder(r.Body).Decode(&final)
		r.Body.Close()
		if final.Status == domain.JobDone || final.Status == domain.JobFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if final.Status != domain.JobDone {
		t.Fatalf("expected job done, got %s (%s)", final.Status, final.Error)
	}
	if final.Result == nil || final.Result.ImageURL == "" || len(final.Result.SellingPoints) == 0 {
		t.Fatalf("expected complete listing, got %+v", final.Result)
	}
}

func TestAuthRequired(t *testing.T) {
	ts, cancel := newTestServer(t)
	defer cancel()
	defer ts.Close()

	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/me", nil)
	resp, _ := http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestQuotaEnforced(t *testing.T) {
	ts, cancel := newTestServer(t)
	defer cancel()
	defer ts.Close()

	resp := postJSON(t, ts.URL+"/api/v1/auth/register", "", map[string]string{
		"email": "limited@example.com", "password": "secret123",
	})
	var a authResponse
	_ = json.NewDecoder(resp.Body).Decode(&a)
	resp.Body.Close()

	// Free plan allows 5; the 6th must be rejected with 402.
	for i := 0; i < 5; i++ {
		r := postJSON(t, ts.URL+"/api/v1/listings", a.Token, map[string]string{"product_name": "x"})
		if r.StatusCode != http.StatusAccepted {
			t.Fatalf("listing %d should be accepted, got %d", i+1, r.StatusCode)
		}
		r.Body.Close()
	}
	r := postJSON(t, ts.URL+"/api/v1/listings", a.Token, map[string]string{"product_name": "x"})
	if r.StatusCode != http.StatusPaymentRequired {
		t.Fatalf("6th listing should be 402, got %d", r.StatusCode)
	}
	r.Body.Close()
}
