package handler

import (
	"encoding/json"
	"net/http"

	"github.com/nostalgia296/ocs-ai/internal/config"
	"github.com/nostalgia296/ocs-ai/internal/model"
)

// HealthResponse is the JSON response from /api/health.
type HealthResponse struct {
	Status             string   `json:"status"`
	Service            string   `json:"service"`
	Version            string   `json:"version"`
	APIConfigured      bool     `json:"api_configured"`
	ModelCount         int      `json:"model_count"`
	EnabledModelCount  int      `json:"enabled_model_count"`
	ReadyQuestionTypes []string `json:"ready_question_types"`
	HasMultimodalModel bool     `json:"has_multimodal_model"`
	InitError          *string  `json:"init_error"`
}

// HealthHandler provides the health check endpoint.
type HealthHandler struct {
	modelManager *model.Manager
	cfg          *config.Config
}

func NewHealthHandler(mm *model.Manager, cfg *config.Config) *HealthHandler {
	return &HealthHandler{modelManager: mm, cfg: cfg}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"status": "error", "error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	runtime := h.modelManager.GetRuntimeSummary()

	status := "ok"
	if !runtime.CanAnswerAny {
		status = "error"
	}

	resp := HealthResponse{
		Status:             status,
		Service:            "OCS AI Answerer",
		Version:            "3.1.0",
		APIConfigured:      runtime.CanAnswerAny,
		ModelCount:         runtime.ModelCount,
		EnabledModelCount:  runtime.EnabledModelCount,
		ReadyQuestionTypes: runtime.ReadyQuestionTypes,
		HasMultimodalModel: runtime.HasMultimodalModel,
		InitError:          runtime.InitError,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
