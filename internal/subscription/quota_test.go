package subscription

import (
	"context"
	"testing"
	"time"

	"ecomteam/internal/domain"
	"ecomteam/internal/store"
)

func TestReserveAllowsUpToLimitThenRejects(t *testing.T) {
	s := store.NewMemory()
	c := NewCatalog("", "")
	q := NewQuota(s, c)
	ctx := context.Background()
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	// Free plan = 5/month.
	for i := 1; i <= 5; i++ {
		allowed, st, err := q.Reserve(ctx, "u1", domain.PlanFree, now)
		if err != nil || !allowed {
			t.Fatalf("reservation %d should be allowed (err=%v)", i, err)
		}
		if st.Used != i {
			t.Fatalf("expected used=%d, got %d", i, st.Used)
		}
	}
	// 6th rejected, and usage must not climb past the limit.
	allowed, st, err := q.Reserve(ctx, "u1", domain.PlanFree, now)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if allowed {
		t.Fatal("6th reservation should be rejected")
	}
	if got, _ := s.GetUsage(ctx, "u1", PeriodKey(now)); got != 5 {
		t.Fatalf("usage should stay at 5 after rejection, got %d", got)
	}
	if st.Remaining != 0 {
		t.Fatalf("expected remaining 0, got %d", st.Remaining)
	}
}

func TestRefundReleasesQuota(t *testing.T) {
	s := store.NewMemory()
	q := NewQuota(s, NewCatalog("", ""))
	ctx := context.Background()
	now := time.Now()

	_, _, _ = q.Reserve(ctx, "u1", domain.PlanFree, now)
	q.Refund(ctx, "u1", now)
	if got, _ := s.GetUsage(ctx, "u1", PeriodKey(now)); got != 0 {
		t.Fatalf("expected usage 0 after refund, got %d", got)
	}
}

func TestPeriodRollover(t *testing.T) {
	s := store.NewMemory()
	q := NewQuota(s, NewCatalog("", ""))
	ctx := context.Background()
	may := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	june := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 5; i++ {
		_, _, _ = q.Reserve(ctx, "u1", domain.PlanFree, may)
	}
	// New month → fresh quota.
	allowed, _, _ := q.Reserve(ctx, "u1", domain.PlanFree, june)
	if !allowed {
		t.Fatal("new billing period should reset quota")
	}
}

func TestProQuotaLargerThanFree(t *testing.T) {
	c := NewCatalog("v_pro", "v_biz")
	if c.Get(domain.PlanPro).MonthlyQuota <= c.Get(domain.PlanFree).MonthlyQuota {
		t.Fatal("pro quota should exceed free quota")
	}
	if p, ok := c.ByVariant("v_pro"); !ok || p.ID != domain.PlanPro {
		t.Fatalf("ByVariant should map v_pro to pro plan, got %+v ok=%v", p, ok)
	}
}
