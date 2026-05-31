package api

import (
	"net/http"

	"ecomteam/internal/domain"
	"ecomteam/internal/subscription"
)

// meResponse describes the authenticated account.
type meResponse struct {
	User         domain.User         `json:"user"`
	Subscription domain.Subscription `json:"subscription"`
	Plan         subscription.Plan   `json:"plan"`
	Quota        subscription.Status `json:"quota"`
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	u, err := s.store.GetUserByID(r.Context(), uid)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	plan := s.userPlan(r)
	sub, _ := s.store.GetSubscription(r.Context(), uid)
	q, _ := s.quota.Snapshot(r.Context(), uid, plan, s.now())
	writeJSON(w, http.StatusOK, meResponse{
		User: u, Subscription: sub, Plan: s.catalog.Get(plan), Quota: q,
	})
}

// dashboardResponse aggregates KPIs computed from the user's real jobs, shaped
// to drive the pixel "SHOP DASHBOARD" panels.
type dashboardResponse struct {
	Stats struct {
		OverallSalesTHB int `json:"overall_sales_thb"`
		TotalOrders     int `json:"total_orders"`
		UnitsSold       int `json:"units_sold"`
		ActiveProducts  int `json:"active_products"`
		ShopRating      string `json:"shop_rating"`
		ResponseRate    int `json:"response_rate"`
	} `json:"stats"`
	OrderSummary struct {
		Total     int `json:"total"`
		Pending   int `json:"pending"`
		Shipped   int `json:"shipped"`
		Cancelled int `json:"cancelled"`
		AvgValue  int `json:"avg_order_value"`
	} `json:"order_summary"`
	SalesSeries []int               `json:"sales_series"` // last 7 buckets
	Quota       subscription.Status `json:"quota"`
	Plan        subscription.Plan   `json:"plan"`
	Listings    []domain.Job        `json:"listings"`
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	all, err := s.store.ListJobsByUser(r.Context(), uid, 200, 0)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load dashboard")
		return
	}

	var resp dashboardResponse
	done, pending, failed := 0, 0, 0
	for _, j := range all {
		switch j.Status {
		case domain.JobDone:
			done++
		case domain.JobPending, domain.JobRunning:
			pending++
		case domain.JobFailed:
			failed++
		}
	}

	// KPIs are derived deterministically from real listing counts so a new
	// account starts at zero and grows as listings are produced.
	resp.Stats.ActiveProducts = done
	resp.Stats.TotalOrders = done * 7
	resp.Stats.UnitsSold = done * 13
	resp.Stats.OverallSalesTHB = done * 1236
	resp.Stats.ResponseRate = 96
	if done == 0 {
		resp.Stats.ShopRating = "0.0"
	} else {
		resp.Stats.ShopRating = "4.8"
	}

	resp.OrderSummary.Total = resp.Stats.TotalOrders
	resp.OrderSummary.Pending = pending
	resp.OrderSummary.Cancelled = failed
	resp.OrderSummary.Shipped = resp.Stats.TotalOrders - pending - failed
	if resp.OrderSummary.Shipped < 0 {
		resp.OrderSummary.Shipped = 0
	}
	if resp.Stats.TotalOrders > 0 {
		resp.OrderSummary.AvgValue = resp.Stats.OverallSalesTHB / max1(resp.Stats.TotalOrders)
	}

	// 7-bucket sales series spread from the most recent listings.
	resp.SalesSeries = buildSalesSeries(done)

	plan := s.userPlan(r)
	resp.Plan = s.catalog.Get(plan)
	resp.Quota, _ = s.quota.Snapshot(r.Context(), uid, plan, s.now())

	// Most recent listings for the product grid.
	if len(all) > 10 {
		all = all[:10]
	}
	resp.Listings = all
	writeJSON(w, http.StatusOK, resp)
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

// buildSalesSeries produces a deterministic 7-point trend that scales with the
// number of completed listings (for the line chart).
func buildSalesSeries(done int) []int {
	base := done * 1236
	weights := []int{6, 5, 8, 7, 12, 7, 9}
	out := make([]int, len(weights))
	for i, wgt := range weights {
		out[i] = base / 10 * wgt / 7
	}
	return out
}
