// Package subscription defines the plan catalog and quota enforcement.
package subscription

import "ecomteam/internal/domain"

// Plan describes a subscription tier.
type Plan struct {
	ID            domain.PlanID `json:"id"`
	Name          string        `json:"name"`
	PriceTHB      int           `json:"price_thb"`
	MonthlyQuota  int           `json:"monthly_quota"`
	LSVariantID   string        `json:"-"` // LemonSqueezy variant id (empty for free)
}

// Catalog holds the available plans, keyed by id.
type Catalog struct {
	plans map[domain.PlanID]Plan
}

// NewCatalog builds the default plan catalog. The Pro/Business LemonSqueezy
// variant IDs come from configuration.
func NewCatalog(lsVariantPro, lsVariantBusiness string) *Catalog {
	return &Catalog{plans: map[domain.PlanID]Plan{
		domain.PlanFree:     {ID: domain.PlanFree, Name: "Free", PriceTHB: 0, MonthlyQuota: 5},
		domain.PlanPro:      {ID: domain.PlanPro, Name: "Pro", PriceTHB: 299, MonthlyQuota: 100, LSVariantID: lsVariantPro},
		domain.PlanBusiness: {ID: domain.PlanBusiness, Name: "Business", PriceTHB: 999, MonthlyQuota: 500, LSVariantID: lsVariantBusiness},
	}}
}

// Get returns the plan for an id, falling back to Free for unknown ids.
func (c *Catalog) Get(id domain.PlanID) Plan {
	if p, ok := c.plans[id]; ok {
		return p
	}
	return c.plans[domain.PlanFree]
}

// All returns the plans in display order (free, pro, business).
func (c *Catalog) All() []Plan {
	return []Plan{
		c.plans[domain.PlanFree],
		c.plans[domain.PlanPro],
		c.plans[domain.PlanBusiness],
	}
}

// ByVariant returns the plan matching a LemonSqueezy variant id, if any.
func (c *Catalog) ByVariant(variantID string) (Plan, bool) {
	if variantID == "" {
		return Plan{}, false
	}
	for _, p := range c.plans {
		if p.LSVariantID == variantID {
			return p, true
		}
	}
	return Plan{}, false
}
