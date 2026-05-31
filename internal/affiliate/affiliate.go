// Package affiliate provides the affiliate/referral program: AI-assisted
// signup, promo-content generation, product recommendations, and commission.
package affiliate

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"strings"

	"ecomteam/internal/domain"
	"ecomteam/internal/llm"
)

// CommissionRate is the share of a referred upgrade paid to the affiliate.
const CommissionRate = 0.20

// AI wraps the LLM client with affiliate-specific prompts.
type AI struct {
	client llm.Client
}

// New builds the affiliate AI helper.
func New(client llm.Client) *AI { return &AI{client: client} }

// Profile is the AI-generated affiliate application.
type Profile struct {
	Bio   string `json:"bio"`
	Niche string `json:"niche"`
	Pitch string `json:"pitch"`
}

// GenerateProfile drafts an affiliate application from optional hints.
func (a *AI) GenerateProfile(ctx context.Context, nicheHint, audience string) (Profile, error) {
	system := "agent:aff_profile\n" +
		"You help a new affiliate apply to an e-commerce AI tool's partner program. " +
		"Write a short professional Thai affiliate profile. " +
		`Respond as JSON: {"bio":"...","niche":"...","pitch":"..."}`
	user := fmt.Sprintf("Niche hint: %s\nAudience: %s", fallback(nicheHint, "ทั่วไป"), fallback(audience, "ผู้ขายออนไลน์"))

	out, err := a.client.Chat(ctx, system, user)
	if err != nil {
		return Profile{}, err
	}
	var p Profile
	if err := json.Unmarshal([]byte(out), &p); err != nil {
		return Profile{}, fmt.Errorf("affiliate profile: bad JSON: %w", err)
	}
	if p.Bio == "" {
		return Profile{}, fmt.Errorf("affiliate profile: empty bio")
	}
	return p, nil
}

// GeneratePromoContent writes ready-to-post promo copy that includes the
// affiliate's referral link.
func (a *AI) GeneratePromoContent(ctx context.Context, topic, referralLink string) ([]string, error) {
	system := "agent:aff_content\n" +
		"You are a social-media copywriter for an affiliate. Write 3 short Thai promotional posts " +
		"(each 1-3 sentences, with emojis) promoting an AI e-commerce listing tool. End each post with " +
		"the referral link provided. " +
		`Respond as JSON: {"posts":["...","...","..."]}`
	user := fmt.Sprintf("Topic: %s\nReferral link to include: %s", fallback(topic, "เครื่องมือ AI ทำรูปสินค้า"), referralLink)

	out, err := a.client.Chat(ctx, system, user)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Posts []string `json:"posts"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return nil, fmt.Errorf("affiliate content: bad JSON: %w", err)
	}
	if len(parsed.Posts) == 0 {
		return nil, fmt.Errorf("affiliate content: no posts")
	}
	// Guarantee the link appears even if the model omitted it.
	for i, post := range parsed.Posts {
		if referralLink != "" && !strings.Contains(post, referralLink) {
			parsed.Posts[i] = post + " " + referralLink
		}
	}
	return parsed.Posts, nil
}

// Recommendation is a suggested niche/product to promote.
type Recommendation struct {
	Category string `json:"category"`
	Reason   string `json:"reason"`
}

// RecommendProducts suggests product categories the affiliate should promote.
func (a *AI) RecommendProducts(ctx context.Context, niche string) ([]Recommendation, error) {
	system := "agent:aff_reco\n" +
		"You advise an affiliate which product categories to promote for the best commission. " +
		"Suggest 4 categories in Thai with a one-line reason each. " +
		`Respond as JSON: {"recommendations":[{"category":"...","reason":"..."}]}`
	user := "Affiliate niche: " + fallback(niche, "ทั่วไป")

	out, err := a.client.Chat(ctx, system, user)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Recommendations []Recommendation `json:"recommendations"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return nil, fmt.Errorf("affiliate reco: bad JSON: %w", err)
	}
	if len(parsed.Recommendations) == 0 {
		return nil, fmt.Errorf("affiliate reco: empty")
	}
	return parsed.Recommendations, nil
}

// NewCode returns a short, URL-safe, human-friendly referral code.
func NewCode() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	out := make([]byte, 6)
	for i := range b {
		out[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(out)
}

// Commission returns the affiliate payout for a referred plan upgrade (THB).
func Commission(plan domain.PlanID, priceTHB int) int {
	return int(float64(priceTHB) * CommissionRate)
}

func fallback(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
