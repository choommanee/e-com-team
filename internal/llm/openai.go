package llm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAI implements Client against the OpenAI HTTP API (chat + image).
type OpenAI struct {
	apiKey     string
	chatModel  string
	imageModel string
	baseURL    string
	http       *http.Client
}

// NewOpenAI builds an OpenAI client.
func NewOpenAI(apiKey, chatModel, imageModel string) *OpenAI {
	return &OpenAI{
		apiKey:     apiKey,
		chatModel:  chatModel,
		imageModel: imageModel,
		baseURL:    "https://api.openai.com/v1",
		http:       &http.Client{Timeout: 90 * time.Second},
	}
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	Temperature    float64         `json:"temperature"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *apiError `json:"error"`
}

type apiError struct {
	Message string `json:"message"`
}

// Chat calls the chat completions endpoint requesting a JSON object response.
func (o *OpenAI) Chat(ctx context.Context, system, user string) (string, error) {
	reqBody := chatRequest{
		Model: o.chatModel,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
		Temperature:    0.7,
	}
	var out chatResponse
	if err := o.post(ctx, "/chat/completions", reqBody, &out); err != nil {
		return "", err
	}
	if out.Error != nil {
		return "", fmt.Errorf("openai chat: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("openai chat: empty response")
	}
	return out.Choices[0].Message.Content, nil
}

type imageRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	N      int    `json:"n"`
	Size   string `json:"size"`
}

type imageResponse struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
	} `json:"data"`
	Error *apiError `json:"error"`
}

// Image calls the image generation endpoint and returns decoded PNG bytes.
func (o *OpenAI) Image(ctx context.Context, prompt string) ([]byte, error) {
	reqBody := imageRequest{
		Model:  o.imageModel,
		Prompt: prompt,
		N:      1,
		Size:   "1024x1024",
	}
	var out imageResponse
	if err := o.post(ctx, "/images/generations", reqBody, &out); err != nil {
		return nil, err
	}
	if out.Error != nil {
		return nil, fmt.Errorf("openai image: %s", out.Error.Message)
	}
	if len(out.Data) == 0 || out.Data[0].B64JSON == "" {
		return nil, fmt.Errorf("openai image: empty response")
	}
	return base64.StdEncoding.DecodeString(out.Data[0].B64JSON)
}

func (o *OpenAI) post(ctx context.Context, path string, body, out any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		// Try to decode an error envelope; fall back to raw body.
		if jsonErr := json.Unmarshal(data, out); jsonErr == nil {
			return nil // caller inspects out.Error
		}
		return fmt.Errorf("openai %s: status %d: %s", path, resp.StatusCode, string(data))
	}
	return json.Unmarshal(data, out)
}
