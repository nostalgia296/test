package log

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const CSVHeaders = "时间戳,题型,题目,选项,原始回答,思考过程,处理后答案,AI耗时(秒),总耗时(秒),模型,思考模式,输入Token,输出Token,总Token,费用(元),提供商"

type AnswerRecord struct {
	Timestamp      string  `csv:"时间戳"`
	QuestionType   string  `csv:"题型"`
	Question       string  `csv:"题目"`
	Options        string  `csv:"选项"`
	RawAnswer      string  `csv:"原始回答"`
	Reasoning      string  `csv:"思考过程"`
	ProcessedAnswer string `csv:"处理后答案"`
	AITime         float64 `csv:"AI耗时(秒)"`
	TotalTime      float64 `csv:"总耗时(秒)"`
	ModelName      string  `csv:"模型"`
	ReasoningUsed  bool    `csv:"思考模式"`
	PromptTokens   int     `csv:"输入Token"`
	CompletionTokens int   `csv:"输出Token"`
	TotalTokens    int     `csv:"总Token"`
	Cost           float64 `csv:"费用(元)"`
	Provider       string  `csv:"提供商"`
}

func (r *AnswerRecord) ToSlice() []string {
	return []string{
		r.Timestamp,
		r.QuestionType,
		r.Question,
		r.Options,
		r.RawAnswer,
		r.Reasoning,
		r.ProcessedAnswer,
		fmt.Sprintf("%.2f", r.AITime),
		fmt.Sprintf("%.2f", r.TotalTime),
		r.ModelName,
		boolToStr(r.ReasoningUsed),
		fmt.Sprintf("%d", r.PromptTokens),
		fmt.Sprintf("%d", r.CompletionTokens),
		fmt.Sprintf("%d", r.TotalTokens),
		fmt.Sprintf("%.4f", r.Cost),
		r.Provider,
	}
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func strToBool(s string) bool {
	return strings.ToLower(strings.TrimSpace(s)) == "true"
}

func AppendRecord(filePath string, record AnswerRecord) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	CheckAndFixHeader(filePath)

	record.Timestamp = time.Now().Format("2006-01-02 15:04:05")

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	writer.UseCRLF = true

	if err := writer.Write(record.ToSlice()); err != nil {
		return err
	}

	writer.Flush()
	return writer.Error()
}

func CheckAndFixHeader(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return writeHeader(filePath)
		}
		return err
	}

	if len(data) == 0 {
		return writeHeader(filePath)
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 {
		return writeHeader(filePath)
	}

	firstLine := strings.TrimSpace(lines[0])
	if firstLine == CSVHeaders {
		return nil
	}

	// Header mismatch - backup and rewrite
	backupPath := filePath + ".backup"
	os.Rename(filePath, backupPath)

	return writeHeader(filePath)
}

func writeHeader(filePath string) error {
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(CSVHeaders + "\n")
	return err
}

// Stats holds CSV statistics.
type Stats struct {
	TotalRecords      int
	TypeDistribution  map[string]int
	ReasoningCount    int
	TotalAITime       float64
	TotalPromptTokens int
	TotalOutputTokens int
	ModelCount        map[string]int
}

func ReadStats(filePath string) (*Stats, error) {
	stats := &Stats{
		TypeDistribution: make(map[string]int),
		ModelCount:       make(map[string]int),
	}

	records, err := ReadAllRecords(filePath)
	if err != nil {
		return nil, err
	}

	stats.TotalRecords = len(records)
	for _, r := range records {
		stats.TypeDistribution[r.QuestionType]++
		if r.ReasoningUsed {
			stats.ReasoningCount++
		}
		stats.TotalAITime += r.AITime
		stats.TotalPromptTokens += r.PromptTokens
		stats.TotalOutputTokens += r.CompletionTokens
		if r.ModelName != "" {
			stats.ModelCount[r.ModelName]++
		}
	}

	return stats, nil
}

func ReadAllRecords(filePath string) ([]AnswerRecord, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var records []AnswerRecord

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || line == CSVHeaders {
			continue
		}

		fields, err := parseCSVLine(line)
		if err != nil {
			continue
		}

		if len(fields) < 16 {
			continue
		}

		aiTime := 0.0
		fmt.Sscanf(fields[7], "%f", &aiTime)
		totalTime := 0.0
		fmt.Sscanf(fields[8], "%f", &totalTime)
		promptTokens := 0
		fmt.Sscanf(fields[11], "%d", &promptTokens)
		completionTokens := 0
		fmt.Sscanf(fields[12], "%d", &completionTokens)
		totalTokens := 0
		fmt.Sscanf(fields[13], "%d", &totalTokens)
		cost := 0.0
		fmt.Sscanf(fields[14], "%f", &cost)

		records = append(records, AnswerRecord{
			Timestamp:       fields[0],
			QuestionType:    fields[1],
			Question:        fields[2],
			Options:         fields[3],
			RawAnswer:       fields[4],
			Reasoning:       fields[5],
			ProcessedAnswer: fields[6],
			AITime:          aiTime,
			TotalTime:       totalTime,
			ModelName:       fields[9],
			ReasoningUsed:   strToBool(fields[10]),
			PromptTokens:    promptTokens,
			CompletionTokens: completionTokens,
			TotalTokens:     totalTokens,
			Cost:            cost,
			Provider:        fields[15],
		})
	}

	return records, nil
}

func parseCSVLine(line string) ([]string, error) {
	var fields []string
	var current strings.Builder
	inQuotes := false

	for i := 0; i < len(line); i++ {
		ch := line[i]
		if ch == '"' {
			if inQuotes && i+1 < len(line) && line[i+1] == '"' {
				current.WriteByte('"')
				i++
			} else {
				inQuotes = !inQuotes
			}
		} else if ch == ',' && !inQuotes {
			fields = append(fields, current.String())
			current.Reset()
		} else {
			current.WriteByte(ch)
		}
	}
	fields = append(fields, current.String())

	return fields, nil
}

func ClearFile(filePath string) error {
	return writeHeader(filePath)
}
