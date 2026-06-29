package chat

import (
	"strings"

	"github.com/nostalgia296/ocs-ai/internal/llm"
)

// ExtractText extracts the text answer from a Chat Completions response.
func ExtractText(resp ChatCompletionsResponse) string {
	if len(resp.Choices) == 0 {
		return ""
	}
	return strings.TrimSpace(resp.Choices[0].Message.Content)
}

// ExtractReasoning extracts reasoning_content from a Chat Completions response.
// Used by DeepSeek thinking mode and other providers that return chain-of-thought.
func ExtractReasoning(resp ChatCompletionsResponse) string {
	if len(resp.Choices) == 0 {
		return ""
	}
	return strings.TrimSpace(resp.Choices[0].Message.ReasoningContent)
}

// ExtractUsage extracts usage info from a Chat Completions response.
func ExtractUsage(resp ChatCompletionsResponse) *llm.UsageInfo {
	return &llm.UsageInfo{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}
}
