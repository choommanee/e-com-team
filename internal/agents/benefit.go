package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"ecomteam/internal/llm"
)

// Benefit (E-commerce strategist) extracts 3 core selling points.
type Benefit struct{}

func (Benefit) Name() string { return NameBenefit }

func (Benefit) Run(ctx context.Context, d *StageData, client llm.Client, p Progress) error {
	const task = "Analyzing selling points"
	p(NameBenefit, 35, task)

	system := "agent:benefit\n" +
		"You are an e-commerce strategist. Given a product name, identify exactly 3 core " +
		"selling points that solve a real customer problem. " + langInstruction(d.Lang) +
		` Respond as JSON: {"selling_points":["...","...","..."]}`
	user := "Product: " + d.ProductName

	out, err := client.Chat(ctx, system, user)
	if err != nil {
		return fmt.Errorf("benefit: %w", err)
	}
	var parsed struct {
		SellingPoints []string `json:"selling_points"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return fmt.Errorf("benefit: bad JSON: %w", err)
	}
	if len(parsed.SellingPoints) == 0 {
		return fmt.Errorf("benefit: no selling points returned")
	}
	d.Listing.SellingPoints = parsed.SellingPoints
	p(NameBenefit, 100, "Done")
	return nil
}
