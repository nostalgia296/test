package anthropic

import (
	"strings"

	"github.com/nostalgia296/ocs-ai/internal/llm"
)

// ExtractText extracts the text answer from an Anthropic Messages response.
func ExtractText(resp MessagesResponse) string {
	for _, block := range resp.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			return strings.TrimSpace(block.Text)
		}
	}
	return ""
}

// ExtractReasoning extracts the thinking/reasoning content from an Anthropic response.
func ExtractReasoning(resp MessagesResponse) string {
	for _, block := range resp.Content {
		if block.Type == "thinking" && strings.TrimSpace(block.Thinking) != "" {
			return strings.TrimSpace(block.Thinking)
		}
	}
	return ""
}

// ExtractUsage extracts usage info from an Anthropic Messages response.
func ExtractUsage(resp MessagesResponse) *llm.UsageInfo {
	return &llm.UsageInfo{
		PromptTokens:     resp.Usage.InputTokens,
		CompletionTokens: resp.Usage.OutputTokens,
		TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
	}
}
