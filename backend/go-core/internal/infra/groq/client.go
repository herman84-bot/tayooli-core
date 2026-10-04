package groq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Model fallback chain — primary → fallback1 → fallback2.
// Updated Aug 2026: llama-3.3-70b and llama-3.1-8b deprecated.
// Using GPT-OSS models (paid but cheapest on Groq).
var modelFallbackChain = []string{
	"openai/gpt-oss-20b",
	"qwen/qwen3.6-27b",
}

// ChatMessage represents a message in the conversation.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the request to the Groq API.
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
	TopP        float64       `json:"top_p"`
}

// ChatResponse is the response from the Groq API.
type ChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Model  string `json:"model"`
	Usage  struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// Client is the Groq API client.
type Client struct {
	apiKey     string
	httpClient *http.Client
	baseURL    string

	// Rate limiter: token bucket per IP or global.
	mu          sync.Mutex
	rateTokens  float64
	rateMax     float64
	rateRefill  time.Duration
	lastRefill  time.Time
}

// NewClient creates a new Groq API client.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    "https://api.groq.com/openai/v1",
		rateTokens: 30, // 30 tokens = 30 requests burst
		rateMax:    30,
		rateRefill: time.Minute,
		lastRefill: time.Now(),
	}
}

// ChatCompletion sends a chat completion request with auto-model fallback.
func (c *Client) ChatCompletion(ctx context.Context, messages []ChatMessage) (string, string, error) {
	// Rate limit check
	if !c.allow() {
		return "", "", fmt.Errorf("rate limit exceeded: max 30 messages per minute")
	}

	// Sanitize input messages
	sanitized := make([]ChatMessage, len(messages))
	for i, m := range messages {
		sanitized[i] = ChatMessage{
			Role:    m.Role,
			Content: SanitizeInput(m.Content),
		}
	}

	reqBody := ChatRequest{
		Messages:    sanitized,
		Temperature: 0.7,
		MaxTokens:   1024,
		TopP:        0.9,
	}

	var lastErr error
	for _, model := range modelFallbackChain {
		reqBody.Model = model
		reply, err := c.doRequest(ctx, reqBody)
		if err == nil {
			return reply, model, nil
		}
		lastErr = err
		// If context cancelled, don't try next model
		if ctx.Err() != nil {
			return "", "", ctx.Err()
		}
	}

	return "", "", fmt.Errorf("all models failed: %w", lastErr)
}

// doRequest sends a single request to the Groq API.
func (c *Client) doRequest(ctx context.Context, reqBody ChatRequest) (string, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("groq request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		return "", fmt.Errorf("rate limited by Groq (429)")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("groq returned %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("groq error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("groq returned no choices")
	}

	reply := SanitizeOutput(chatResp.Choices[0].Message.Content)
	return reply, nil
}

// allow implements a simple token bucket rate limiter.
func (c *Client) allow() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(c.lastRefill)
	if elapsed >= c.rateRefill {
		c.rateTokens = c.rateMax
		c.lastRefill = now
	}

	if c.rateTokens < 1 {
		return false
	}
	c.rateTokens--
	return true
}

// ──────────────────────────────────────────────────────────────────────────────
// Sanitization
// ──────────────────────────────────────────────────────────────────────────────

var (
	htmlTagRegex = regexp.MustCompile(`<[^>]*>`)
	scriptRegex  = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
)

// SanitizeInput strips potentially dangerous content from user input.
func SanitizeInput(input string) string {
	// Remove script tags
	s := scriptRegex.ReplaceAllString(input, "")
	// Remove HTML tags
	s = htmlTagRegex.ReplaceAllString(s, "")
	// Trim whitespace
	s = strings.TrimSpace(s)
	// Limit length (max 2000 chars)
	if len(s) > 2000 {
		s = s[:2000]
	}
	return s
}

// SanitizeOutput cleans AI response for safe display.
func SanitizeOutput(output string) string {
	// Remove any script tags that might slip through
	s := scriptRegex.ReplaceAllString(output, "")
	// Strip markdown formatting
	s = stripMarkdown(s)
	s = strings.TrimSpace(s)
	return s
}

// stripMarkdown removes common markdown formatting from AI responses.
func stripMarkdown(s string) string {
	// Remove bold: **text** → text
	for strings.Contains(s, "**") {
		start := strings.Index(s, "**")
		end := strings.Index(s[start+2:], "**")
		if end == -1 {
			break
		}
		s = s[:start] + s[start+2:start+2+end] + s[start+2+end+2:]
	}
	// Remove italic: *text* → text
	for strings.Contains(s, "*") {
		start := strings.Index(s, "*")
		end := strings.Index(s[start+1:], "*")
		if end == -1 {
			break
		}
		s = s[:start] + s[start+1:start+1+end] + s[start+1+end+1:]
	}
	// Remove headers: ### text → text
	for strings.HasPrefix(s, "#") {
		s = strings.TrimPrefix(s, "#")
	}
	return strings.TrimSpace(s)
}
