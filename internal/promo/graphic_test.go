package promo

import (
	"os"
	"strings"
	"testing"
)

func TestBuildProducesPNG(t *testing.T) {
	b, err := NewBuilder()
	if err != nil {
		t.Fatalf("builder: %v", err)
	}
	// No image URL → solid backdrop fallback, still renders.
	out, err := b.Build("", "ครีมกันแดด SPF50 ลดแรง", 199, 12)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !strings.HasPrefix(string(out[:8]), "\x89PNG") {
		t.Fatalf("expected PNG output")
	}
	// Optionally dump for manual inspection.
	if os.Getenv("DUMP_PROMO") != "" {
		_ = os.WriteFile("/tmp/promo-sample.png", out, 0o644)
	}
}
