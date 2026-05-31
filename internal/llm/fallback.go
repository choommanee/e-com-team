package llm

import (
	"context"
	"log"
)

// ImageFallback wraps a primary client: text always uses the primary, but if
// image generation fails (e.g. gpt-image-1 needs org verification, or no
// credit), it falls back to the secondary client's image so the pipeline still
// completes. The fallback event is logged.
type ImageFallback struct {
	primary  Client
	fallback Client
}

// WithImageFallback returns a client that delegates Chat to primary and Image
// to primary-then-fallback.
func WithImageFallback(primary, fallback Client) *ImageFallback {
	return &ImageFallback{primary: primary, fallback: fallback}
}

// Chat always uses the primary client.
func (f *ImageFallback) Chat(ctx context.Context, system, user string) (string, error) {
	return f.primary.Chat(ctx, system, user)
}

// Image tries the primary; on error it logs and returns the fallback image.
func (f *ImageFallback) Image(ctx context.Context, prompt string) ([]byte, error) {
	img, err := f.primary.Image(ctx, prompt)
	if err == nil {
		return img, nil
	}
	log.Printf("llm: real image generation failed (%v); using placeholder fallback", err)
	return f.fallback.Image(ctx, prompt)
}
