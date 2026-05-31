// Package shopeeaff is a client for the Shopee Affiliate Open API (GraphQL):
// discovering real products and generating affiliate short links. It has a
// real client and a Mock so the whole flow runs offline.
package shopeeaff

import "context"

// Product is a real Shopee product offer (from productOfferV2).
type Product struct {
	ItemID         string  `json:"item_id"`
	ShopID         string  `json:"shop_id"`
	Name           string  `json:"name"`
	PriceMin       float64 `json:"price_min"`
	PriceMax       float64 `json:"price_max"`
	CommissionRate float64 `json:"commission_rate"` // 0..1
	ImageURL       string  `json:"image_url"`
	OfferLink      string  `json:"offer_link"` // canonical product URL
	ProductLink    string  `json:"product_link"`
	Sales          int     `json:"sales"`
	RatingStar     float64 `json:"rating_star"`
	ShopName       string  `json:"shop_name"`
}

// Client talks to the Shopee Affiliate Open API.
type Client interface {
	// SearchProducts discovers real products by keyword (productOfferV2).
	SearchProducts(ctx context.Context, keyword string, limit int) ([]Product, error)
	// ShortLink converts a Shopee product URL into the affiliate's tracking
	// short link (generateShortLink), tagging it with optional sub IDs.
	ShortLink(ctx context.Context, originURL string, subIDs []string) (string, error)
}
