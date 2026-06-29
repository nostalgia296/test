package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ModelCallResult holds the result of calling an LLM model.
type ModelCallResult struct {
	Answer    string
	Reasoning string
	Usage     *UsageInfo
}

// ModelConfigForCall is the model configuration needed for making LLM calls.
type ModelConfigForCall struct {
	ID             string
	Name           string
	Provider       string
	APIKey         string
	BaseURL        string
	ModelName      string
	IsMultimodal   bool
	MaxTokens      int
	Temperature    float64
	TopP           float64
	APIProtocol    string
	DSThinkingMode bool
}

// UsageInfo holds token usage information.
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CallModel calls the LLM with the given parameters.
func CallModel(ctx context.Context, httpClient *http.Client, model ModelConfigForCall, prompt string, imageURLs []string, imageItems []map[string]string) (*ModelCallResult, error) {
	inferredProvider := InferProvider(model.ModelName, model.BaseURL, model.Provider)
	multimodalURLs := imageURLs
	if !model.IsMultimodal {
		multimodalURLs = nil
	}

	messages, _, _ := BuildMultimodalMessages(ctx, prompt, inferredProvider, multimodalURLs, imageItems, true, httpClient)

	maxAttempts := 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err := callModelOnce(ctx, httpClient, model, messages)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if attempt < maxAttempts {
			select {
			case <-time.After(1 * time.Second):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("model call failed after %d attempts: %w", maxAttempts, lastErr)
}

func callModelOnce(ctx context.Context, httpClient *http.Client, model ModelConfigForCall, messages []Message) (*ModelCallResult, error) {
	// Chat Completions API
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

	respBody, err := doOpenAIRequest(ctx, httpClient, model.BaseURL, "/v1/chat/completions", model.APIKey, reqBody)
	if err != nil {
		return nil, err
	}

	var chatResp ChatCompletionsResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, err
	}

	return &ModelCallResult{
		Answer:    ExtractTextFromChatCompletions(chatResp),
		Reasoning: ExtractReasoningFromChatCompletions(chatResp),
		Usage:     extractUsageFromChatCompletions(chatResp),
	}, nil
}

func doOpenAIRequest(ctx context.Context, httpClient *http.Client, baseURL, path, apiKey string, body interface{}) ([]byte, error) {
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

// FormatTime formats seconds into a human-readable string.
func FormatTime(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%.1f秒", seconds)
	}
	minutes := int(seconds / 60)
	secs := seconds - float64(minutes)*60
	return fmt.Sprintf("%d分%.1f秒", minutes, secs)
}
