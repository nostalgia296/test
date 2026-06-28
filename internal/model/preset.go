package model

import (
	"github.com/nostalgia296/ocs-ai/internal/config"
	"time"
)

const (
	// BootstrapVersion tracks the builtin presets bootstrap schema version.
	// Set to 0 to disable builtin presets bootstrapping.
	BuiltinPresetBootstrapVersion = 0
)

type ModelConfig struct {
	Name                string  `json:"name"`
	Provider            string  `json:"provider"`
	APIKey              string  `json:"api_key"`
	BaseURL             string  `json:"base_url"`
	ModelName           string  `json:"model_name"`
	IsMultimodal        bool    `json:"is_multimodal"`
	MaxTokens           int     `json:"max_tokens"`
	Temperature         float64 `json:"temperature"`
	TopP                float64 `json:"top_p"`
	SupportsReasoning   bool    `json:"supports_reasoning"`
	ReasoningParamName  string  `json:"reasoning_param_name"`
	ReasoningParamValue string  `json:"reasoning_param_value"`
	APIProtocol         string  `json:"api_protocol"`
	Enabled             bool    `json:"enabled"`
	IsBuiltin           bool    `json:"is_builtin"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

type QuestionTypeConfig struct {
	Models         []string `json:"models"`
	EnableReasoning bool    `json:"enable_reasoning"`
}

type ModelData struct {
	Models              map[string]ModelConfig            `json:"models"`
	Metadata            map[string]string                 `json:"metadata"`
	QuestionTypeModels  map[string]QuestionTypeConfig     `json:"question_type_models"`
	Version             string                            `json:"version"`
	UpdatedAt           string                            `json:"updated_at"`
}

var DefaultQuestionTypeModels = map[string]QuestionTypeConfig{
	config.QuestionTypeSingle:     {Models: []string{}, EnableReasoning: false},
	config.QuestionTypeMultiple:   {Models: []string{}, EnableReasoning: true},
	config.QuestionTypeJudgement:  {Models: []string{}, EnableReasoning: false},
	config.QuestionTypeCompletion: {Models: []string{}, EnableReasoning: false},
	config.QuestionTypeImage:      {Models: []string{}, EnableReasoning: false},
}

func NewModelData() ModelData {
	return ModelData{
		Models:             make(map[string]ModelConfig),
		Metadata:           map[string]string{"builtin_presets_bootstrap_version": "0"},
		QuestionTypeModels: copyQuestionTypeConfigs(DefaultQuestionTypeModels),
		Version:            "1.0",
		UpdatedAt:          time.Now().Format(time.RFC3339),
	}
}

func copyQuestionTypeConfigs(src map[string]QuestionTypeConfig) map[string]QuestionTypeConfig {
	dst := make(map[string]QuestionTypeConfig, len(src))
	for k, v := range src {
		models := make([]string, len(v.Models))
		copy(models, v.Models)
		dst[k] = QuestionTypeConfig{
			Models:         models,
			EnableReasoning: v.EnableReasoning,
		}
	}
	return dst
}
