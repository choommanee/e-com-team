package api

import (
	"io"
	"net/http"

	"ecomteam/internal/domain"
)

// handleWebhook verifies and applies LemonSqueezy subscription lifecycle events.
func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read body")
		return
	}
	sig := r.Header.Get("X-Signature")
	if !s.billing.VerifyWebhook(body, sig) {
		writeError(w, http.StatusUnauthorized, "invalid signature")
		return
	}
	evt, err := s.billing.ParseEvent(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not parse event")
		return
	}
	if evt.UserID == "" {
		// Nothing to attribute the event to; acknowledge to stop retries.
		w.WriteHeader(http.StatusOK)
		return
	}

	// Resolve the plan from the variant id when the provider didn't set it.
	plan := evt.Plan
	if plan == "" {
		if p, ok := s.catalog.ByVariant(evt.VariantID); ok {
			plan = p.ID
		}
	}

	switch evt.EventName {
	case "subscription_created", "subscription_updated", "subscription_resumed", "subscription_unpaused":
		if plan == "" {
			plan = domain.PlanFree
		}
		s.applySubscription(r, evt.UserID, plan, "active", evt.SubscriptionID)
	case "subscription_cancelled", "subscription_expired", "subscription_paused":
		// Downgrade to free on cancellation/expiry.
		s.applySubscription(r, evt.UserID, domain.PlanFree, "cancelled", evt.SubscriptionID)
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) applySubscription(r *http.Request, uid string, plan domain.PlanID, status, subID string) {
	_ = s.store.UpsertSubscription(r.Context(), domain.Subscription{
		UserID:         uid,
		Plan:           plan,
		Status:         status,
		LSSubscription: subID,
		PeriodStart:    s.now(),
		PeriodEnd:      s.now().AddDate(0, 1, 0),
	})
}

// handleMockActivate lets the mock billing flow upgrade a plan without a real
// payment provider (used only when BILLING_MODE=mock).
func (s *Server) handleMockActivate(w http.ResponseWriter, r *http.Request) {
	if s.cfg.BillingMode != "mock" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	uid := r.URL.Query().Get("user")
	plan := domain.PlanID(r.URL.Query().Get("plan"))
	if uid == "" || (plan != domain.PlanPro && plan != domain.PlanBusiness && plan != domain.PlanFree) {
		writeError(w, http.StatusBadRequest, "user and valid plan required")
		return
	}
	_ = s.store.UpsertSubscription(r.Context(), domain.Subscription{
		UserID: uid, Plan: plan, Status: "active",
		LSSubscription: "mock-" + uid, PeriodStart: s.now(), PeriodEnd: s.now().AddDate(0, 1, 0),
	})
	http.Redirect(w, r, "/dashboard?upgraded="+string(plan), http.StatusSeeOther)
}
