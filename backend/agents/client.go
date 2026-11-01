// Package agents talks to a local Ollama server to role-play the
// citizens of one chunk (docs/API.md §8 "Воля жителей", §10 memory).
// The integration is deliberately forgiving: when Ollama is offline or
// slow, every call degrades into a canned reply and a zero will-shift,
// so the game loop never blocks on an LLM.
package agents

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client is a thin wrapper over the Ollama /api/chat endpoint.
type Client struct {
	baseURL string
	model   string
	http    *http.Client
	enabled bool
}

// ErrDisabled means the client was told not to reach out (OLLAMA_DISABLE=1).
var ErrDisabled = errors.New("agents: ollama integration disabled")

// HTTPError reports a non-200 answer from the Ollama daemon.
type HTTPError struct{ Status int }

func (e *HTTPError) Error() string {
	return fmt.Sprintf("agents: ollama answered %d", e.Status)
}

// NewFromEnv builds the client from OLLAMA_URL / OLLAMA_MODEL env vars.
// Defaults: http://127.0.0.1:11434 and llama3.2:3b. If OLLAMA_DISABLE=1
// the client stays constructed but disabled (chat answers with fallbacks).
func NewFromEnv() *Client {
	url := strings.TrimRight(os.Getenv("OLLAMA_URL"), "/")
	if url == "" {
		url = "http://127.0.0.1:11434"
	}
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "llama3.2:3b"
	}
	return &Client{
		baseURL: url,
		model:   model,
		http:    &http.Client{Timeout: 25 * time.Second},
		enabled: os.Getenv("OLLAMA_DISABLE") != "1",
	}
}

// Enabled reports whether outbound LLM calls are allowed at all.
func (c *Client) Enabled() bool { return c.enabled }

// chatRequest matches the Ollama /api/chat JSON body (non-streaming).
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Options  chatOptions   `json:"options"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatOptions struct {
	Temperature float64 `json:"temperature"`
	NumPredict  int     `json:"num_predict"`
}

type chatResponse struct {
	Message chatMessage `json:"message"`
}

// Chat sends one prompt pair (system + user) and returns the raw answer.
// Any transport/decoding failure surfaces as an error the caller maps to
// its fallback path; ctx bounds the wait even below the client timeout.
func (c *Client) Chat(ctx context.Context, system, user string) (string, error) {
	if !c.enabled {
		return "", ErrDisabled
	}
	body := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Stream:  false,
		Options: chatOptions{Temperature: 0.7, NumPredict: 220},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/chat", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", &HTTPError{Status: resp.StatusCode}
	}
	var out chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.Message.Content), nil
}
