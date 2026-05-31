package api

import (
	"errors"
	"net/http"
	"time"

	"ecomteam/internal/affiliate"
	"ecomteam/internal/domain"
	"ecomteam/internal/store"
)

// affiliateView is the affiliate dashboard payload (record + referral link).
type affiliateView struct {
	Affiliate domain.Affiliate `json:"affiliate"`
	Link      string           `json:"referral_link"`
}

func (s *Server) referralLink(code string) string {
	return s.cfg.PublicBaseURL + "/r/" + code
}

type affiliateApplyRequest struct {
	NicheHint string `json:"niche_hint"`
	Audience  string `json:"audience"`
}

// handleAffiliateApply runs the AI to draft a profile and enrolls the user.
func (s *Server) handleAffiliateApply(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)

	// Idempotent: if already enrolled, return the existing affiliate.
	if existing, err := s.store.GetAffiliateByUser(r.Context(), uid); err == nil {
		writeJSON(w, http.StatusOK, affiliateView{Affiliate: existing, Link: s.referralLink(existing.Code)})
		return
	}

	var req affiliateApplyRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	profile, err := s.affiliate.GenerateProfile(r.Context(), req.NicheHint, req.Audience)
	if err != nil {
		writeError(w, http.StatusBadGateway, "AI could not draft profile: "+err.Error())
		return
	}

	// Allocate a unique referral code (retry on collision).
	var code string
	for attempt := 0; attempt < 5; attempt++ {
		code = affiliate.NewCode()
		if _, err := s.store.GetAffiliateByCode(r.Context(), code); errors.Is(err, store.ErrNotFound) {
			break
		}
	}

	aff := domain.Affiliate{
		UserID: uid, Code: code, Status: "approved",
		Bio: profile.Bio, Niche: profile.Niche, Pitch: profile.Pitch,
		CreatedAt: s.now(),
	}
	if err := s.store.CreateAffiliate(r.Context(), aff); err != nil {
		writeError(w, http.StatusInternalServerError, "could not enroll affiliate")
		return
	}
	writeJSON(w, http.StatusCreated, affiliateView{Affiliate: aff, Link: s.referralLink(code)})
}

// handleAffiliateGet returns the caller's affiliate dashboard.
func (s *Server) handleAffiliateGet(w http.ResponseWriter, r *http.Request) {
	aff, err := s.store.GetAffiliateByUser(r.Context(), userID(r))
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"enrolled": false})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load affiliate")
		return
	}
	writeJSON(w, http.StatusOK, affiliateView{Affiliate: aff, Link: s.referralLink(aff.Code)})
}

type affiliateContentRequest struct {
	Topic string `json:"topic"`
}

// handleAffiliateContent generates promo posts containing the referral link.
func (s *Server) handleAffiliateContent(w http.ResponseWriter, r *http.Request) {
	aff, err := s.store.GetAffiliateByUser(r.Context(), userID(r))
	if err != nil {
		writeError(w, http.StatusForbidden, "apply to the affiliate program first")
		return
	}
	var req affiliateContentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	posts, err := s.affiliate.GeneratePromoContent(r.Context(), req.Topic, s.referralLink(aff.Code))
	if err != nil {
		writeError(w, http.StatusBadGateway, "AI could not generate content: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"posts": posts})
}

// handleAffiliateRecommend suggests product categories to promote.
func (s *Server) handleAffiliateRecommend(w http.ResponseWriter, r *http.Request) {
	aff, err := s.store.GetAffiliateByUser(r.Context(), userID(r))
	if err != nil {
		writeError(w, http.StatusForbidden, "apply to the affiliate program first")
		return
	}
	recs, err := s.affiliate.RecommendProducts(r.Context(), aff.Niche)
	if err != nil {
		writeError(w, http.StatusBadGateway, "AI could not recommend: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recommendations": recs})
}

// handleReferralLanding tracks a click, drops a referral cookie, and sends the
// visitor to the app so they can sign up attributed to the affiliate.
func (s *Server) handleReferralLanding(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	aff, err := s.store.GetAffiliateByCode(r.Context(), code)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	aff.Clicks++
	_ = s.store.UpdateAffiliate(r.Context(), aff)

	http.SetCookie(w, &http.Cookie{
		Name: "ecom_ref", Value: code, Path: "/",
		MaxAge: int((30 * 24 * time.Hour).Seconds()), HttpOnly: false,
	})
	http.Redirect(w, r, "/dashboard?ref="+code, http.StatusSeeOther)
}

// attributeReferral records that a new user signed up via an affiliate code.
// It prefers an explicit code, falling back to the ecom_ref cookie. Self-
// referral and unknown codes are ignored.
func (s *Server) attributeReferral(r *http.Request, newUserID, explicitCode string) {
	code := explicitCode
	if code == "" {
		if c, err := r.Cookie("ecom_ref"); err == nil {
			code = c.Value
		}
	}
	if code == "" {
		return
	}
	aff, err := s.store.GetAffiliateByCode(r.Context(), code)
	if err != nil || aff.UserID == newUserID {
		return
	}
	_ = s.store.RecordReferral(r.Context(), domain.Referral{
		ReferredUserID: newUserID, AffiliateCode: code, CreatedAt: s.now(),
	})
}

// creditReferral pays an affiliate commission the first time a referred user
// upgrades to a paid plan.
func (s *Server) creditReferral(r *http.Request, referredUserID string, plan domain.PlanID) {
	if plan == domain.PlanFree {
		return
	}
	ref, err := s.store.GetReferralByUser(r.Context(), referredUserID)
	if err != nil || ref.Converted {
		return
	}
	aff, err := s.store.GetAffiliateByCode(r.Context(), ref.AffiliateCode)
	if err != nil {
		return
	}
	price := s.catalog.Get(plan).PriceTHB
	aff.EarningsTHB += affiliate.Commission(plan, price)
	_ = s.store.UpdateAffiliate(r.Context(), aff)
	_ = s.store.MarkReferralConverted(r.Context(), referredUserID)
}
