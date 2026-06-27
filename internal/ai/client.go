package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	ErrNotConfigured = errors.New("ai provider is not configured")
	ErrUnauthorized  = errors.New("ai authentication failed")
)

type Config struct {
	BaseURL string
	APIKey  string
	Model   string
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

func New(cfg Config) *Client {
	return &Client{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  strings.TrimSpace(cfg.APIKey),
		model:   strings.TrimSpace(cfg.Model),
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (c *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	if c == nil || c.baseURL == "" || c.apiKey == "" || c.model == "" {
		return "", ErrNotConfigured
	}
	payload := chatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: 0.2,
		MaxTokens:   500,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", ErrUnauthorized
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("ai provider returned status %d", resp.StatusCode)
	}
	var result chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		data, _ := io.ReadAll(resp.Body)
		if len(data) > 0 {
			return "", fmt.Errorf("decode ai response: %w: %s", err, string(data))
		}
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", errors.New("ai provider returned no choices")
	}
	content := strings.TrimSpace(result.Choices[0].Message.Content)
	if content == "" {
		return "", errors.New("ai provider returned empty content")
	}
	return content, nil
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}
