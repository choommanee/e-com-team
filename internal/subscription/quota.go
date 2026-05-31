package subscription

import (
	"context"
	"time"

	"ecomteam/internal/domain"
	"ecomteam/internal/store"
)

// PeriodKey returns the billing-period bucket key for a time ("YYYY-MM").
// A user's monthly quota resets when this key changes.
func PeriodKey(t time.Time) string {
	return t.Format("2006-01")
}

// Quota enforces per-plan monthly limits using the store's atomic usage counter.
type Quota struct {
	store   store.Store
	catalog *Catalog
}

// NewQuota builds a Quota enforcer.
func NewQuota(s store.Store, c *Catalog) *Quota {
	return &Quota{store: s, catalog: c}
}

// Status reports current usage for a user under a plan in the given period.
type Status struct {
	Plan      domain.PlanID `json:"plan"`
	Used      int           `json:"used"`
	Limit     int           `json:"limit"`
	Remaining int           `json:"remaining"`
}

// Snapshot returns current usage without modifying anything.
func (q *Quota) Snapshot(ctx context.Context, userID string, plan domain.PlanID, now time.Time) (Status, error) {
	p := q.catalog.Get(plan)
	used, err := q.store.GetUsage(ctx, userID, PeriodKey(now))
	if err != nil {
		return Status{}, err
	}
	rem := p.MonthlyQuota - used
	if rem < 0 {
		rem = 0
	}
	return Status{Plan: p.ID, Used: used, Limit: p.MonthlyQuota, Remaining: rem}, nil
}

// Reserve atomically claims one unit of quota. It returns allowed=false (and
// refunds the reservation) when the plan limit would be exceeded.
func (q *Quota) Reserve(ctx context.Context, userID string, plan domain.PlanID, now time.Time) (allowed bool, st Status, err error) {
	p := q.catalog.Get(plan)
	period := PeriodKey(now)
	newCount, err := q.store.IncrementUsage(ctx, userID, period)
	if err != nil {
		return false, Status{}, err
	}
	if newCount > p.MonthlyQuota {
		// Over limit: refund and reject.
		_ = q.store.DecrementUsage(ctx, userID, period)
		rem := 0
		return false, Status{Plan: p.ID, Used: p.MonthlyQuota, Limit: p.MonthlyQuota, Remaining: rem}, nil
	}
	rem := p.MonthlyQuota - newCount
	return true, Status{Plan: p.ID, Used: newCount, Limit: p.MonthlyQuota, Remaining: rem}, nil
}

// Refund releases a previously reserved unit (e.g. when a job fails).
func (q *Quota) Refund(ctx context.Context, userID string, now time.Time) {
	_ = q.store.DecrementUsage(ctx, userID, PeriodKey(now))
}
