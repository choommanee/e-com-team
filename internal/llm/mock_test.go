package llm

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestMockChatReturnsValidJSONPerAgent(t *testing.T) {
	m := NewMock()
	cases := []string{"agent:benefit", "agent:promo", "agent:design", "agent:prompt", "agent:qc"}
	for _, tag := range cases {
		out, err := m.Chat(context.Background(), "system "+tag, "any product")
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", tag, err)
		}
		var v map[string]any
		if err := json.Unmarshal([]byte(out), &v); err != nil {
			t.Fatalf("%s: response is not valid JSON: %v (%q)", tag, err, out)
		}
		if len(v) == 0 {
			t.Fatalf("%s: expected non-empty JSON object", tag)
		}
	}
}

func TestMockChatIsDeterministic(t *testing.T) {
	m := NewMock()
	a, _ := m.Chat(context.Background(), "agent:benefit", "x")
	b, _ := m.Chat(context.Background(), "agent:benefit", "x")
	if a != b {
		t.Fatalf("expected deterministic output, got %q vs %q", a, b)
	}
}

func TestMockImageReturnsPNG(t *testing.T) {
	m := NewMock()
	data, err := m.Image(context.Background(), "anything")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(string(data[:8]), "\x89PNG") {
		t.Fatalf("expected PNG magic header, got %v", data[:8])
	}
}
