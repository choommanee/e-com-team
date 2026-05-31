package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"ecomteam/internal/llm"
)

// Design (graphic designer) decides layout + color tone.
type Design struct{}

func (Design) Name() string { return NameDesign }

func (Design) Run(ctx context.Context, d *StageData, client llm.Client, p Progress) error {
	const task = "Designing layout"
	p(NameDesign, 35, task)

	system := "agent:design\n" +
		"You are a graphic designer for marketplace listings. Decide a layout (where the product, " +
		"headline and promo badge sit) and an eye-catching color tone. " + langInstruction(d.Lang) +
		` Respond as JSON: {"layout":"...","color_tone":"..."}`
	user := fmt.Sprintf("Product: %s\nHeadline: %s\nPromotion: %s",
		d.ProductName, d.Listing.Headline, d.Listing.Promotion)

	out, err := client.Chat(ctx, system, user)
	if err != nil {
		return fmt.Errorf("design: %w", err)
	}
	var parsed struct {
		Layout    string `json:"layout"`
		ColorTone string `json:"color_tone"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return fmt.Errorf("design: bad JSON: %w", err)
	}
	d.Listing.Layout = parsed.Layout
	d.Listing.ColorTone = parsed.ColorTone
	p(NameDesign, 100, "Done")
	return nil
}
