package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client talks to any OpenAI-compatible chat-completions server. That's
// the one API shape shared by Ollama, llama.cpp's own server, LM Studio,
// and vLLM — which covers every common way of serving a model trained or
// converted with Unsloth, regardless of which of those you actually run.
// It's entirely optional: if nothing answers, the wizard falls back to
// the static question set instead of blocking.
type Client struct {
	BaseURL string // e.g. http://localhost:11434 or http://localhost:8080 — /v1/... appended automatically
	Model   string
	HTTP    *http.Client
}

func NewClient(baseURL, model string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Model:   model,
		HTTP:    &http.Client{Timeout: 20 * time.Second},
	}
}

func (c *Client) v1(path string) string {
	base := c.BaseURL
	if !strings.HasSuffix(base, "/v1") {
		base += "/v1"
	}
	return base + path
}

// Available does a fast, short-timeout check against the standard
// /v1/models listing every OpenAI-compatible server exposes, so the
// wizard never hangs waiting on a server that isn't running. The reason
// string is non-empty on failure, so the caller can show *why* — not just
// that it failed.
func (c *Client) Available(ctx context.Context) (bool, string) {
	ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.v1("/models"), nil)
	if err != nil {
		return false, err.Error()
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Sprintf("server responded %d", resp.StatusCode)
	}
	return true, ""
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string        `json:"model"`
	Messages       []chatMessage `json:"messages"`
	ResponseFormat struct {
		Type string `json:"type"`
	} `json:"response_format"`
	Stream bool `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

// AskQuestion asks the local model for the next structured question via
// the standard chat-completions shape, using JSON mode so the response is
// constrained rather than trusted on faith. The result is validated
// before use; the caller falls back to the static set on any error.
func (c *Client) AskQuestion(ctx context.Context, systemContext, phaseKey string) (Question, error) {
	instruction := fmt.Sprintf(
		"%s\n\nCurrent phase: %s\nRespond with ONLY valid JSON of the shape "+
			"{\"question\":string,\"options\":[{\"label\":string,\"explanation\":string}],\"allow_custom\":bool}. "+
			"No prose, no markdown fences.",
		systemContext, phaseKey,
	)

	reqBody := chatRequest{
		Model:    c.Model,
		Messages: []chatMessage{{Role: "user", Content: instruction}},
		Stream:   false,
	}
	reqBody.ResponseFormat.Type = "json_object"

	body, err := json.Marshal(reqBody)
	if err != nil {
		return Question{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.v1("/chat/completions"), bytes.NewReader(body))
	if err != nil {
		return Question{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Question{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// Check the status before decoding: a 400/500 that still returns
		// valid JSON (e.g. an empty choices array) would otherwise be
		// misreported as "model returned no choices".
		return Question{}, fmt.Errorf("server responded %d", resp.StatusCode)
	}

	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return Question{}, err
	}
	if len(cr.Choices) == 0 {
		return Question{}, fmt.Errorf("model returned no choices")
	}

	var q Question
	if err := json.Unmarshal([]byte(cr.Choices[0].Message.Content), &q); err != nil {
		return Question{}, fmt.Errorf("model returned invalid JSON: %w", err)
	}
	if !q.Valid() {
		return Question{}, fmt.Errorf("model response failed schema validation")
	}
	return q, nil
}
