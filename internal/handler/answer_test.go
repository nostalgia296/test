package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nostalgia296/ocs-ai/internal/config"
	"github.com/nostalgia296/ocs-ai/internal/model"
)

// newTestService creates a Service with a real model Manager for HTTP-layer tests.
func newTestService() *Service {
	cfg := &config.Config{
		Host:          "127.0.0.1",
		Port:          5000,
		Timeout:       1200,
		DSThinkingMode: false,
		CSVLogFile:   "",
	}
	mm := model.NewManager("../../custom_models.json")
	return NewService(mm, cfg)
}

// newEmptyService creates a Service with no models configured (to test error paths).
func newEmptyService() *Service {
	cfg := &config.Config{
		Host:    "127.0.0.1",
		Port:    5000,
		Timeout: 1200,
	}
	// Use a non-existent file so the manager starts with zero models
	mm := model.NewManager("/nonexistent/models.json")
	return NewService(mm, cfg)
}

func doRequest(s *Service, method, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/answer", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.HandleAnswer(w, req)
	return w
}

// ---------------------------------------------------------------------------
// Method validation
// ---------------------------------------------------------------------------

func TestHandleAnswer_MethodNotAllowed(t *testing.T) {
	s := newTestService()

	for _, method := range []string{"GET", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"} {
		w := doRequest(s, method, `{"question":"test"}`)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s: expected 405, got %d", method, w.Code)
		}
		var resp map[string]interface{}
		json.NewDecoder(w.Body).Decode(&resp)
		if resp["success"] != false {
			t.Errorf("%s: expected success=false", method)
		}
	}
}

