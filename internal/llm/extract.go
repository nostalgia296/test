package llm

import (
	"encoding/json"
	"strings"
)

// ExtractReasoningFromResponsesAPI extracts reasoning text from Responses API response.
func ExtractReasoningFromResponsesAPI(resp ResponsesResponse) string {
	var reasoningParts []string

	for _, item := range resp.Output {
		if item.Type != "reasoning" {
			continue
		}
		for _, summaryItem := range item.Summary {
			text := strings.TrimSpace(summaryItem.Text)
			if text != "" && (summaryItem.Type == "summary_text" || summaryItem.Type == "text" || summaryItem.Type == "output_text") {
				reasoningParts = append(reasoningParts, text)
			}
		}
	}

	if len(reasoningParts) > 0 {
		return strings.Join(reasoningParts, "\n")
	}
	return ""
}

// ExtractTextFromResponsesAPI extracts text output from Responses API response.
func ExtractTextFromResponsesAPI(resp ResponsesResponse) string {
	var parts []string

	for _, item := range resp.Output {
		if item.Type == "output_text" || item.Type == "text" {
			for _, contentItem := range item.Content {
				text := strings.TrimSpace(contentItem.Text)
				if text != "" {
					parts = append(parts, text)
				}
			}
		}
	}

	return strings.Join(parts, "\n")
}

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

// ExtractUsageFromRaw extracts usage info from raw JSON response bytes.
func ExtractUsageFromRaw(data []byte) *UsageInfo {
	var raw struct {
		Usage map[string]int `json:"usage"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}
	if raw.Usage == nil {
		return nil
	}
	return &UsageInfo{
		PromptTokens:     raw.Usage["prompt_tokens"],
		CompletionTokens: raw.Usage["completion_tokens"],
		TotalTokens:      raw.Usage["total_tokens"],
	}
}

func extractUsageFromChatCompletions(resp ChatCompletionsResponse) *UsageInfo {
	return &UsageInfo{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}
}
