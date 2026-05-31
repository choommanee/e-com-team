package llm

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGeminiChatParsesResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "gemini-test:generateContent") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"ok\":true}"}]}}]}`))
	}))
	defer srv.Close()

	g := NewGemini("k", "gemini-test", "img-test")
	g.baseURL = srv.URL

	out, err := g.Chat(context.Background(), "sys", "user")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out != `{"ok":true}` {
		t.Fatalf("unexpected chat output: %q", out)
	}
}

func TestGeminiImageDecodesInlineData(t *testing.T) {
	pngB64 := base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\nfake"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"image/png","data":"` + pngB64 + `"}}]}}]}`))
	}))
	defer srv.Close()

	g := NewGemini("k", "c", "img")
	g.baseURL = srv.URL

	img, err := g.Image(context.Background(), "a cat")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.HasPrefix(string(img), "\x89PNG") {
		t.Fatalf("expected decoded PNG bytes, got %q", img[:8])
	}
}

func TestGeminiChatSurfacesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"error":{"message":"quota exceeded"}}`))
	}))
	defer srv.Close()

	g := NewGemini("k", "c", "img")
	g.baseURL = srv.URL
	if _, err := g.Chat(context.Background(), "s", "u"); err == nil || !strings.Contains(err.Error(), "quota") {
		t.Fatalf("expected quota error, got %v", err)
	}
}
