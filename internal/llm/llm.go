// Package llm defines a small provider interface so the orchestrator never
// depends on a specific vendor. Two implementations are provided:
//
//   - AnthropicProvider: calls the real Claude API (requires ANTHROPIC_API_KEY)
//   - MockProvider: deterministic canned responses, used when no API key is
//     configured, so the whole system is runnable and demoable offline.
package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Provider generates a single completion given a system prompt (the
// persona) and the running transcript of the debate so far.
type Provider interface {
	Complete(systemPrompt string, transcript []Message) (string, error)
}

type Message struct {
	Role    string // "user" or "assistant"
	Content string
}

// ---------------------------------------------------------------------
// Anthropic provider
// ---------------------------------------------------------------------

type AnthropicProvider struct {
	APIKey string
	Model  string
	Client *http.Client
}

func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		APIKey: apiKey,
		Model:  "claude-sonnet-4-6",
		Client: &http.Client{Timeout: 30 * time.Second},
	}
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string              `json:"model"`
	MaxTokens int                 `json:"max_tokens"`
	System    string              `json:"system,omitempty"`
	Messages  []anthropicMessage  `json:"messages"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (p *AnthropicProvider) Complete(systemPrompt string, transcript []Message) (string, error) {
	msgs := make([]anthropicMessage, 0, len(transcript))
	for _, m := range transcript {
		msgs = append(msgs, anthropicMessage{Role: m.Role, Content: m.Content})
	}
	if len(msgs) == 0 {
		msgs = append(msgs, anthropicMessage{Role: "user", Content: "Please open the debate."})
	}

	body, _ := json.Marshal(anthropicRequest{
		Model:     p.Model,
		MaxTokens: 400,
		System:    systemPrompt,
		Messages:  msgs,
	})

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var parsed anthropicResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("decode anthropic response: %w (body: %s)", err, raw)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("anthropic API error: %s", parsed.Error.Message)
	}
	if len(parsed.Content) == 0 {
		return "", fmt.Errorf("anthropic returned no content")
	}
	return parsed.Content[0].Text, nil
}

// ---------------------------------------------------------------------
// Mock provider — deterministic, offline, used for demos / tests
// ---------------------------------------------------------------------

type MockProvider struct{}

func NewMockProvider() *MockProvider { return &MockProvider{} }

func (p *MockProvider) Complete(systemPrompt string, transcript []Message) (string, error) {
	round := len(transcript)
	switch {
	case round == 0:
		return fmt.Sprintf("[%s] Let me open with the strongest version of my position, grounded in first principles.", personaLabel(systemPrompt)), nil
	case round%3 == 0:
		return fmt.Sprintf("[%s] I want to concede one point from the previous speaker while sharpening my core argument.", personaLabel(systemPrompt)), nil
	default:
		return fmt.Sprintf("[%s] Responding directly to that: I think the previous point overlooks a key trade-off worth naming.", personaLabel(systemPrompt)), nil
	}
}

func personaLabel(systemPrompt string) string {
	if len(systemPrompt) > 24 {
		return systemPrompt[:24] + "…"
	}
	return systemPrompt
}
