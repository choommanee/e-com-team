package agents

import (
	"context"
	"encoding/json"
	"fmt"

	"ecomteam/internal/llm"
)

// Prompt (prompt engineer) converts the concept into a detailed English image prompt.
type Prompt struct{}

func (Prompt) Name() string { return NamePrompt }

func (Prompt) Run(ctx context.Context, d *StageData, client llm.Client, p Progress) error {
	const task = "Writing image prompt"
	p(NamePrompt, 35, task)

	// The image prompt is always English (model requirement) but must instruct
	// the exact on-image text, which may be Thai.
	system := "agent:prompt\n" +
		"You are a prompt engineer for an image model. Produce ONE detailed English prompt for a " +
		"professional e-commerce product photo. Specify lighting, camera angle, background, and " +
		"explicitly state the exact text to render on the image (keep that text in its original " +
		`language). Respond as JSON: {"image_prompt":"..."}`
	user := fmt.Sprintf("Product: %s\nHeadline to print: %s\nPromo to print: %s\nLayout: %s\nColor tone: %s",
		d.ProductName, d.Listing.Headline, d.Listing.Promotion, d.Listing.Layout, d.Listing.ColorTone)

	out, err := client.Chat(ctx, system, user)
	if err != nil {
		return fmt.Errorf("prompt: %w", err)
	}
	var parsed struct {
		ImagePrompt string `json:"image_prompt"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return fmt.Errorf("prompt: bad JSON: %w", err)
	}
	if parsed.ImagePrompt == "" {
		return fmt.Errorf("prompt: empty image prompt")
	}
	d.Listing.ImagePrompt = parsed.ImagePrompt
	p(NamePrompt, 100, "Done")
	return nil
}
