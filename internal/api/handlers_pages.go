package api

import (
	"net/http"

	"ecomteam/internal/subscription"
)

// pageData is the view model passed to HTML templates.
type pageData struct {
	Title   string
	Plans   []subscription.Plan
	Agents  []string
	BaseURL string
}

func (s *Server) pageLanding(w http.ResponseWriter, r *http.Request) {
	// "/" only matches the root; anything else under it is a 404.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.render(w, "landing", pageData{
		Title:   "E-COMMERCE HUB — AI Listing Team",
		Plans:   s.catalog.All(),
		BaseURL: s.cfg.PublicBaseURL,
	})
}

func (s *Server) pageDashboard(w http.ResponseWriter, r *http.Request) {
	s.render(w, "dashboard", pageData{
		Title:   "E-COMMERCE HUB v2.0",
		Plans:   s.catalog.All(),
		Agents:  []string{"BENEFIT", "PROMO", "DESIGN", "PROMPT", "STUDIO", "QUALITY CHECK"},
		BaseURL: s.cfg.PublicBaseURL,
	})
}

func (s *Server) render(w http.ResponseWriter, name string, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
	}
}
