package llm

import (
	"context"
	"errors"
	"testing"
)

type failImage struct{ chatOut string }

func (f failImage) Chat(context.Context, string, string) (string, error) { return f.chatOut, nil }
func (failImage) Image(context.Context, string) ([]byte, error) {
	return nil, errors.New("image model not available")
}

func TestImageFallbackUsesSecondaryOnError(t *testing.T) {
	fb := WithImageFallback(failImage{chatOut: "hi"}, NewMock())

	// Chat comes from primary.
	out, _ := fb.Chat(context.Background(), "s", "u")
	if out != "hi" {
		t.Fatalf("chat should use primary, got %q", out)
	}

	// Image falls back to the mock PNG.
	img, err := fb.Image(context.Background(), "x")
	if err != nil {
		t.Fatalf("fallback should succeed, got %v", err)
	}
	if string(img[:8]) != "\x89PNG\r\n\x1a\n" {
		t.Fatalf("expected PNG from fallback")
	}
}

func TestImageFallbackPrefersPrimary(t *testing.T) {
	// When primary image works, fallback is not used. Mock primary always works.
	fb := WithImageFallback(NewMock(), failImage{})
	if _, err := fb.Image(context.Background(), "x"); err != nil {
		t.Fatalf("primary works, should not error: %v", err)
	}
}
