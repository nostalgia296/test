package llm

import (
	"context"
	"fmt"
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

// ProviderFunc is the signature for a provider's call implementation.
type ProviderFunc func(ctx context.Context, httpClient *http.Client, model ModelConfigForCall, messages []Message) (*ModelCallResult, error)

var providers = map[string]ProviderFunc{}

// RegisterProvider registers an API protocol implementation.
// Sub-packages (chat, anthropic) call this in init() to register themselves.
func RegisterProvider(protocol string, fn ProviderFunc) {
	providers[protocol] = fn
}

// CallModel calls the LLM with the given parameters, routing to the correct API provider.
func CallModel(ctx context.Context, httpClient *http.Client, model ModelConfigForCall, prompt string, imageURLs []string, imageItems []map[string]string) (*ModelCallResult, error) {
	inferredProvider := InferProvider(model.ModelName, model.BaseURL, model.Provider)
	multimodalURLs := imageURLs
	if !model.IsMultimodal {
		multimodalURLs = nil
	}

	protocol := model.APIProtocol
	if protocol == "" {
		protocol = "chat_completions"
	}

	messages, _, _ := BuildMultimodalMessages(ctx, prompt, inferredProvider, protocol, multimodalURLs, imageItems, true, httpClient)

	providerFn, ok := providers[protocol]
	if !ok {
		return nil, fmt.Errorf("unsupported API protocol: %s", protocol)
	}

	maxAttempts := 3
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err := providerFn(ctx, httpClient, model, messages)
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

// FormatTime formats seconds into a human-readable string.
func FormatTime(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%.1f秒", seconds)
	}
	minutes := int(seconds / 60)
	secs := seconds - float64(minutes)*60
	return fmt.Sprintf("%d分%.1f秒", minutes, secs)
}

// InferProvider determines the provider name from model configuration.
func InferProvider(modelName, baseURL, provider string) string {
	if provider != "" {
		return provider
	}
	if strings.Contains(baseURL, "deepseek") {
		return "deepseek"
	}
	if strings.Contains(baseURL, "anthropic") {
		return "anthropic"
	}
	return "openai_compatible"
}
