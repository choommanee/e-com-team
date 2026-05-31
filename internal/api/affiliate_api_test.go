package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"ecomteam/internal/domain"
)

func registerUser(t *testing.T, base, email string) string {
	t.Helper()
	resp := postJSON(t, base+"/api/v1/auth/register", "", map[string]string{
		"email": email, "password": "secret123",
	})
	defer resp.Body.Close()
	var a authResponse
	_ = json.NewDecoder(resp.Body).Decode(&a)
	return a.Token
}

func TestAffiliateApplyAndContent(t *testing.T) {
	ts, cancel := newTestServer(t)
	defer cancel()
	defer ts.Close()

	tok := registerUser(t, ts.URL, "creator@aff.com")

	// Apply → AI drafts a profile + assigns a code.
	resp := postJSON(t, ts.URL+"/api/v1/affiliate/apply", tok, map[string]string{
		"niche_hint": "บิวตี้", "audience": "วัยรุ่น",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("apply status = %d", resp.StatusCode)
	}
	var av affiliateView
	_ = json.NewDecoder(resp.Body).Decode(&av)
	resp.Body.Close()
	if av.Affiliate.Code == "" || av.Affiliate.Bio == "" || av.Link == "" {
		t.Fatalf("incomplete affiliate: %+v", av)
	}

	// Content → posts contain the referral link.
	resp = postJSON(t, ts.URL+"/api/v1/affiliate/content", tok, map[string]string{"topic": "ครีม"})
	if resp.StatusCode != 200 {
		t.Fatalf("content status = %d", resp.StatusCode)
	}
	var content struct {
		Posts []string `json:"posts"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&content)
	resp.Body.Close()
	if len(content.Posts) == 0 {
		t.Fatal("expected promo posts")
	}

	// Recommendations.
	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/affiliate/recommendations", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr, _ := http.DefaultClient.Do(req)
	if rr.StatusCode != 200 {
		t.Fatalf("recommendations status = %d", rr.StatusCode)
	}
	rr.Body.Close()
}

func TestAffiliateReferralAndCommission(t *testing.T) {
	ts, cancel := newTestServer(t)
	defer cancel()
	defer ts.Close()

	// Affiliate enrolls.
	affTok := registerUser(t, ts.URL, "promoter@aff.com")
	resp := postJSON(t, ts.URL+"/api/v1/affiliate/apply", affTok, map[string]string{})
	var av affiliateView
	_ = json.NewDecoder(resp.Body).Decode(&av)
	resp.Body.Close()
	code := av.Affiliate.Code

	// A new user registers with the referral code.
	resp = postJSON(t, ts.URL+"/api/v1/auth/register", "", map[string]string{
		"email": "referred@buyer.com", "password": "secret123", "ref": code,
	})
	var referred authResponse
	_ = json.NewDecoder(resp.Body).Decode(&referred)
	resp.Body.Close()

	// Referred user upgrades to Pro via the mock-activate endpoint.
	upURL := ts.URL + "/dev/mock-activate?user=" + referred.User.ID + "&plan=pro"
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	r, err := client.Get(upURL)
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	r.Body.Close()

	// Affiliate should now show 1 signup and commission earnings (20% of 299 = 59).
	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/affiliate", nil)
	req.Header.Set("Authorization", "Bearer "+affTok)
	gr, _ := http.DefaultClient.Do(req)
	var got affiliateView
	_ = json.NewDecoder(gr.Body).Decode(&got)
	gr.Body.Close()

	if got.Affiliate.Signups != 1 {
		t.Fatalf("expected 1 signup, got %d", got.Affiliate.Signups)
	}
	if got.Affiliate.EarningsTHB != 59 {
		t.Fatalf("expected 59 THB commission, got %d", got.Affiliate.EarningsTHB)
	}
}

func TestReferralLandingSetsCookieAndTracksClick(t *testing.T) {
	ts, cancel := newTestServer(t)
	defer cancel()
	defer ts.Close()

	affTok := registerUser(t, ts.URL, "linker@aff.com")
	resp := postJSON(t, ts.URL+"/api/v1/affiliate/apply", affTok, map[string]string{})
	var av affiliateView
	_ = json.NewDecoder(resp.Body).Decode(&av)
	resp.Body.Close()

	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	r, err := client.Get(ts.URL + "/r/" + av.Affiliate.Code)
	if err != nil {
		t.Fatalf("landing: %v", err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", r.StatusCode)
	}
	var hasCookie bool
	for _, c := range r.Cookies() {
		if c.Name == "ecom_ref" && c.Value == av.Affiliate.Code {
			hasCookie = true
		}
	}
	if !hasCookie {
		t.Fatal("expected ecom_ref cookie to be set")
	}

	// Click should be tracked.
	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/affiliate", nil)
	req.Header.Set("Authorization", "Bearer "+affTok)
	gr, _ := http.DefaultClient.Do(req)
	var got affiliateView
	_ = json.NewDecoder(gr.Body).Decode(&got)
	gr.Body.Close()
	if got.Affiliate.Clicks != 1 {
		t.Fatalf("expected 1 click, got %d", got.Affiliate.Clicks)
	}
}

var _ = domain.PlanPro
