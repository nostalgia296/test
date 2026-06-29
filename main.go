package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/nostalgia296/ocs-ai/internal/config"
	"github.com/nostalgia296/ocs-ai/internal/handler"
	"github.com/nostalgia296/ocs-ai/internal/model"
)

func main() {
	cfg, err := config.Load(".env")
	if err != nil {
		log.Printf("Warning: error loading config: %v", err)
	}

	modelManager := model.NewManager("custom_models.json")

	runtime := modelManager.GetRuntimeSummary()
	printBanner(cfg, modelManager, runtime)

	answerHandler := handler.NewService(modelManager, cfg)
	healthHandler := handler.NewHealthHandler(modelManager, cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/answer", answerHandler.HandleAnswer)
	mux.HandleFunc("/api/health", healthHandler.ServeHTTP)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("Starting OCS AI Answerer on %s", addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func printBanner(cfg *config.Config, mm *model.Manager, runtime model.RuntimeSummary) {
	readyTypes := ""
	if len(runtime.ReadyQuestionTypes) > 0 {
		for _, t := range runtime.ReadyQuestionTypes {
			readyTypes += config.QuestionTypeNames[t] + "、"
		}
		readyTypes = readyTypes[:len(readyTypes)-3]
	} else {
		readyTypes = "无"
	}
	fmt.Printf("OCS AI 答题服务已启动\n")
	fmt.Printf("监听地址: %s:%d\n", cfg.Host, cfg.Port)
	fmt.Printf("可用题型: %s\n", readyTypes)

	dsThinking := "未启用"
	if cfg.DSThinkingMode {
		dsThinking = "已启用 (全局)"
	} else {
		for _, m := range mm.GetAllModels(false) {
			if m.DSThinkingMode && m.Enabled {
				dsThinking = "已启用 (模型: " + m.Name + ")"
				break
			}
		}
	}
	fmt.Printf("DS思考模式: %s\n", dsThinking)
	fmt.Printf("模型数量: %d (启用 %d)\n", runtime.ModelCount, runtime.EnabledModelCount)
}


