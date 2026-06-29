package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nostalgia296/ocs-ai/internal/llm"
)

func init() {
	llm.RegisterProvider("chat_completions", Call)
}

// Call implements the Chat Completions (OpenAI-compatible) API.
func Call(ctx context.Context, httpClient *http.Client, model llm.ModelConfigForCall, messages []llm.Message) (*llm.ModelCallResult, error) {
	chatMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		chatMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	reqBody := map[string]interface{}{
		"model":      model.ModelName,
		"messages":   chatMessages,
		"max_tokens": model.MaxTokens,
		"stream":     false,
	}

	// DeepSeek thinking mode: enable thinking, remove temperature/top_p
	if model.DSThinkingMode {
		reqBody["thinking"] = map[string]string{"type": "enabled"}
	} else {
		reqBody["temperature"] = model.Temperature
		reqBody["top_p"] = model.TopP
	}

	respBody, err := doRequest(ctx, httpClient, model.BaseURL, "/v1/chat/completions", model.APIKey, reqBody)
	if err != nil {
		return nil, err
	}

	var chatResp ChatCompletionsResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, err
	}

	return &llm.ModelCallResult{
		Answer:    ExtractText(chatResp),
		Reasoning: ExtractReasoning(chatResp),
		Usage:     ExtractUsage(chatResp),
	}, nil
}

// doRequest sends a JSON POST request with Bearer auth to the Chat Completions API.
func doRequest(ctx context.Context, httpClient *http.Client, baseURL, path, apiKey string, body interface{}) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := strings.TrimRight(baseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bad status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
