package api

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

type ctxKey string

const (
	ctxUserID ctxKey = "user_id"
	ctxEmail  ctxKey = "email"
)

// requireAuth validates a JWT from the Authorization header or, for SSE which
// cannot set headers, the `token` query parameter.
func (s *Server) requireAuth(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			writeError(w, http.StatusUnauthorized, "missing token")
			return
		}
		uid, email, err := s.tokens.Parse(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, uid)
		ctx = context.WithValue(ctx, ctxEmail, email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

func userID(r *http.Request) string {
	if v, ok := r.Context().Value(ctxUserID).(string); ok {
		return v
	}
	return ""
}

func userEmail(r *http.Request) string {
	if v, ok := r.Context().Value(ctxEmail).(string); ok {
		return v
	}
	return ""
}

// recoverer turns panics into 500s instead of crashing the server.
func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic: %v (%s %s)", rec, r.Method, r.URL.Path)
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// logger logs each request with status and duration.
func (s *Server) logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Millisecond))
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Flush exposes the underlying flusher for SSE.
func (s *statusWriter) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
