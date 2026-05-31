package agents

import (
	"context"
	"fmt"

	"ecomteam/internal/llm"
)

// Studio (image generator) produces the final product image bytes.
// It does not write to disk — it stores PNG bytes in StageData.ImagePNG so the
// worker can persist them and assign a public URL.
type Studio struct{}

func (Studio) Name() string { return NameStudio }

func (Studio) Run(ctx context.Context, d *StageData, client llm.Client, p Progress) error {
	p(NameStudio, 25, "Generating image")

	png, err := client.Image(ctx, d.Listing.ImagePrompt)
	if err != nil {
		return fmt.Errorf("studio: %w", err)
	}
	if len(png) == 0 {
		return fmt.Errorf("studio: empty image")
	}
	d.ImagePNG = png
	p(NameStudio, 100, "Done")
	return nil
}