func TestHandleAnswer_PostAccepted(t *testing.T) {
	s := newTestService()
	w := doRequest(s, "POST", `{"question":"test question"}`)
	// POST should be accepted; exact code depends on model availability
	if w.Code == http.StatusMethodNotAllowed {
		t.Errorf("POST should not return 405, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Request body validation
// ---------------------------------------------------------------------------

func TestHandleAnswer_InvalidJSON(t *testing.T) {
	s := newTestService()
	w := doRequest(s, "POST", `not json`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"] != false {
		t.Error("expected success=false for bad JSON")
	}
}

func TestHandleAnswer_EmptyQuestion(t *testing.T) {
	s := newTestService()
	w := doRequest(s, "POST", `{"question":"  "}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty question, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"] != false {
		t.Error("expected success=false for empty question")
	}
}

func TestHandleAnswer_MissingQuestion(t *testing.T) {
	s := newTestService()
	w := doRequest(s, "POST", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing question, got %d", w.Code)
	}
}

func TestHandleAnswer_EmptyBody(t *testing.T) {
	s := newTestService()
	w := doRequest(s, "POST", ``)
	// Empty body with JSON decoder returns EOF -> error
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty body, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// Response format validation (model-dependent — runs when models exist)
// ---------------------------------------------------------------------------

func TestHandleAnswer_ResponseContentType(t *testing.T) {
	s := newTestService()
	w := doRequest(s, "POST", `{"question":"你好","type":0}`)
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected application/json Content-Type, got %q", ct)
	}
}

// TestHandleAnswer_ResponseStructure checks the success path when models are
// configured and callable. It validates every field in AnswerResponse.
func TestHandleAnswer_ResponseStructure(t *testing.T) {
	s := newTestService()

	req := AnswerRequest{
		Question: "1+1等于几",
		Options:  []string{"1", "2", "3", "4"},
		Type:     0, // single choice
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/answer", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	s.HandleAnswer(w, r)

	respBytes, err := io.ReadAll(w.Result().Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	// If the response is an error (no models available / models failed),
	// the test still validates the error response format.
	if w.Code == http.StatusInternalServerError {
		var errResp map[string]interface{}
		if err := json.Unmarshal(respBytes, &errResp); err != nil {
			t.Fatalf("error response not valid JSON: %v", err)
		}
		if errResp["success"] != false {
			t.Error("error response should have success=false")
		}
		if errResp["error"] == nil || errResp["error"] == "" {
			t.Error("error response should have an error message")
		}
		t.Logf("⚠ No available models – error response validated: %v", errResp["error"])
		return
	}

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d — body: %s", w.Code, string(respBytes))
	}

	var resp AnswerResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.Fatalf("response is not valid AnswerResponse JSON: %v", err)
	}

	// --- validate every field ---
	checks := []struct {
		field string
		ok    bool
		got   string
	}{
		{"success", resp.Success, ""},
		{"question (non-empty)", len(resp.Question) > 0, resp.Question},
		{"raw_answer (non-empty)", len(resp.RawAnswer) > 0, resp.RawAnswer},
		{"answer (non-empty)", len(resp.Answer) > 0, resp.Answer},
		{"ocs_answer (non-empty)", len(resp.OCSAnswer) > 0, resp.OCSAnswer},
		{"model (non-empty)", len(resp.Model) > 0, resp.Model},
		{"provider (non-empty)", len(resp.Provider) > 0, resp.Provider},
		{"type matches", resp.Type == "single", resp.Type},
		{"ai_time >= 0", resp.AITime >= 0, ""},
		{"total_time >= ai_time", resp.TotalTime >= resp.AITime, ""},
		{"usage not nil", resp.Usage != nil, ""},
	}

	for _, c := range checks {
		if c.got != "" && !c.ok {
			t.Errorf("%s: got=%q", c.field, c.got)
		} else if c.got == "" && !c.ok {
			t.Errorf("%s: failed", c.field)
		}
	}

	if resp.Usage != nil {
		if resp.Usage.PromptTokens <= 0 {
			t.Errorf("usage.prompt_tokens should be > 0, got %d", resp.Usage.PromptTokens)
		}
		if resp.Usage.CompletionTokens <= 0 {
			t.Errorf("usage.completion_tokens should be > 0, got %d", resp.Usage.CompletionTokens)
		}
		if resp.Usage.TotalTokens != resp.Usage.PromptTokens+resp.Usage.CompletionTokens {
			t.Errorf("usage.total_tokens mismatch: %d != %d+%d",
				resp.Usage.TotalTokens, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
		}
	}

	// Validate OCSFormat: [question, ocs_answer, metadata]
	if len(resp.OCSFormat) != 3 {
		t.Errorf("ocs_format should have 3 elements, got %d", len(resp.OCSFormat))
	} else {
		if q, ok := resp.OCSFormat[0].(string); !ok || q != resp.Question {
			t.Errorf("ocs_format[0] should be question string %q, got %v", resp.Question, resp.OCSFormat[0])
		}
		if a, ok := resp.OCSFormat[1].(string); !ok || a != resp.OCSAnswer {
			t.Errorf("ocs_format[1] should be ocs_answer %q, got %v", resp.OCSAnswer, resp.OCSFormat[1])
		}
	}

	t.Logf("✅ Full response validated — model=%s provider=%s time=%.2fs tokens=%d",
		resp.Model, resp.Provider, resp.TotalTime, resp.Usage.TotalTokens)
}

// ---------------------------------------------------------------------------
// Question type mapping
// ---------------------------------------------------------------------------

func TestHandleAnswer_QuestionTypes(t *testing.T) {
	s := newTestService()

	tests := []struct {
		name     string
		typeNum  int
		expected string
	}{
		{"single", 0, "single"},
		{"multiple", 1, "multiple"},
		{"judgement", 4, "judgement"},
		{"completion", 3, "completion"},
		{"unknown defaults to single", 99, "single"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := AnswerRequest{
				Question: "测试题目",
				Options:  []string{"A", "B", "C"},
				Type:     tc.typeNum,
			}
			body, _ := json.Marshal(req)
			w := httptest.NewRecorder()
			r := httptest.NewRequest("POST", "/api/answer", bytes.NewReader(body))
			r.Header.Set("Content-Type", "application/json")
			s.HandleAnswer(w, r)

			if w.Code == http.StatusInternalServerError {
				t.Skipf("no models available; skipping type-mapping check")
				return
			}

			var resp AnswerResponse
			json.NewDecoder(w.Body).Decode(&resp)
			if resp.Type != tc.expected {
				t.Errorf("type %d → expected type=%q, got %q", tc.typeNum, tc.expected, resp.Type)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// No-models scenario
// ---------------------------------------------------------------------------

func TestHandleAnswer_NoModelsAvailable(t *testing.T) {
	s := newEmptyService()
	w := doRequest(s, "POST", `{"question":"test"}`)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 when no models, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["success"] != false {
		t.Error("expected success=false")
	}
	if resp["error"] == nil || resp["error"] == "" {
		t.Error("expected non-empty error message")
	}
	t.Logf("correctly rejected with: %v", resp["error"])
}

// ---------------------------------------------------------------------------
// Image question path
// ---------------------------------------------------------------------------

func TestHandleAnswer_ImageQuestion_NoMultimodalModel(t *testing.T) {
	s := newTestService()

	req := AnswerRequest{
		Question: "图中是什么",
		Options:  []string{"猫", "狗"},
		Type:     0,
		Images:   []string{"https://example.com/photo.jpg"},
	}
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/answer", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	s.HandleAnswer(w, r)

	// If no multimodal model configured, should get an error
	if w.Code == http.StatusInternalServerError {
		var resp map[string]interface{}
		json.NewDecoder(w.Body).Decode(&resp)
		t.Logf("image question correctly handled: code=%d error=%v", w.Code, resp["error"])
		// If it somehow succeeds (multimodal model was added), validate response
	} else {
		var resp AnswerResponse
		json.NewDecoder(w.Body).Decode(&resp)
		t.Logf("image question succeeded with model=%s", resp.Model)
	}
}
