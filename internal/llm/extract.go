package llm

import (
	"strings"
)

// ExtractReasoningFromChatCompletions extracts reasoning_content from Chat Completions response.
// Used by DeepSeek thinking mode and other providers that return chain-of-thought.
func ExtractReasoningFromChatCompletions(resp ChatCompletionsResponse) string {
	if len(resp.Choices) == 0 {
		return ""
	}
	return strings.TrimSpace(resp.Choices[0].Message.ReasoningContent)
}

// ExtractTextFromChatCompletions extracts text from Chat Completions response.
func ExtractTextFromChatCompletions(resp ChatCompletionsResponse) string {
	if len(resp.Choices) == 0 {
		return ""
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content)
}

func extractUsageFromChatCompletions(resp ChatCompletionsResponse) *UsageInfo {
	return &UsageInfo{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}
}
