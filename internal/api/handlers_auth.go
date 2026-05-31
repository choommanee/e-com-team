package api

import (
	"errors"
	"net/http"
	"strings"

	"ecomteam/internal/auth"
	"ecomteam/internal/domain"
	"ecomteam/internal/store"
)

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  domain.User `json:"user"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if !strings.Contains(req.Email, "@") || len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "valid email and password (min 6 chars) required")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not hash password")
		return
	}
	u := domain.User{ID: newID(), Email: req.Email, PasswordHash: hash, CreatedAt: s.now()}
	if err := s.store.CreateUser(r.Context(), u); err != nil {
		if errors.Is(err, store.ErrDuplicate) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not create user")
		return
	}

	// New users start on the free plan.
	_ = s.store.UpsertSubscription(r.Context(), domain.Subscription{
		UserID: u.ID, Plan: domain.PlanFree, Status: "active",
		PeriodStart: s.now(), PeriodEnd: s.now().AddDate(0, 1, 0),
	})

	s.issueToken(w, u)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req authRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	u, err := s.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil || !auth.CheckPassword(u.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}
	s.issueToken(w, u)
}

func (s *Server) issueToken(w http.ResponseWriter, u domain.User) {
	token, err := s.tokens.Issue(u.ID, u.Email, s.now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue token")
		return
	}
	writeJSON(w, http.StatusOK, authResponse{Token: token, User: u})
}
