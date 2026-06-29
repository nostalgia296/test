package llm

import (
	"strings"
)

// ExtractReasoningFromChatCompletions extracts reasoning from Chat Completions response.
func ExtractReasoningFromChatCompletions(resp ChatCompletionsResponse) string {
	if len(resp.Choices) == 0 {
		return ""
	}

	message := resp.Choices[0].Message
	reasoningContent := strings.TrimSpace(message.ReasoningContent)
	if reasoningContent != "" {
		return reasoningContent
	}

	return ""
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
