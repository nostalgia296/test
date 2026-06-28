package model

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/nostalgia296/ocs-ai/internal/config"
)

type Manager struct {
	ConfigFile         string
	Models             map[string]ModelConfig
	Metadata           map[string]string
	QuestionTypeModels map[string]QuestionTypeConfig
}

func NewManager(configFile string) *Manager {
	m := &Manager{
		ConfigFile:         configFile,
		Models:             make(map[string]ModelConfig),
		Metadata:           map[string]string{"builtin_presets_bootstrap_version": "0"},
		QuestionTypeModels: copyQuestionTypeConfigs(DefaultQuestionTypeModels),
	}
	if err := m.Load(); err != nil {
		log.Printf("Warning: failed to load model config: %v", err)
	}
	return m
}

func (m *Manager) Load() error {
	data, err := os.ReadFile(m.ConfigFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	var md ModelData
	if err := json.Unmarshal(data, &md); err != nil {
		return err
	}

	m.Models = md.Models
	if md.Metadata != nil {
		m.Metadata = md.Metadata
	}
	if len(m.Metadata) == 0 {
		m.Metadata = map[string]string{"builtin_presets_bootstrap_version": "0"}
	}
	m.QuestionTypeModels = md.QuestionTypeModels

	changed := m.normalizeState()
	if changed {
		m.Save()
	}
	return nil
}

func (m *Manager) Save() error {
	md := ModelData{
		Models:             m.Models,
		Metadata:           m.Metadata,
		QuestionTypeModels: m.QuestionTypeModels,
		Version:            "1.0",
		UpdatedAt:          time.Now().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(md, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.ConfigFile, data, 0644)
}

func (m *Manager) GetModel(id string) *ModelConfig {
	if cfg, ok := m.Models[id]; ok {
		return &cfg
	}
	return nil
}

func (m *Manager) GetAllModels(enabledOnly bool) []ModelConfig {
	var result []ModelConfig
	for _, cfg := range m.Models {
		if !enabledOnly || cfg.Enabled {
			result = append(result, cfg)
		}
	}
	return result
}

func (m *Manager) GetModelIDs() []string {
	ids := make([]string, 0, len(m.Models))
	for id := range m.Models {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (m *Manager) AddModel(id string, cfg ModelConfig) error {
	if _, exists := m.Models[id]; exists {
		return fmt.Errorf("model %s already exists", id)
	}
	now := time.Now().Format(time.RFC3339)
	cfg.CreatedAt = now
	cfg.UpdatedAt = now
	if !cfg.IsBuiltin {
		cfg.IsBuiltin = false
	}
	m.Models[id] = cfg
	return m.Save()
}

func (m *Manager) UpdateModel(id string, updates ModelConfig) error {
	existing, ok := m.Models[id]
	if !ok {
		return fmt.Errorf("model %s not found", id)
	}
	updates.CreatedAt = existing.CreatedAt
	updates.UpdatedAt = time.Now().Format(time.RFC3339)
	m.Models[id] = updates
	return m.Save()
}

func (m *Manager) DeleteModel(id string) error {
	if _, ok := m.Models[id]; !ok {
		return fmt.Errorf("model %s not found", id)
	}
	delete(m.Models, id)
	m.removeFromMappings(id)
	return m.Save()
}

func (m *Manager) SetQuestionTypeModels(questionType string, modelIDs []string, enableReasoning *bool) error {
	modelsCopy := make([]string, len(modelIDs))
	copy(modelsCopy, modelIDs)
	cfg := m.QuestionTypeModels[questionType]
	cfg.Models = modelsCopy
	if enableReasoning != nil {
		cfg.EnableReasoning = *enableReasoning
	}
	m.QuestionTypeModels[questionType] = cfg
	return m.Save()
}

func (m *Manager) GetQuestionTypeModels(questionType string) []string {
	if cfg, ok := m.QuestionTypeModels[questionType]; ok {
		result := make([]string, len(cfg.Models))
		copy(result, cfg.Models)
		return result
	}
	return nil
}

func (m *Manager) GetQuestionTypeReasoning(questionType string) bool {
	if cfg, ok := m.QuestionTypeModels[questionType]; ok {
		return cfg.EnableReasoning
	}
	return false
}

func (m *Manager) GetAvailableModels(questionType string, hasImages bool) []string {
	candidates := []string{}
	seen := map[string]bool{}

	appendModels := func(ids []string, requireMultimodal bool) {
		for _, id := range ids {
			if seen[id] {
				continue
			}
			model := m.GetModel(id)
			if model == nil || !model.Enabled {
				continue
			}
			if requireMultimodal && !model.IsMultimodal {
				continue
			}
			candidates = append(candidates, id)
			seen[id] = true
		}
	}

	if hasImages {
		appendModels(m.GetQuestionTypeModels(config.QuestionTypeImage), true)
	}
	appendModels(m.GetQuestionTypeModels(questionType), hasImages)
	return candidates
}

func (m *Manager) HasMultimodalModel() bool {
	for _, model := range m.Models {
		if model.Enabled && model.IsMultimodal {
			return true
		}
	}
	return false
}

func (m *Manager) GetRuntimeSummary() RuntimeSummary {
	enabledCount := 0
	for _, model := range m.Models {
		if model.Enabled {
			enabledCount++
		}
	}

	mappedTypes := map[string][]string{}
	var readyTypes []string

	for _, qt := range []string{
		config.QuestionTypeSingle, config.QuestionTypeMultiple,
		config.QuestionTypeJudgement, config.QuestionTypeCompletion, config.QuestionTypeImage,
	} {
		hasImages := qt == config.QuestionTypeImage
		ids := m.GetAvailableModels(qt, hasImages)
		mappedTypes[qt] = ids
		if len(ids) > 0 {
			readyTypes = append(readyTypes, qt)
		}
	}

	canAnswerAny := len(readyTypes) > 0
	var initError *string

	switch {
	case len(m.Models) == 0:
		e := "未配置任何模型，请到模型管理页添加或启用模型"
		initError = &e
	case enabledCount == 0:
		e := "所有模型均已禁用，请到模型管理页启用至少一个模型"
		initError = &e
	case !canAnswerAny:
		e := "未为任何题型配置可用模型，请到模型管理页设置题型映射"
		initError = &e
	}

	return RuntimeSummary{
		ModelCount:          len(m.Models),
		EnabledModelCount:   enabledCount,
		MappedQuestionTypes: mappedTypes,
		ReadyQuestionTypes:  readyTypes,
		HasMultimodalModel:  m.HasMultimodalModel(),
		CanAnswerAny:        canAnswerAny,
		InitError:           initError,
	}
}

type RuntimeSummary struct {
	ModelCount          int               `json:"model_count"`
	EnabledModelCount   int               `json:"enabled_model_count"`
	MappedQuestionTypes map[string][]string `json:"mapped_question_types"`
	ReadyQuestionTypes  []string           `json:"ready_question_types"`
	HasMultimodalModel  bool               `json:"has_multimodal_model"`
	CanAnswerAny        bool               `json:"can_answer_any"`
	InitError           *string            `json:"init_error"`
}

func (m *Manager) BootstrapBuiltinPresets(cfg *config.Config) error {
	// No builtin presets - users configure their own models
	m.Metadata["builtin_presets_bootstrap_version"] = "0"
	return nil
}

func (m *Manager) normalizeState() bool {
	changed := false

	normalizedModels := make(map[string]ModelConfig, len(m.Models))
	for id, cfg := range m.Models {
		if cfg.MaxTokens == 0 {
			cfg.MaxTokens = 2000
		}
		if cfg.Temperature == 0 {
			cfg.Temperature = 0.1
		}
		if cfg.TopP == 0 {
			cfg.TopP = 0.95
		}
		if cfg.SupportsReasoning && cfg.ReasoningParamName == "" {
			cfg.ReasoningParamName = "reasoning_effort"
		}
		if cfg.SupportsReasoning && cfg.ReasoningParamValue == "" {
			cfg.ReasoningParamValue = "medium"
		}
		if cfg.APIProtocol == "" {
			cfg.APIProtocol = config.ModelAPICompatOpenAI
		}
		normalizedModels[id] = cfg
	}
	if len(normalizedModels) != len(m.Models) {
		changed = true
	}
	m.Models = normalizedModels

	normalizedMappings := normalizeQuestionTypeMappings(m.QuestionTypeModels)
	if len(normalizedMappings) != len(m.QuestionTypeModels) {
		changed = true
	}
	m.QuestionTypeModels = normalizedMappings

	if m.sanitizeMappings() {
		changed = true
	}

	return changed
}

func normalizeQuestionTypeMappings(src map[string]QuestionTypeConfig) map[string]QuestionTypeConfig {
	result := copyQuestionTypeConfigs(DefaultQuestionTypeModels)
	for qt := range result {
		cfg, ok := src[qt]
		if !ok {
			continue
		}
		if len(cfg.Models) > 0 || cfg.EnableReasoning {
			models := make([]string, len(cfg.Models))
			copy(models, cfg.Models)
			result[qt] = QuestionTypeConfig{
				Models:         models,
				EnableReasoning: cfg.EnableReasoning,
			}
		}
	}
	return result
}

func (m *Manager) sanitizeMappings() bool {
	changed := false
	for qt, cfg := range m.QuestionTypeModels {
		filtered := make([]string, 0, len(cfg.Models))
		for _, modelID := range cfg.Models {
			if modelID == "" {
				continue
			}
			if qt == config.QuestionTypeImage {
				model := m.GetModel(modelID)
				if model == nil || !model.IsMultimodal {
					continue
				}
			}
			filtered = append(filtered, modelID)
		}
		if len(filtered) != len(cfg.Models) {
			qtCfg := m.QuestionTypeModels[qt]
			qtCfg.Models = filtered
			m.QuestionTypeModels[qt] = qtCfg
			changed = true
		}
	}
	return changed
}

func (m *Manager) removeFromMappings(modelID string) bool {
	changed := false
	for qt, cfg := range m.QuestionTypeModels {
		filtered := make([]string, 0, len(cfg.Models))
		for _, id := range cfg.Models {
			if id != modelID {
				filtered = append(filtered, id)
			} else {
				changed = true
			}
		}
		qtCfg := m.QuestionTypeModels[qt]
		qtCfg.Models = filtered
		m.QuestionTypeModels[qt] = qtCfg
	}
	return changed
}
