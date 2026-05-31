package api

import (
	"net/http"

	"ecomteam/internal/domain"
)

type checkoutRequest struct {
	Plan string `json:"plan"`
}

func (s *Server) handleCheckout(w http.ResponseWriter, r *http.Request) {
	var req checkoutRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	plan := domain.PlanID(req.Plan)
	if plan != domain.PlanPro && plan != domain.PlanBusiness {
		writeError(w, http.StatusBadRequest, "plan must be 'pro' or 'business'")
		return
	}

	variant := s.cfg.LSVariantPro
	if plan == domain.PlanBusiness {
		variant = s.cfg.LSVariantBusiness
	}

	url, err := s.billing.Checkout(r.Context(), userID(r), userEmail(r), plan, variant)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not start checkout: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

func (s *Server) handlePortal(w http.ResponseWriter, r *http.Request) {
	sub, err := s.store.GetSubscription(r.Context(), userID(r))
	if err != nil || sub.LSSubscription == "" {
		writeError(w, http.StatusBadRequest, "no active subscription to manage")
		return
	}
	url, err := s.billing.PortalURL(r.Context(), sub.LSSubscription)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not open portal: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}
