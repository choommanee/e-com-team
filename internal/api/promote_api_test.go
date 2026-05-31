package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestProductSearch(t *testing.T) {
	ts, cancel := newTestServer(t)
	defer cancel()
	defer ts.Close()
	tok := registerUser(t, ts.URL, "finder@aff.com")

	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/products/search?q=ครีมกันแดด&limit=3", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		t.Fatalf("status = %d", r.StatusCode)
	}
	var out struct {
		Products []map[string]any `json:"products"`
	}
	_ = json.NewDecoder(r.Body).Decode(&out)
	if len(out.Products) == 0 {
		t.Fatal("expected real product results")
	}
	if out.Products[0]["offer_link"] == "" {
		t.Fatal("product should have an offer_link")
	}
}

func TestPromoteRealProduct(t *testing.T) {
	ts, cancel := newTestServer(t)
	defer cancel()
	defer ts.Close()
	tok := registerUser(t, ts.URL, "promoter2@aff.com")

	resp := postJSON(t, ts.URL+"/api/v1/promote", tok, map[string]any{
		"origin_url":   "https://shopee.co.th/product/1199001/228000001",
		"product_name": "ครีมกันแดด SPF50",
		"price_thb":    199,
		"sub_ids":      []string{"facebook"},
	})
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("promote status = %d", resp.StatusCode)
	}
	var out promoteResponse
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if !strings.HasPrefix(out.ShortLink, "https://shope.ee/") {
		t.Fatalf("expected affiliate short link, got %q", out.ShortLink)
	}
	if out.Promo.Caption == "" || len(out.Promo.Hashtags) == 0 {
		t.Fatalf("expected promo caption + hashtags, got %+v", out.Promo)
	}
	// Caption must contain the affiliate link (so the post is monetized).
	if !strings.Contains(out.Promo.Caption, out.ShortLink) {
		t.Fatalf("caption must include the affiliate link")
	}
}

func TestPromoteRejectsNonShopeeURL(t *testing.T) {
	ts, cancel := newTestServer(t)
	defer cancel()
	defer ts.Close()
	tok := registerUser(t, ts.URL, "x@aff.com")

	resp := postJSON(t, ts.URL+"/api/v1/promote", tok, map[string]any{"origin_url": "https://example.com/foo"})
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-Shopee URL, got %d", resp.StatusCode)
	}
}
