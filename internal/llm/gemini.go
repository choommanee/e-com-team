package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Gemini implements Client against Google's Generative Language API
// (aistudio.google.com). Text uses generateContent with JSON output; images
// use an image-capable Gemini model returning inline base64 data.
type Gemini struct {
	apiKey     string
	chatModel  string
	imageModel string
	baseURL    string
	http       *http.Client
}

// NewGemini builds a Gemini client.
func NewGemini(apiKey, chatModel, imageModel string) *Gemini {
	return &Gemini{
		apiKey:     apiKey,
		chatModel:  chatModel,
		imageModel: imageModel,
		baseURL:    "https://generativelanguage.googleapis.com/v1beta",
		http:       &http.Client{Timeout: 90 * time.Second},
	}
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Chat sends a system + user prompt and requests a JSON object response.
func (g *Gemini) Chat(ctx context.Context, system, user string) (string, error) {
	body := map[string]any{
		"systemInstruction": map[string]any{"parts": []geminiPart{{Text: system}}},
		"contents":          []geminiContent{{Role: "user", Parts: []geminiPart{{Text: user}}}},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
			"temperature":      0.7,
		},
	}
	var out geminiResponse
	if err := g.post(ctx, g.chatModel, body, &out); err != nil {
		return "", err
	}
	if out.Error != nil {
		return "", fmt.Errorf("gemini chat: %s", out.Error.Message)
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini chat: empty response")
	}
	return out.Candidates[0].Content.Parts[0].Text, nil
}

// Image generates an image and returns the decoded bytes from inline data.
func (g *Gemini) Image(ctx context.Context, prompt string) ([]byte, error) {
	body := map[string]any{
		"contents": []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
		"generationConfig": map[string]any{
			"responseModalities": []string{"TEXT", "IMAGE"},
		},
	}
	var out geminiResponse
	if err := g.post(ctx, g.imageModel, body, &out); err != nil {
		return nil, err
	}
	if out.Error != nil {
		return nil, fmt.Errorf("gemini image: %s", out.Error.Message)
	}
	for _, c := range out.Candidates {
		for _, p := range c.Content.Parts {
			if p.InlineData != nil && p.InlineData.Data != "" {
				return base64.StdEncoding.DecodeString(p.InlineData.Data)
			}
		}
	}
	return nil, fmt.Errorf("gemini image: no image in response")
}

func (g *Gemini) post(ctx context.Context, model string, body, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("%s/models/%s:generateContent?key=%s", g.baseURL, model, url.QueryEscape(g.apiKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		if jsonErr := json.Unmarshal(data, out); jsonErr == nil {
			return nil // caller inspects out.Error
		}
		return fmt.Errorf("gemini %s: status %d: %s", model, resp.StatusCode, string(data))
	}
	return json.Unmarshal(data, out)
}
