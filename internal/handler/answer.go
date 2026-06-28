package handler

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/nostalgia296/ocs-ai/internal/answer"
	"github.com/nostalgia296/ocs-ai/internal/config"
	"github.com/nostalgia296/ocs-ai/internal/llm"
	"github.com/nostalgia296/ocs-ai/internal/log"
	"github.com/nostalgia296/ocs-ai/internal/model"
	"github.com/nostalgia296/ocs-ai/internal/prompt"
)

var (
	iconKeywords = map[string]bool{
		"/icon/": true, "/icons/": true, "icon/": true,
		"video.png": true, "audio.png": true, "play.png": true, "pause.png": true,
	}
	imgPattern = regexp.MustCompile(`(https?://[a-zA-Z0-9\-._~:/?#\[\]@!$&'()*+,;=%]+?\.(?:jpg|jpeg|png|gif|bmp|webp))`)
)

// AnswerRequest is the JSON request body for /api/answer.
type AnswerRequest struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
	Type     int      `json:"type"`
	Images   []string `json:"images"`
}

// AnswerResponse is the JSON response from /api/answer.
type AnswerResponse struct {
	Success       bool                   `json:"success"`
	Question      string                 `json:"question"`
	Answer        string                 `json:"answer"`
	OCSAnswer     string                 `json:"ocs_answer"`
	Type          string                 `json:"type"`
	RawAnswer     string                 `json:"raw_answer"`
	Model         string                 `json:"model"`
	Provider      string                 `json:"provider"`
	ReasoningUsed bool                   `json:"reasoning_used"`
	AITime        float64                `json:"ai_time"`
	TotalTime     float64                `json:"total_time"`
	Usage         *UsageInfo             `json:"usage"`
	OCSFormat     []interface{}          `json:"ocs_format"`
}

// UsageInfo holds token usage data.
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Service provides the core answering business logic.
type Service struct {
	modelManager *model.Manager
	cfg          *config.Config
	httpClient   *http.Client
}

func NewService(mm *model.Manager, cfg *config.Config) *Service {
	return &Service{
		modelManager: mm,
		cfg:          cfg,
		httpClient:   &http.Client{Timeout: time.Duration(cfg.Timeout * float64(time.Second))},
	}
}

