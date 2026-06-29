package anthropic

// MessagesRequest represents the Anthropic Messages API request body.
type MessagesRequest struct {
	Model     string           `json:"model"`
	Messages  []AnthropicMessage `json:"messages"`
	System    string           `json:"system,omitempty"`
	MaxTokens int              `json:"max_tokens"`
	Stream    bool             `json:"stream,omitempty"`
	Thinking  *ThinkingConfig  `json:"thinking,omitempty"`
}

// ThinkingConfig configures Anthropic extended thinking.
type ThinkingConfig struct {
	Type         string `json:"type"`         // "enabled"
	BudgetTokens int    `json:"budget_tokens"` // max tokens for thinking
}

// AnthropicMessage is a message in Anthropic format.
type AnthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"` // string or []ContentBlock
}

// ContentBlock is a single content block (text, image, thinking, etc.).
type ContentBlock struct {
	Type      string       `json:"type"`
	Text      string       `json:"text,omitempty"`
	Thinking  string       `json:"thinking,omitempty"`
	Signature string       `json:"signature,omitempty"`
	Source    *ImageSource `json:"source,omitempty"`
}

// ImageSource describes an image in Anthropic format.
type ImageSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // e.g. "image/jpeg"
	Data      string `json:"data"`       // raw base64
}

// MessagesResponse represents the Anthropic Messages API response.
type MessagesResponse struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Role       string         `json:"role"`
	Content    []ContentBlock `json:"content"`
	Model      string         `json:"model"`
	StopReason string         `json:"stop_reason"`
	Usage      AnthropicUsage `json:"usage"`
}

// AnthropicUsage represents usage from Anthropic Messages response.
type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ErrorResponse represents an Anthropic API error.
type ErrorResponse struct {
	Type  string     `json:"type"`
	Error ErrorDetail `json:"error"`
}

// ErrorDetail holds Anthropic error details.
type ErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
