package model

import (
	"github.com/nostalgia296/ocs-ai/internal/config"
	"time"
)

const (
	BuiltinPresetBootstrapVersion = 1

	PresetDeepSeekV4Flash = "preset_deepseek_v4_flash"
	PresetDeepSeekV4Pro   = "preset_deepseek_v4_pro"
	PresetDoubao          = "preset_doubao"

	BuiltinPresetIDs = PresetDeepSeekV4Flash + "|" + PresetDeepSeekV4Pro + "|" + PresetDoubao
)

var LegacyPresetIDMap = map[string]string{
	"system_deepseek":          PresetDeepSeekV4Flash,
	"system_deepseek_chat":     PresetDeepSeekV4Flash,
	"system_deepseek_reasoner": PresetDeepSeekV4Pro,
	"system_doubao":            PresetDoubao,
}

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

func BuildBuiltinPreset(presetID string, source map[string]ModelConfig, deepSeekKey, deepSeekURL, doubaoKey, doubaoURL string, cfg *config.Config) ModelConfig {
	now := time.Now().Format(time.RFC3339)

	switch presetID {
	case PresetDeepSeekV4Flash:
		apiKey := deepSeekKey
		if source != nil {
			if m, ok := source[PresetDeepSeekV4Flash]; ok {
				apiKey = m.APIKey
			}
			if m, ok := source["system_deepseek_chat"]; ok && apiKey == deepSeekKey {
				apiKey = m.APIKey
			}
			if m, ok := source["system_deepseek"]; ok && apiKey == deepSeekKey {
				apiKey = m.APIKey
			}
		}
		return ModelConfig{
			Name:                "DeepSeek V4 Flash",
			Provider:            "openai",
			APIKey:              apiKey,
			BaseURL:             deepSeekURL,
			ModelName:           "deepseek-v4-flash",
			IsMultimodal:        false,
			MaxTokens:           cfg.MaxTokens,
			Temperature:         cfg.Temperature,
			TopP:                cfg.TopP,
			SupportsReasoning:   true,
			ReasoningParamName:  "reasoning_effort",
			ReasoningParamValue: cfg.ReasoningEffort,
			APIProtocol:         config.ModelAPICompatOpenAI,
			Enabled:             apiKey != "",
			IsBuiltin:           true,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
	case PresetDeepSeekV4Pro:
		apiKey := deepSeekKey
		if source != nil {
			if m, ok := source[PresetDeepSeekV4Pro]; ok {
				apiKey = m.APIKey
			}
			if m, ok := source["system_deepseek_reasoner"]; ok && apiKey == deepSeekKey {
				apiKey = m.APIKey
			}
			if m, ok := source["system_deepseek_chat"]; ok && apiKey == deepSeekKey {
				apiKey = m.APIKey
			}
			if m, ok := source["system_deepseek"]; ok && apiKey == deepSeekKey {
				apiKey = m.APIKey
			}
		}
		return ModelConfig{
			Name:                "DeepSeek V4 Pro",
			Provider:            "openai",
			APIKey:              apiKey,
			BaseURL:             deepSeekURL,
			ModelName:           "deepseek-v4-pro",
			IsMultimodal:        false,
			MaxTokens:           cfg.ReasoningMaxTokens,
			Temperature:         cfg.Temperature,
			TopP:                cfg.TopP,
			SupportsReasoning:   true,
			ReasoningParamName:  "reasoning_effort",
			ReasoningParamValue: cfg.ReasoningEffort,
			APIProtocol:         config.ModelAPICompatOpenAI,
			Enabled:             apiKey != "",
			IsBuiltin:           true,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
	case PresetDoubao:
		apiKey := doubaoKey
		if source != nil {
			if m, ok := source[PresetDoubao]; ok {
				apiKey = m.APIKey
			}
			if m, ok := source["system_doubao"]; ok && apiKey == doubaoKey {
				apiKey = m.APIKey
			}
		}
		return ModelConfig{
			Name:                "Doubao",
			Provider:            "openai",
			APIKey:              apiKey,
			BaseURL:             doubaoURL,
			ModelName:           "doubao-seed-1-6-251015",
			IsMultimodal:        true,
			MaxTokens:           cfg.MaxTokens,
			Temperature:         cfg.Temperature,
			TopP:                cfg.TopP,
			SupportsReasoning:   true,
			ReasoningParamName:  "reasoning_effort",
			ReasoningParamValue: cfg.ReasoningEffort,
			APIProtocol:         config.ModelAPICompatOpenAI,
			Enabled:             apiKey != "",
			IsBuiltin:           true,
			CreatedAt:           now,
			UpdatedAt:           now,
		}
	}

	return ModelConfig{}
}
