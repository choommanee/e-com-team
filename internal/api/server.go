// Package api wires the HTTP server: routing, middleware, JSON handlers, the
// SSE stream, and the server-rendered pixel dashboard.
package api

import (
	"html/template"
	"net/http"
	"time"

	"ecomteam/internal/auth"
	"ecomteam/internal/billing"
	"ecomteam/internal/config"
	"ecomteam/internal/events"
	"ecomteam/internal/jobs"
	"ecomteam/internal/store"
	"ecomteam/internal/subscription"
)

// Server holds all dependencies for the HTTP layer.
type Server struct {
	cfg       config.Config
	store     store.Store
	tokens    *auth.TokenManager
	bus       *events.Bus
	pool      *jobs.Pool
	quota     *subscription.Quota
	catalog   *subscription.Catalog
	billing   billing.Provider
	templates *template.Template
	now       func() time.Time
}

// Deps bundles the constructed dependencies for the server.
type Deps struct {
	Config    config.Config
	Store     store.Store
	Tokens    *auth.TokenManager
	Bus       *events.Bus
	Pool      *jobs.Pool
	Quota     *subscription.Quota
	Catalog   *subscription.Catalog
	Billing   billing.Provider
	Templates *template.Template
}

// New builds a Server.
func New(d Deps) *Server {
	return &Server{
		cfg:       d.Config,
		store:     d.Store,
		tokens:    d.Tokens,
		bus:       d.Bus,
		pool:      d.Pool,
		quota:     d.Quota,
		catalog:   d.Catalog,
		billing:   d.Billing,
		templates: d.Templates,
		now:       time.Now,
	}
}

// Handler builds the HTTP router.
func (s *Server) Handler(staticFS http.Handler) http.Handler {
	mux := http.NewServeMux()

	// --- Auth ---
	mux.HandleFunc("POST /api/v1/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)

	// --- Account ---
	mux.Handle("GET /api/v1/me", s.requireAuth(s.handleMe))

	// --- Listings ---
	mux.Handle("POST /api/v1/listings", s.requireAuth(s.handleCreateListing))
	mux.Handle("GET /api/v1/listings", s.requireAuth(s.handleListListings))
	mux.Handle("GET /api/v1/listings/{id}", s.requireAuth(s.handleGetListing))

	// --- Dashboard + realtime ---
	mux.Handle("GET /api/v1/dashboard", s.requireAuth(s.handleDashboard))
	mux.Handle("GET /api/v1/events", s.requireAuth(s.handleEvents))

	// --- Billing ---
	mux.Handle("POST /api/v1/billing/checkout", s.requireAuth(s.handleCheckout))
	mux.Handle("POST /api/v1/billing/portal", s.requireAuth(s.handlePortal))
	mux.HandleFunc("POST /webhooks/lemonsqueezy", s.handleWebhook)
	mux.HandleFunc("GET /dev/mock-activate", s.handleMockActivate)

	// --- Static + images + pages ---
	mux.Handle("GET /static/", http.StripPrefix("/static/", staticFS))
	mux.Handle("GET /images/", http.StripPrefix("/images/",
		http.FileServer(http.Dir(s.cfg.OutputDir))))
	mux.HandleFunc("GET /dashboard", s.pageDashboard)
	mux.HandleFunc("GET /", s.pageLanding)

	return s.recoverer(s.logger(mux))
}
