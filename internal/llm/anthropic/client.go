package anthropic

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
	llm.RegisterProvider("anthropic", Call)
}

// Call implements the Anthropic Messages API.
func Call(ctx context.Context, httpClient *http.Client, model llm.ModelConfigForCall, messages []llm.Message) (*llm.ModelCallResult, error) {
	// Build Anthropic-formatted messages from internal format
	anthropicMessages, systemContent := convertMessages(messages)

	reqBody := MessagesRequest{
		Model:     model.ModelName,
		Messages:  anthropicMessages,
		System:    systemContent,
		MaxTokens: model.MaxTokens,
		Stream:    false,
	}

	// Enable extended thinking if DSThinkingMode is on
	if model.DSThinkingMode {
		// Anthropic recommends budget_tokens >= 1024 and remaining max_tokens for the actual response
		thinkingBudget := model.MaxTokens / 3
		if thinkingBudget < 1024 {
			thinkingBudget = 1024
		}
		reqBody.Thinking = &ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: thinkingBudget,
		}
		// MaxTokens must be larger than thinking budget_tokens
		// Ensure at least 256 tokens for the response beyond thinking
		if reqBody.MaxTokens <= thinkingBudget+256 {
			reqBody.MaxTokens = thinkingBudget + 1024
		}
	}

	respBody, err := doRequest(ctx, httpClient, model.BaseURL, "/v1/messages", model.APIKey, reqBody)
	if err != nil {
		return nil, err
	}

	var msgResp MessagesResponse
	if err := json.Unmarshal(respBody, &msgResp); err != nil {
		return nil, err
	}

	return &llm.ModelCallResult{
		Answer:    ExtractText(msgResp),
		Reasoning: ExtractReasoning(msgResp),
		Usage:     ExtractUsage(msgResp),
	}, nil
}

// convertMessages converts internal llm.Message format to Anthropic message format.
// It extracts system messages (top-level in Anthropic) and builds the messages list.
func convertMessages(messages []llm.Message) ([]AnthropicMessage, string) {
	var systemContent string
	var anthropicMessages []AnthropicMessage

	for _, msg := range messages {
		if msg.Role == "system" {
			// Anthropic uses system as a top-level field; extract the content
			switch c := msg.Content.(type) {
			case string:
				systemContent = c
			default:
				// For non-string system content, JSON-marshal it as a string
				data, _ := json.Marshal(c)
				systemContent = string(data)
			}
			continue
		}
		anthropicMessages = append(anthropicMessages, AnthropicMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return anthropicMessages, systemContent
}

// doRequest sends a JSON POST request with x-api-key auth to the Anthropic API.
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
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

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
