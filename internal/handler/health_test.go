package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nostalgia296/ocs-ai/internal/config"
	"github.com/nostalgia296/ocs-ai/internal/model"
)

func TestHealth_MethodNotAllowed(t *testing.T) {
	cfg := config.DefaultConfig
	mm := model.NewManager("../../custom_models.json")
	h := NewHealthHandler(mm, &cfg)

	for _, method := range []string{"POST", "PUT", "DELETE", "PATCH"} {
		req := httptest.NewRequest(method, "/api/health", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: expected 405, got %d", method, w.Code)
		}
	}
}

func TestHealth_GetReturns200(t *testing.T) {
	cfg := config.DefaultConfig
	mm := model.NewManager("../../custom_models.json")
	h := NewHealthHandler(mm, &cfg)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected JSON Content-Type, got %q", ct)
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}

	// Basic fields that should always be present
	if resp.Service != "OCS AI Answerer" {
		t.Errorf("expected service='OCS AI Answerer', got %q", resp.Service)
	}
	if resp.Version == "" {
		t.Error("version should not be empty")
	}

	t.Logf("health: status=%s service=%s version=%s models=%d/%d ready=%v",
		resp.Status, resp.Service, resp.Version,
		resp.EnabledModelCount, resp.ModelCount, resp.ReadyQuestionTypes)
}

func TestHealth_NoModels(t *testing.T) {
	cfg := config.DefaultConfig
	mm := model.NewManager("/nonexistent/models.json")
	h := NewHealthHandler(mm, &cfg)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	var resp HealthResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.ModelCount != 0 {
		t.Errorf("expected 0 models, got %d", resp.ModelCount)
	}
	if resp.Status == "ok" {
		t.Log("status is ok even with no models (not an error condition)")
	}
}
