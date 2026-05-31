// Package llm abstracts text + image generation behind a single interface so
// the rest of the app can run against OpenAI or a deterministic mock.
package llm

import "context"

// Client generates text (chat) and images.
type Client interface {
	// Chat sends a system + user prompt and returns the assistant's text.
	Chat(ctx context.Context, system, user string) (string, error)
	// Image generates an image from a prompt and returns raw PNG bytes.
	Image(ctx context.Context, prompt string) ([]byte, error)
}
