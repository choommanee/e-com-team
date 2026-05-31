package shopeeaff

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMockSearchAndShortLink(t *testing.T) {
	m := NewMock()
	ps, err := m.SearchProducts(context.Background(), "ครีมกันแดด", 2)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(ps) != 2 {
		t.Fatalf("expected 2 products, got %d", len(ps))
	}
	if ps[0].Name == "" || ps[0].OfferLink == "" || ps[0].CommissionRate <= 0 {
		t.Fatalf("incomplete product: %+v", ps[0])
	}
	link, err := m.ShortLink(context.Background(), ps[0].OfferLink, []string{"fb"})
	if err != nil || !strings.HasPrefix(link, "https://shope.ee/") {
		t.Fatalf("short link: %v / %q", err, link)
	}
}

func TestShopeeSignDeterministic(t *testing.T) {
	s := NewShopee("app123", "secretXYZ", "th")
	got := s.sign(`{"query":"x"}`, 1700000000)
	want := sha256.Sum256([]byte("app123" + "1700000000" + `{"query":"x"}` + "secretXYZ"))
	if got != hex.EncodeToString(want[:]) {
		t.Fatalf("signature mismatch:\n got %s\nwant %s", got, hex.EncodeToString(want[:]))
	}
}

func TestShopeeSearchParsesGraphQL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the auth header is well-formed.
		if !strings.HasPrefix(r.Header.Get("Authorization"), "SHA256 Credential=app123,") {
			t.Errorf("bad auth header: %s", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"data":{"productOfferV2":{"nodes":[
			{"itemId":228000001,"shopId":1199001,"productName":"ครีมกันแดด X","priceMin":199,"priceMax":299,
			 "commissionRate":0.12,"imageUrl":"https://img/x","offerLink":"https://shopee.co.th/p/1","productLink":"https://shopee.co.th/p/1",
			 "sales":4820,"ratingStar":4.9,"shopName":"Official"}]}}}`))
	}))
	defer srv.Close()

	s := NewShopee("app123", "secretXYZ", "th")
	s.baseURL = srv.URL
	s.now = func() time.Time { return time.Unix(1700000000, 0) }

	ps, err := s.SearchProducts(context.Background(), "ครีมกันแดด", 5)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(ps) != 1 || ps[0].ItemID != "228000001" || ps[0].CommissionRate != 0.12 || ps[0].Sales != 4820 {
		t.Fatalf("unexpected parse: %+v", ps)
	}
}

func TestShopeeShortLinkMutation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"generateShortLink":{"shortLink":"https://shope.ee/ABC123"}}}`))
	}))
	defer srv.Close()

	s := NewShopee("app123", "secretXYZ", "th")
	s.baseURL = srv.URL
	link, err := s.ShortLink(context.Background(), "https://shopee.co.th/p/1", []string{"ig", "fb"})
	if err != nil {
		t.Fatalf("shortlink: %v", err)
	}
	if link != "https://shope.ee/ABC123" {
		t.Fatalf("unexpected link: %s", link)
	}
}

func TestShopeeSurfacesGraphQLError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[{"message":"invalid signature"}]}`))
	}))
	defer srv.Close()
	s := NewShopee("a", "b", "th")
	s.baseURL = srv.URL
	if _, err := s.SearchProducts(context.Background(), "x", 1); err == nil || !strings.Contains(err.Error(), "invalid signature") {
		t.Fatalf("expected graphql error, got %v", err)
	}
}
