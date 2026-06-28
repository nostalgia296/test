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
	printBanner(cfg, runtime)

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

func printBanner(cfg *config.Config, runtime model.RuntimeSummary) {
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
	fmt.Printf("深度思考: %s\n", boolStr(cfg.EnableReasoning))
	fmt.Printf("模型数量: %d (启用 %d)\n", runtime.ModelCount, runtime.EnabledModelCount)
}

func boolStr(b bool) string {
	if b {
		return "已启用"
	}
	return "未启用"
}