func (s *Service) HandleAnswer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"success": false, "error": "method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	var req AnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "无效的请求数据"})
		return
	}

	question := strings.TrimSpace(req.Question)
	if question == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "题目不能为空"})
		return
	}

	options := normalizeOptions(req.Options)
	qType := config.QuestionTypeSingle
	qTypeName := "单选题"
	if key, ok := config.TypeNumToKey[req.Type]; ok {
		qType = key
		qTypeName = config.QuestionTypeNames[qType]
	}

	if qType == config.QuestionTypeCompletion {
		options = []string{}
	}

	// Extract and filter images
	imageItems, imageURLs := extractImages(question, options, req.Images)
	imageItems = filterIconImages(imageItems)
	imageURLs = make([]string, len(imageItems))
	for i, item := range imageItems {
		imageURLs[i] = item["url"]
	}

	useOptionLabels := (qType == config.QuestionTypeSingle || qType == config.QuestionTypeMultiple) &&
		hasOptionImages(imageItems)

	// Build prompt
	promptText := prompt.Build(question, options, qType, useOptionLabels)

	// Determine reasoning mode
	forceReasoning := determineReasoning(s.cfg, qType, len(imageURLs) > 0)

	// Check available models
	typeModels := s.modelManager.GetAvailableModels(qType, len(imageURLs) > 0)
	if len(typeModels) == 0 {
		errorMsg := qTypeName + "未配置可用模型，请到模型管理页设置题型映射"
		if len(imageURLs) > 0 {
			errorMsg = "图片题未配置可用的多模态模型，请到模型管理页为图片题配置模型"
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": errorMsg})
		return
	}

	// Call models with failover
	startTime := time.Now()
	var reasoning, rawAnswer string
	var usage *llm.UsageInfo
	var actualModelID, modelName, actualProvider string
	var reasoningUsed bool

	for _, tryModelID := range typeModels {
		m := s.modelManager.GetModel(tryModelID)
		if m == nil {
			continue
		}

		llmModel := llm.ModelConfigForCall{
			ID:                tryModelID,
			Name:              m.Name,
			Provider:          m.Provider,
			APIKey:            m.APIKey,
			BaseURL:           m.BaseURL,
			ModelName:         m.ModelName,
			IsMultimodal:      m.IsMultimodal,
			MaxTokens:         m.MaxTokens,
			Temperature:       m.Temperature,
			TopP:              m.TopP,
			SupportsReasoning: m.SupportsReasoning,
			ReasoningParamName:  m.ReasoningParamName,
			ReasoningParamValue: m.ReasoningParamValue,
			APIProtocol:       m.APIProtocol,
		}

		result, err := llm.CallModel(r.Context(), s.httpClient, llmModel, promptText, imageURLs, imageItems, forceReasoning)
		if err != nil {
			continue
		}

		actualModelID = tryModelID
		modelName = m.Name
		actualProvider = llm.InferProvider(m.ModelName, m.BaseURL, m.Provider)
		reasoning = result.Reasoning
		rawAnswer = result.Answer
		usage = result.Usage
		reasoningUsed = result.ReasoningUsed
		break
	}

	aiTime := time.Since(startTime).Seconds()
	totalTime := aiTime

	if rawAnswer == "" {
		errorMsg := "可用模型均调用失败，请检查模型配置或网络连接"
		if len(imageURLs) > 0 {
			errorMsg = "图片题未配置可用的多模态模型，请到模型管理页为图片题配置模型"
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": errorMsg})
		return
	}

	// Process answer
	processedAnswer := answer.ProcessAnswer(rawAnswer, qType, options, useOptionLabels)
	ocsAnswer := answer.ResolveAnswerForOcs(processedAnswer, rawAnswer, qType, options, useOptionLabels)

	// Token totals
	promptTokens := 0
	completionTokens := 0
	if usage != nil {
		promptTokens = usage.PromptTokens
		completionTokens = usage.CompletionTokens
	}
	totalTokens := promptTokens + completionTokens

	// Log to CSV
	log.AppendRecord(s.cfg.CSVLogFile, log.AnswerRecord{
		QuestionType:     qTypeName,
		Question:         question,
		Options:          strings.Join(options, " | "),
		RawAnswer:        rawAnswer,
		Reasoning:        reasoning,
		ProcessedAnswer:  processedAnswer,
		AITime:           aiTime,
		TotalTime:        totalTime,
		ModelName:        modelName,
		ReasoningUsed:    reasoningUsed,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		Provider:         actualProvider,
	})

	// Build OCS tags
	tags := buildTags(reasoningUsed, forceReasoning, actualModelID, modelName, s.modelManager)

	ocsFormat := []interface{}{
		question,
		ocsAnswer,
		map[string]interface{}{
			"ai":             true,
			"tags":           tags,
			"model":          modelName,
			"provider":       actualProvider,
			"reasoning_used": reasoningUsed,
			"ai_time":        roundFloat(aiTime, 2),
			"total_time":     roundFloat(totalTime, 2),
			"usage": map[string]int{
				"prompt_tokens":     promptTokens,
				"completion_tokens": completionTokens,
				"total_tokens":      totalTokens,
			},
		},
	}

	resp := AnswerResponse{
		Success:       true,
		Question:      question,
		Answer:        processedAnswer,
		OCSAnswer:     ocsAnswer,
		Type:          qType,
		RawAnswer:     rawAnswer,
		Model:         modelName,
		Provider:      actualProvider,
		ReasoningUsed: reasoningUsed,
		AITime:        roundFloat(aiTime, 2),
		TotalTime:     roundFloat(totalTime, 2),
		Usage: &UsageInfo{
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:      totalTokens,
		},
		OCSFormat: ocsFormat,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func normalizeOptions(options []string) []string {
	if len(options) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(options))
	for _, opt := range options {
		opt = strings.TrimSpace(opt)
		if opt != "" {
			result = append(result, opt)
		}
	}
	return result
}

func hasOptionImages(imageItems []map[string]string) bool {
	for _, item := range imageItems {
		if item["source"] == "option" {
			return true
		}
	}
	return false
}

func extractImages(question string, options []string, apiImages []string) ([]map[string]string, []string) {
	var imageItems []map[string]string
	seenURLs := map[string]bool{}

	addImage := func(url, label, source string) {
		url = cleanURL(url)
		if url == "" || seenURLs[url] {
			return
		}
		seenURLs[url] = true
		imageItems = append(imageItems, map[string]string{"url": url, "label": label, "source": source})
	}

	foundImages := imgPattern.FindAllString(question, -1)
	for i, imgURL := range foundImages {
		addImage(imgURL, formatImageLabel("Question Image", i+1), "question")
	}

	if len(options) > 0 {
		for i, opt := range options {
			optImages := imgPattern.FindAllString(opt, -1)
			for j, imgURL := range optImages {
				label := string(rune('A' + i))
				addImage(imgURL, formatImageLabel("Option", j+1, label), "option")
			}
		}
	}

	for i, img := range apiImages {
		if img == "" {
			continue
		}
		imgURL := cleanURL(img)
		if seenURLs[imgURL] {
			continue
		}
		imageItems = append(imageItems, map[string]string{"url": imgURL, "label": formatImageLabel("API Image", i+1), "source": "api"})
	}

	imageURLs := make([]string, len(imageItems))
	for i, item := range imageItems {
		imageURLs[i] = item["url"]
	}

	return imageItems, imageURLs
}

func formatImageLabel(prefix string, index int, extra ...string) string {
	label := fmt.Sprintf("%s %d", prefix, index)
	if len(extra) > 0 {
		label = fmt.Sprintf("%s %s_%d", prefix, extra[0], index)
	}
	return label
}

func cleanURL(url string) string {
	url = strings.TrimSpace(url)
	// Find the extension and strip only the fragment (everything after #)
	extMatch := regexp.MustCompile(`(?i)\.(jpg|jpeg|png|gif|bmp|webp)`).FindStringIndex(url)
	if len(extMatch) > 0 {
		// keep everything up to and including the extension, but strip fragment
		raw := url[:extMatch[1]]
		if idx := strings.Index(raw, "#"); idx != -1 {
			return raw[:idx]
		}
		return raw
	}
	if idx := strings.Index(url, "#"); idx != -1 {
		return url[:idx]
	}
	return url
}

func filterIconImages(items []map[string]string) []map[string]string {
	result := []map[string]string{}
	for _, item := range items {
		urlLower := strings.ToLower(item["url"])
		matched := false
		for kw := range iconKeywords {
			if strings.Contains(urlLower, kw) {
				matched = true
				break
			}
		}
		if !matched {
			result = append(result, item)
		}
	}
	return result
}

func determineReasoning(cfg *config.Config, qType string, hasImages bool) bool {
	if qType == config.QuestionTypeMultiple && cfg.AutoReasoningForMultiple {
		return true
	}
	if hasImages && cfg.AutoReasoningForImages {
		return true
	}
	return cfg.EnableReasoning
}

func buildTags(reasoningUsed, forceReasoning bool, modelID string, modelName string, mm *model.Manager) []map[string]string {
	tags := []map[string]string{}

	if reasoningUsed {
		tags = append(tags, map[string]string{
			"text":   "深度思考",
			"title":  "使用深度思考模式生成，答案更准确",
			"color":  "purple",
		})
		if forceReasoning {
			tags = append(tags, map[string]string{
				"text":   "自动思考",
				"title":  "多选题自动启用深度思考",
				"color":  "orange",
			})
		}
	}

	if modelID != "" {
		m := mm.GetModel(modelID)
		if m != nil {
			tagText := "内置预设"
			if !m.IsBuiltin {
				tagText = "自定义模型"
			}
			tags = append(tags, map[string]string{
				"text":   tagText,
				"title":  "使用模型: " + modelName,
				"color":  "green",
			})
		}
	}

	return tags
}

func roundFloat(v float64, decimals int) float64 {
	factor := math.Pow10(decimals)
	return math.Round(v*factor) / factor
}
