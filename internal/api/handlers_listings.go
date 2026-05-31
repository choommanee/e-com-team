package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"ecomteam/internal/domain"
	"ecomteam/internal/store"
)

type createListingRequest struct {
	ProductName string `json:"product_name"`
	Lang        string `json:"lang"`
}

func (s *Server) handleCreateListing(w http.ResponseWriter, r *http.Request) {
	var req createListingRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.ProductName = strings.TrimSpace(req.ProductName)
	if req.ProductName == "" {
		writeError(w, http.StatusBadRequest, "product_name is required")
		return
	}
	if req.Lang != "en" {
		req.Lang = "th"
	}

	uid := userID(r)
	plan := s.userPlan(r)

	allowed, status, err := s.quota.Reserve(r.Context(), uid, plan, s.now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "quota check failed")
		return
	}
	if !allowed {
		writeJSON(w, http.StatusPaymentRequired, map[string]any{
			"error":   "monthly quota exceeded for your plan",
			"quota":   status,
			"upgrade": "/dashboard",
		})
		return
	}

	job := domain.Job{
		ID:          newID(),
		UserID:      uid,
		ProductName: req.ProductName,
		Lang:        req.Lang,
		Status:      domain.JobPending,
		CreatedAt:   s.now(),
		UpdatedAt:   s.now(),
	}
	if err := s.store.CreateJob(r.Context(), job); err != nil {
		// Roll back the reservation if we couldn't persist the job.
		s.quota.Refund(r.Context(), uid, s.now())
		writeError(w, http.StatusInternalServerError, "could not create job")
		return
	}
	s.pool.Enqueue(job.ID)

	writeJSON(w, http.StatusAccepted, job)
}

func (s *Server) handleGetListing(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, err := s.store.GetJob(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "listing not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load listing")
		return
	}
	if job.UserID != userID(r) {
		writeError(w, http.StatusNotFound, "listing not found")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleListListings(w http.ResponseWriter, r *http.Request) {
	limit := queryInt(r, "limit", 20)
	offset := queryInt(r, "offset", 0)
	jobs, err := s.store.ListJobsByUser(r.Context(), userID(r), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list listings")
		return
	}
	if jobs == nil {
		jobs = []domain.Job{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"listings": jobs})
}

// userPlan resolves the caller's current plan, defaulting to free.
func (s *Server) userPlan(r *http.Request) domain.PlanID {
	sub, err := s.store.GetSubscription(r.Context(), userID(r))
	if err != nil || sub.Plan == "" {
		return domain.PlanFree
	}
	return sub.Plan
}

func queryInt(r *http.Request, key string, def int) int {
	if v := r.URL.Query().Get(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return def
}
