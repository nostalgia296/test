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
	Reasoning     string
	Answer        string
	Usage         *UsageInfo
	ReasoningUsed bool
}

// ModelConfigForCall is the model configuration needed for making LLM calls.
type ModelConfigForCall struct {
	ID                string
	Name              string
	Provider          string
	APIKey            string
	BaseURL           string
	ModelName         string
	IsMultimodal      bool
	MaxTokens         int
	Temperature       float64
	TopP              float64
	SupportsReasoning bool
	ReasoningParamName  string
	ReasoningParamValue string
	APIProtocol       string
}

// UsageInfo holds token usage information.
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CallModel calls the LLM with the given parameters.
func CallModel(ctx context.Context, httpClient *http.Client, model ModelConfigForCall, prompt string, imageURLs []string, imageItems []map[string]string, forceReasoning bool) (*ModelCallResult, error) {
	inferredProvider := InferProvider(model.ModelName, model.BaseURL, model.Provider)
	multimodalURLs := imageURLs
	if !model.IsMultimodal {
		multimodalURLs = nil
	}

	reasoningPayload, legacyReasoning := BuildReasoningPayload(model, forceReasoning)
	reasoningRequested := reasoningPayload != nil || legacyReasoning != nil
	useResponsesAPI := ShouldUseResponsesAPI(model)

	messages, _, _ := BuildMultimodalMessages(ctx, prompt, inferredProvider, multimodalURLs, imageItems, true, httpClient)

	maxAttempts := 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err := callModelOnce(ctx, httpClient, model, messages, reasoningPayload, legacyReasoning, useResponsesAPI)
		if err == nil {
			result.ReasoningUsed = reasoningRequested
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

func callModelOnce(ctx context.Context, httpClient *http.Client, model ModelConfigForCall, messages []Message, reasoningPayload map[string]interface{}, legacyReasoning *ReasoningParam, useResponsesAPI bool) (*ModelCallResult, error) {
	if useResponsesAPI {
		input, _, _ := BuildResponsesInput(messages)
		reqBody := map[string]interface{}{
			"model":            model.ModelName,
			"input":            input,
			"max_output_tokens": model.MaxTokens,
		}
		if model.Temperature > 0 {
			reqBody["temperature"] = model.Temperature
		}
		if model.TopP > 0 {
			reqBody["top_p"] = model.TopP
		}
		if reasoningPayload != nil {
			reqBody["reasoning"] = reasoningPayload
		} else if legacyReasoning != nil {
			reqBody[legacyReasoning.Name] = legacyReasoning.Value
		}

		respBody, err := doOpenAIRequest(ctx, httpClient, model.BaseURL, "/v1/responses", model.APIKey, reqBody)
		if err != nil {
			return nil, err
		}

		var responsesResp ResponsesResponse
		if err := json.Unmarshal(respBody, &responsesResp); err != nil {
			return nil, err
		}

		return &ModelCallResult{
			Reasoning: ExtractReasoningFromResponsesAPI(responsesResp),
			Answer:    ExtractTextFromResponsesAPI(responsesResp),
			Usage:     ExtractUsageFromRaw(respBody),
		}, nil
	}

	// Chat Completions API
	chatMessages := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		chatMessages[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
	}

	reqBody := map[string]interface{}{
		"model":         model.ModelName,
		"messages":      chatMessages,
		"temperature":   model.Temperature,
		"max_tokens":    model.MaxTokens,
		"top_p":         model.TopP,
		"stream":        false,
	}
	if legacyReasoning != nil {
		reqBody[legacyReasoning.Name] = legacyReasoning.Value
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
		Reasoning: ExtractReasoningFromChatCompletions(chatResp),
		Answer:    ExtractTextFromChatCompletions(chatResp),
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
