package agents

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ecomteam/internal/llm"
)

// Promo (sales copywriter) writes a headline + promotion text.
type Promo struct{}

func (Promo) Name() string { return NamePromo }

func (Promo) Run(ctx context.Context, d *StageData, client llm.Client, p Progress) error {
	const task = "Writing promo copy"
	p(NamePromo, 35, task)

	system := "agent:promo\n" +
		"You are a sales copywriter. Using the product's selling points, write one short " +
		"punchy headline and one promotional message (e.g. free shipping, buy-1-get-1) suitable " +
		"to print on a product image. " + langInstruction(d.Lang) +
		` Respond as JSON: {"headline":"...","promotion":"..."}`
	user := fmt.Sprintf("Product: %s\nSelling points: %s",
		d.ProductName, strings.Join(d.Listing.SellingPoints, "; "))

	out, err := client.Chat(ctx, system, user)
	if err != nil {
		return fmt.Errorf("promo: %w", err)
	}
	var parsed struct {
		Headline  string `json:"headline"`
		Promotion string `json:"promotion"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return fmt.Errorf("promo: bad JSON: %w", err)
	}
	if parsed.Headline == "" {
		return fmt.Errorf("promo: empty headline")
	}
	d.Listing.Headline = parsed.Headline
	d.Listing.Promotion = parsed.Promotion
	p(NamePromo, 100, "Done")
	return nil
}
