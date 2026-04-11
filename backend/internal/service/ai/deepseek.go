package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DeepSeekClient communicates with the DeepSeek API (OpenAI-compatible).
type DeepSeekClient struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

func NewDeepSeekClient(apiKey, baseURL string) *DeepSeekClient {
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	return &DeepSeekClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 120 * time.Second},
	}
}

// IsConfigured returns true if the API key is set.
func (c *DeepSeekClient) IsConfigured() bool {
	return c.apiKey != ""
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Chat sends a message to DeepSeek and returns the response.
func (c *DeepSeekClient) Chat(ctx context.Context, model, systemPrompt, userPrompt string, temperature float64, maxTokens int) (string, int, error) {
	if !c.IsConfigured() {
		return "", 0, fmt.Errorf("DeepSeek API key not configured")
	}

	req := chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", 0, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", 0, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return "", 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("DeepSeek API error %d: %s", resp.StatusCode, string(respBody[:min(300, len(respBody))]))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", 0, fmt.Errorf("unmarshal response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", 0, fmt.Errorf("no response from DeepSeek")
	}

	return chatResp.Choices[0].Message.Content, chatResp.Usage.TotalTokens, nil
}

// ChatJSON sends a prompt and parses the JSON response into target.
func (c *DeepSeekClient) ChatJSON(ctx context.Context, model, systemPrompt, userPrompt string, target any) (int, error) {
	content, tokens, err := c.Chat(ctx, model, systemPrompt, userPrompt, 0.3, 3000)
	if err != nil {
		return 0, err
	}

	// Strip markdown code fences if present
	content = stripCodeFences(content)

	if err := json.Unmarshal([]byte(content), target); err != nil {
		return tokens, fmt.Errorf("parse AI response as JSON: %w\nRaw: %s", err, content[:min(500, len(content))])
	}
	return tokens, nil
}

func stripCodeFences(s string) string {
	// Remove ```json ... ``` wrapping
	if len(s) > 7 && s[:7] == "```json" {
		s = s[7:]
	} else if len(s) > 3 && s[:3] == "```" {
		s = s[3:]
	}
	if len(s) > 3 && s[len(s)-3:] == "```" {
		s = s[:len(s)-3]
	}
	return s
}
