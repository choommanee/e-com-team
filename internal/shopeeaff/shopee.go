package shopeeaff

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Shopee is the real Shopee Affiliate Open API client (GraphQL).
//
// Auth (per Shopee Affiliate Open API docs): a SHA256 signature over the
// concatenation appId + timestamp + payload + secret, sent in the header
//
//	Authorization: SHA256 Credential=<appId>, Timestamp=<ts>, Signature=<sig>
type Shopee struct {
	appID   string
	secret  string
	baseURL string
	now     func() time.Time
	http    *http.Client
}

// NewShopee builds a real client. region selects the open-api host (e.g. "th").
func NewShopee(appID, secret, region string) *Shopee {
	host := "https://open-api.affiliate.shopee." + regionTLD(region)
	return &Shopee{
		appID:   appID,
		secret:  secret,
		baseURL: host + "/graphql",
		now:     time.Now,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

func regionTLD(region string) string {
	switch region {
	case "vn":
		return "vn"
	case "id":
		return "co.id"
	case "my":
		return "com.my"
	case "ph":
		return "ph"
	case "sg":
		return "sg"
	case "br":
		return "com.br"
	default:
		return "co.th"
	}
}

// sign computes the SHA256 signature for a payload at the given timestamp.
func (s *Shopee) sign(payload string, ts int64) string {
	base := s.appID + strconv.FormatInt(ts, 10) + payload + s.secret
	sum := sha256.Sum256([]byte(base))
	return hex.EncodeToString(sum[:])
}

// do executes a GraphQL request and unmarshals data into out.
func (s *Shopee) do(ctx context.Context, query string, out any) error {
	payload, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return err
	}
	ts := s.now().Unix()
	sig := s.sign(string(payload), ts)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("SHA256 Credential=%s, Timestamp=%d, Signature=%s", s.appID, ts, sig))

	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("shopee affiliate: status %d: %s", resp.StatusCode, string(data))
	}
	return json.Unmarshal(data, out)
}

type gqlError struct {
	Message string `json:"message"`
}

// SearchProducts queries productOfferV2 by keyword.
func (s *Shopee) SearchProducts(ctx context.Context, keyword string, limit int) ([]Product, error) {
	if limit <= 0 {
		limit = 10
	}
	query := fmt.Sprintf(`{ productOfferV2(keyword: %q, limit: %d, sortType: 2) { nodes { itemId shopId productName priceMin priceMax commissionRate imageUrl offerLink productLink sales ratingStar shopName } } }`,
		keyword, limit)

	var out struct {
		Data struct {
			ProductOfferV2 struct {
				Nodes []struct {
					ItemID         json.Number `json:"itemId"`
					ShopID         json.Number `json:"shopId"`
					ProductName    string      `json:"productName"`
					PriceMin       json.Number `json:"priceMin"`
					PriceMax       json.Number `json:"priceMax"`
					CommissionRate json.Number `json:"commissionRate"`
					ImageURL       string      `json:"imageUrl"`
					OfferLink      string      `json:"offerLink"`
					ProductLink    string      `json:"productLink"`
					Sales          json.Number `json:"sales"`
					RatingStar     json.Number `json:"ratingStar"`
					ShopName       string      `json:"shopName"`
				} `json:"nodes"`
			} `json:"productOfferV2"`
		} `json:"data"`
		Errors []gqlError `json:"errors"`
	}
	if err := s.do(ctx, query, &out); err != nil {
		return nil, err
	}
	if len(out.Errors) > 0 {
		return nil, fmt.Errorf("shopee affiliate: %s", out.Errors[0].Message)
	}
	products := make([]Product, 0, len(out.Data.ProductOfferV2.Nodes))
	for _, n := range out.Data.ProductOfferV2.Nodes {
		products = append(products, Product{
			ItemID:         n.ItemID.String(),
			ShopID:         n.ShopID.String(),
			Name:           n.ProductName,
			PriceMin:       num(n.PriceMin),
			PriceMax:       num(n.PriceMax),
			CommissionRate: num(n.CommissionRate),
			ImageURL:       n.ImageURL,
			OfferLink:      n.OfferLink,
			ProductLink:    n.ProductLink,
			Sales:          int(num(n.Sales)),
			RatingStar:     num(n.RatingStar),
			ShopName:       n.ShopName,
		})
	}
	return products, nil
}

// ShortLink runs the generateShortLink mutation.
func (s *Shopee) ShortLink(ctx context.Context, originURL string, subIDs []string) (string, error) {
	subs := "[]"
	if len(subIDs) > 0 {
		b, _ := json.Marshal(subIDs)
		subs = string(b)
	}
	query := fmt.Sprintf(`mutation { generateShortLink(input: { originUrl: %q, subIds: %s }) { shortLink } }`, originURL, subs)

	var out struct {
		Data struct {
			GenerateShortLink struct {
				ShortLink string `json:"shortLink"`
			} `json:"generateShortLink"`
		} `json:"data"`
		Errors []gqlError `json:"errors"`
	}
	if err := s.do(ctx, query, &out); err != nil {
		return "", err
	}
	if len(out.Errors) > 0 {
		return "", fmt.Errorf("shopee affiliate: %s", out.Errors[0].Message)
	}
	if out.Data.GenerateShortLink.ShortLink == "" {
		return "", fmt.Errorf("shopee affiliate: empty short link")
	}
	return out.Data.GenerateShortLink.ShortLink, nil
}

func num(n json.Number) float64 {
	f, _ := n.Float64()
	return f
}
