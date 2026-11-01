// Package ai wraps Google Gemini via the official Go SDK (google.golang.org/genai).
package ai

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"google.golang.org/genai"
)

var (
	// ErrDisabled is returned when no API key is configured.
	ErrDisabled = errors.New("AI is not configured")
	// ErrBusy is returned when every configured model is overloaded or timed out.
	ErrBusy = errors.New("AI models are busy")
)

// Message is one chat turn. Role is "user" or "model" (Gemini's naming).
type Message struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type Client struct {
	models  []string // tried in order; later ones are fallbacks
	client  *genai.Client
	initErr error
}

// NewGemini creates a client. models is a comma-separated list, e.g.
// "gemini-3.8-flash,gemini-flash-latest": when a model is overloaded the next is tried.
func NewGemini(apiKey, models string) *Client {
	return newClient(apiKey, models, "")
}

// newClient allows tests to point the SDK at a fake server via baseURL.
func newClient(apiKey, models, baseURL string) *Client {
	c := &Client{}
	for _, m := range strings.Split(models, ",") {
		if m = strings.TrimSpace(m); m != "" {
			c.models = append(c.models, m)
		}
	}
	if apiKey == "" || len(c.models) == 0 {
		return c
	}
	c.client, c.initErr = genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:      apiKey,
		Backend:     genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{BaseURL: baseURL},
	})
	return c
}

// Enabled reports whether an API key and model are configured.
func (c *Client) Enabled() bool { return c.client != nil || c.initErr != nil }

// Chat sends the conversation with a system instruction and returns the model's reply.
func (c *Client) Chat(ctx context.Context, system string, history []Message) (string, error) {
	if !c.Enabled() {
		return "", ErrDisabled
	}
	if c.initErr != nil {
		return "", fmt.Errorf("gemini: %w", c.initErr)
	}

	contents := make([]*genai.Content, 0, len(history))
	for _, m := range history {
		contents = append(contents, genai.NewContentFromText(m.Text, genai.Role(m.Role)))
	}
	cfg := &genai.GenerateContentConfig{
		// Low thinking keeps chat replies fast; the default level can take a minute or more.
		ThinkingConfig: &genai.ThinkingConfig{ThinkingLevel: genai.ThinkingLevelLow},
	}
	if system != "" {
		cfg.SystemInstruction = genai.NewContentFromText(system, genai.RoleUser)
	}

	var lastErr error
	for _, model := range c.models {
		resp, err := c.client.Models.GenerateContent(ctx, model, contents, cfg)
		if err == nil {
			reply := strings.TrimSpace(resp.Text())
			if reply == "" {
				return "", errors.New("gemini returned an empty answer")
			}
			return reply, nil
		}
		if !retryable(err) || ctx.Err() != nil {
			return "", fmt.Errorf("gemini: %w", err)
		}
		log.Printf("gemini model %s unavailable, trying next: %v", model, err)
		lastErr = err
	}
	return "", fmt.Errorf("%w: %v", ErrBusy, lastErr)
}

// retryable reports whether another model might succeed: overload, rate limit or server timeout.
func retryable(err error) bool {
	var apiErr genai.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.Code {
	case 429, 500, 502, 503, 504:
		return true
	}
	return false
}
