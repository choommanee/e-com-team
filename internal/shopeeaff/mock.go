package shopeeaff

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// Mock returns realistic sample products and fake short links so the affiliate
// flow can be exercised without approved Shopee Affiliate credentials.
type Mock struct{}

// NewMock returns a mock client.
func NewMock() *Mock { return &Mock{} }

// sample products keyed loosely by keyword; values mimic real productOfferV2 shape.
func (m *Mock) SearchProducts(_ context.Context, keyword string, limit int) ([]Product, error) {
	if strings.TrimSpace(keyword) == "" {
		return nil, fmt.Errorf("keyword required")
	}
	base := []Product{
		{ItemID: "228xxxx01", ShopID: "1199001", Name: keyword + " รุ่นขายดี Premium ของแท้ 100%",
			PriceMin: 199, PriceMax: 299, CommissionRate: 0.12, Sales: 4820, RatingStar: 4.9,
			ShopName: "Official Store", ImageURL: "https://down-th.img.susercontent.com/file/sample1",
			OfferLink: "https://shopee.co.th/product/1199001/228000001"},
		{ItemID: "228xxxx02", ShopID: "1199002", Name: keyword + " เกรดส่งออก คุ้มราคา",
			PriceMin: 89, PriceMax: 89, CommissionRate: 0.08, Sales: 12750, RatingStar: 4.8,
			ShopName: "Mega Mall TH", ImageURL: "https://down-th.img.susercontent.com/file/sample2",
			OfferLink: "https://shopee.co.th/product/1199002/228000002"},
		{ItemID: "228xxxx03", ShopID: "1199003", Name: keyword + " เซ็ตสุดคุ้ม แถมฟรีของแถม",
			PriceMin: 349, PriceMax: 590, CommissionRate: 0.15, Sales: 980, RatingStar: 4.7,
			ShopName: "Lucky Shop", ImageURL: "https://down-th.img.susercontent.com/file/sample3",
			OfferLink: "https://shopee.co.th/product/1199003/228000003"},
	}
	for i := range base {
		base[i].ProductLink = base[i].OfferLink
	}
	if limit > 0 && limit < len(base) {
		base = base[:limit]
	}
	return base, nil
}

func (m *Mock) ShortLink(_ context.Context, originURL string, subIDs []string) (string, error) {
	if originURL == "" {
		return "", fmt.Errorf("originURL required")
	}
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return "https://shope.ee/" + hex.EncodeToString(b), nil
}
