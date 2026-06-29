package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	QuestionTypeSingle      = "single"
	QuestionTypeMultiple    = "multiple"
	QuestionTypeJudgement   = "judgement"
	QuestionTypeCompletion  = "completion"
	QuestionTypeImage       = "image"

	ModelAPIChat      = "chat_completions"
	ModelAPIAnthropic = "anthropic"
)

type Config struct {
	// Model provider
	ModelProvider string

	// AI parameters
	Temperature float64
	MaxTokens   int
	TopP        float64

	// DeepSeek thinking mode
	DSThinkingMode bool

	// Network config
	HTTPProxy   string
	HTTPSProxy  string
	Timeout     float64
	MaxRetries  int

	// Service config
	Host    string
	Port    int
	Debug   bool

	// Security config
	SecretKeyFile         string
	RateLimitAttempts     int
	RateLimitWindow       int

	// Logging
	CSVLogFile string
	LogLevel   string
}

var DefaultConfig = Config{
	ModelProvider:  "auto",
	Temperature:    0.1,
	MaxTokens:      500,
	TopP:           0.95,
	DSThinkingMode: false,
	Timeout:                  1200.0,
	MaxRetries:               3,
	Host:                     "0.0.0.0",
	Port:                     3000,
	Debug:                    false,
	SecretKeyFile:            ".secret_key",
	RateLimitAttempts:        5,
	RateLimitWindow:          300,
	CSVLogFile:               "ocs_answers_log.csv",
	LogLevel:                 "INFO",
}

// LoadDotenv parses a simple .env file and sets key=value pairs in os.Environ.
// It does not override existing environment variables.
func LoadDotenv(paths ...string) error {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, "\"'")
			if _, exists := os.LookupEnv(key); !exists {
				os.Setenv(key, val)
			}
		}
	}
	return nil
}

func Load(envPaths ...string) (*Config, error) {
	_ = LoadDotenv(envPaths...)

	cfg := DefaultConfig

	cfg.ModelProvider = getEnv("MODEL_PROVIDER", cfg.ModelProvider)

	cfg.Temperature = getEnvFloat("TEMPERATURE", cfg.Temperature)
	cfg.MaxTokens = clampInt(getEnvInt("MAX_TOKENS", cfg.MaxTokens), 1, 8192)
	cfg.TopP = getEnvFloat("TOP_P", cfg.TopP)
	cfg.DSThinkingMode = getEnvBool("DS_THINKING_MODE", cfg.DSThinkingMode)

	cfg.HTTPProxy = getEnv("HTTP_PROXY", "")
	cfg.HTTPSProxy = getEnv("HTTPS_PROXY", "")
	cfg.Timeout = getEnvFloat("TIMEOUT", cfg.Timeout)
	cfg.MaxRetries = getEnvInt("MAX_RETRIES", cfg.MaxRetries)

	cfg.Host = getEnv("HOST", cfg.Host)
	cfg.Port = getEnvInt("PORT", cfg.Port)
	cfg.Debug = getEnvBool("DEBUG", cfg.Debug)

	cfg.SecretKeyFile = getEnv("SECRET_KEY_FILE", cfg.SecretKeyFile)
	cfg.RateLimitAttempts = getEnvInt("RATE_LIMIT_ATTEMPTS", cfg.RateLimitAttempts)
	cfg.RateLimitWindow = getEnvInt("RATE_LIMIT_WINDOW", cfg.RateLimitWindow)

	cfg.CSVLogFile = getEnv("CSV_LOG_FILE", cfg.CSVLogFile)
	cfg.LogLevel = getEnv("LOG_LEVEL", cfg.LogLevel)

	return &cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvFloat(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return fallback
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func ClampFloat64(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// TypeNumToName maps OCS type numbers to question type names
var TypeNumToName = map[int]string{
	0: "单选题",
	1: "多选题",
	3: "填空题",
	4: "判断题",
}

// TypeNumToKey maps OCS type numbers to question type keys.
// Note: QuestionTypeImage (image) does not have a dedicated OCS type number;
// images are sent as part of questions with their original type numbers.
var TypeNumToKey = map[int]string{
	0: QuestionTypeSingle,
	1: QuestionTypeMultiple,
	3: QuestionTypeCompletion,
	4: QuestionTypeJudgement,
}

// KeyToTypeNum maps question type keys back to OCS type numbers.
// Note: QuestionTypeImage has no corresponding OCS type number.
var KeyToTypeNum = map[string]int{
	QuestionTypeSingle:     0,
	QuestionTypeMultiple:   1,
	QuestionTypeCompletion: 3,
	QuestionTypeJudgement:  4,
}

// QuestionTypeNames maps question type keys to display names
var QuestionTypeNames = map[string]string{
	QuestionTypeSingle:     "单选题",
	QuestionTypeMultiple:   "多选题",
	QuestionTypeJudgement:  "判断题",
	QuestionTypeCompletion: "填空题",
}

func ValidateConfigUpdates(data map[string]interface{}) error {
	floatKeys := map[string]bool{
		"TEMPERATURE": true, "TOP_P": true, "TIMEOUT": true,
	}
	intKeys := map[string]bool{
		"MAX_TOKENS": true, "MAX_RETRIES": true, "PORT": true,
	}
	boolKeys := map[string]bool{
		"DEBUG": true,
	}

	for key, val := range data {
		sval, ok := val.(string)
		if !ok {
			return fmt.Errorf("%s 的值类型错误", key)
		}

		if floatKeys[key] {
			if _, err := strconv.ParseFloat(sval, 64); err != nil {
				return fmt.Errorf("%s 的值无效: %s", key, sval)
			}
		} else if intKeys[key] {
			if _, err := strconv.Atoi(sval); err != nil {
				return fmt.Errorf("%s 的值无效: %s", key, sval)
			}
		} else if boolKeys[key] {
			if sval != "true" && sval != "false" {
				return fmt.Errorf("%s 仅支持 true 或 false", key)
			}
		}
	}
	return nil
}
