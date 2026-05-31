package api

import (
	"net/http"
	"strings"

	"ecomteam/internal/affiliate"
	"ecomteam/internal/shopeeaff"
)

// handleProductSearch finds real Shopee products by keyword.
func (s *Server) handleProductSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, http.StatusBadRequest, "query 'q' is required")
		return
	}
	limit := queryInt(r, "limit", 10)
	products, err := s.shopee.SearchProducts(r.Context(), q, limit)
	if err != nil {
		writeError(w, http.StatusBadGateway, "product search failed: "+err.Error())
		return
	}
	if products == nil {
		products = []shopeeaff.Product{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": products})
}

type promoteRequest struct {
	OriginURL   string   `json:"origin_url"`   // a real Shopee product URL (search result or pasted)
	ProductName string   `json:"product_name"` // optional, improves the caption
	PriceTHB    float64  `json:"price_thb"`
	ImageURL    string   `json:"image_url"` // real product image
	SubIDs      []string `json:"sub_ids"`   // optional tracking sub-ids (max 5)
}

type promoteResponse struct {
	ShortLink string                 `json:"short_link"`
	Promo     affiliate.ProductPromo `json:"promo"`
	ImageURL  string                 `json:"image_url"`  // the real product image (echo back)
	GraphicURL string                `json:"graphic_url,omitempty"` // AI promo graphic (added in N3)
}

// handlePromote turns a real Shopee product URL into an affiliate short link
// plus AI-written promo content. No fake product is created.
func (s *Server) handlePromote(w http.ResponseWriter, r *http.Request) {
	var req promoteRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.OriginURL = strings.TrimSpace(req.OriginURL)
	if !strings.Contains(req.OriginURL, "shopee.") && !strings.Contains(req.OriginURL, "shope.ee") {
		writeError(w, http.StatusBadRequest, "origin_url must be a Shopee product link")
		return
	}
	if len(req.SubIDs) > 5 {
		req.SubIDs = req.SubIDs[:5]
	}

	// 1) Real affiliate short link.
	link, err := s.shopee.ShortLink(r.Context(), req.OriginURL, req.SubIDs)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not generate affiliate link: "+err.Error())
		return
	}

	// 2) AI promo content for the real product, ending with the link.
	name := req.ProductName
	if name == "" {
		name = "สินค้าตามลิงก์นี้"
	}
	promo, err := s.affiliate.PromoteProduct(r.Context(), name, req.PriceTHB, link)
	if err != nil {
		writeError(w, http.StatusBadGateway, "could not generate promo: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, promoteResponse{
		ShortLink: link,
		Promo:     promo,
		ImageURL:  req.ImageURL,
	})
}
