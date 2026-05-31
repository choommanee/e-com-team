package affiliate

import (
	"context"
	"strings"
	"testing"

	"ecomteam/internal/domain"
	"ecomteam/internal/llm"
)

func TestGenerateProfile(t *testing.T) {
	a := New(llm.NewMock())
	p, err := a.GenerateProfile(context.Background(), "บิวตี้", "วัยรุ่น")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if p.Bio == "" || p.Niche == "" || p.Pitch == "" {
		t.Fatalf("incomplete profile: %+v", p)
	}
}

func TestGeneratePromoContentIncludesLink(t *testing.T) {
	a := New(llm.NewMock())
	link := "http://localhost:8080/r/ABC123"
	posts, err := a.GeneratePromoContent(context.Background(), "", link)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(posts) == 0 {
		t.Fatal("expected posts")
	}
	for i, p := range posts {
		if !strings.Contains(p, link) {
			t.Fatalf("post %d missing referral link: %q", i, p)
		}
	}
}

func TestRecommendProducts(t *testing.T) {
	a := New(llm.NewMock())
	recs, err := a.RecommendProducts(context.Background(), "บิวตี้")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(recs) < 1 {
		t.Fatal("expected recommendations")
	}
	if recs[0].Category == "" || recs[0].Reason == "" {
		t.Fatalf("incomplete recommendation: %+v", recs[0])
	}
}

func TestNewCodeFormat(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		c := NewCode()
		if len(c) != 6 {
			t.Fatalf("code length != 6: %q", c)
		}
		seen[c] = true
	}
	if len(seen) < 90 {
		t.Fatalf("codes not unique enough: %d/100", len(seen))
	}
}

func TestCommission(t *testing.T) {
	if got := Commission(domain.PlanPro, 299); got != 59 {
		t.Fatalf("expected 20%% of 299 = 59, got %d", got)
	}
}
