package answer

import (
	"regexp"
	"strings"

	"github.com/nostalgia296/ocs-ai/internal/config"
)

var (
	cleanPrefixRe = regexp.MustCompile(`^(答案[是为：:]*|正确答案[是为：:]*|选择[：:]*)`)
	optionPrefixRe = regexp.MustCompile(`^[A-Z][.、)]\s*`)
	punctuationRe = regexp.MustCompile(`[。，、；：！？\s]`)
)

// CleanAnswer removes obvious formatting marks without modifying content.
func CleanAnswer(text string) string {
	if text == "" {
		return ""
	}

	// Remove common prefixes at start of line
	text = cleanPrefixRe.ReplaceAllString(text, "")
	text = strings.TrimSpace(text)

	// Remove markdown formatting symbols
	text = strings.ReplaceAll(text, "*", "")
	text = strings.ReplaceAll(text, "`", "")
	text = strings.ReplaceAll(text, "_", "")
	text = strings.TrimSpace(text)

	// Remove leading option identifiers (e.g. "A. ")
	text = optionPrefixRe.ReplaceAllString(text, "")
	text = strings.TrimSpace(text)

	return text
}

// MatchOption checks if answer matches option using multiple strategies.
func MatchOption(answer, option string) bool {
	answer = strings.TrimSpace(answer)
	option = strings.TrimSpace(option)

	if answer == "" || option == "" {
		return false
	}

	// Exact match (case insensitive, ignoring spaces)
	if strings.EqualFold(answer, option) {
		return true
	}

	// Remove punctuation and match
	answerClean := punctuationRe.ReplaceAllString(answer, "")
	optionClean := punctuationRe.ReplaceAllString(option, "")
	if strings.EqualFold(answerClean, optionClean) {
		return true
	}

	return false
}

var (
	charRangePatternCache = make(map[string]*regexp.Regexp)
	compactPattern        = regexp.MustCompile(`[^A-Z]`)
)

// ExtractOptionIndexes extracts option indexes (0-based) from A/B/C style answers.
func ExtractOptionIndexes(answer string, options []string) []int {
	if answer == "" || len(options) == 0 {
		return nil
	}

	maxLabel := string(rune('A' + min(len(options), 26) - 1))
	upperAnswer := strings.ToUpper(answer)
	charRange := "A-" + maxLabel

	charRangeRe, ok := charRangePatternCache[charRange]
	if !ok {
		patterns := []string{
			`选项\s*([` + charRange + `])`,
			`OPTION\s*([` + charRange + `])`,
			`(?<![A-Z0-9])([` + charRange + `])(?![A-Z0-9])`,
		}
		charRangeRe = regexp.MustCompile(strings.Join(patterns, "|"))
		charRangePatternCache[charRange] = charRangeRe
	}

	indexes := []int{}
	for _, match := range charRangeRe.FindAllStringSubmatch(upperAnswer, -1) {
		if len(match) > 1 {
			idx := int(match[1][0] - 'A')
			if 0 <= idx && idx < len(options) && !contains(indexes, idx) {
				indexes = append(indexes, idx)
			}
		}
	}

	// Compact: extract all A-Z letters
	compact := compactPattern.ReplaceAllString(upperAnswer, "")
	if len(indexes) == 0 && compact != "" {
		allValid := true
		for _, ch := range compact {
			if ch < 'A' || ch > rune(maxLabel[0]) {
				allValid = false
				break
			}
		}
		if allValid {
			for _, ch := range compact {
				idx := int(ch - 'A')
				if !contains(indexes, idx) {
					indexes = append(indexes, idx)
				}
			}
		}
	}

	return indexes
}

// OptionIndexesToAnswer converts option indexes to joined answer string.
func OptionIndexesToAnswer(indexes []int, options []string) string {
	parts := []string{}
	for _, idx := range indexes {
		if 0 <= idx && idx < len(options) {
			parts = append(parts, strings.TrimSpace(options[idx]))
		}
	}
	return strings.Join(parts, "#")
}

// ResolveAnswerForOcs generates the final answer for OCS matching.
// When options contain images, the model returns A/B/C/D but OCS needs
// the actual option content (image URLs) for matching.
func ResolveAnswerForOcs(processedAnswer, rawAnswer, qType string, options []string, useOptionLabels bool) string {
	if processedAnswer == "" {
		return processedAnswer
	}

	if !useOptionLabels || (qType != config.QuestionTypeSingle && qType != config.QuestionTypeMultiple) || len(options) == 0 {
		return processedAnswer
	}

	indexes := ExtractOptionIndexes(processedAnswer, options)
	if len(indexes) == 0 && rawAnswer != "" {
		indexes = ExtractOptionIndexes(rawAnswer, options)
	}

	if len(indexes) > 0 {
		resolved := OptionIndexesToAnswer(indexes, options)
		if resolved != "" {
			return resolved
		}
	}

	return processedAnswer
}

// ProcessAnswer processes and cleans the AI's raw answer based on question type.
func ProcessAnswer(rawAnswer, qType string, options []string, useOptionLabels bool) string {
	if rawAnswer == "" {
		return ""
	}

	rawAnswer = strings.TrimSpace(rawAnswer)

	switch qType {
	case config.QuestionTypeSingle:
		return processSingleChoice(rawAnswer, options, useOptionLabels)
	case config.QuestionTypeMultiple:
		return processMultipleChoice(rawAnswer, options, useOptionLabels)
	case config.QuestionTypeJudgement:
		return processJudgement(rawAnswer, options)
	case config.QuestionTypeCompletion:
		cleaned := CleanAnswer(rawAnswer)
		if cleaned != "" {
			return cleaned
		}
		return rawAnswer
	default:
		cleaned := CleanAnswer(rawAnswer)
		if cleaned != "" {
			return cleaned
		}
		return rawAnswer
	}
}

func processSingleChoice(rawAnswer string, options []string, useOptionLabels bool) string {
	if len(options) == 0 {
		return CleanAnswer(rawAnswer)
	}

	if useOptionLabels {
		indexes := ExtractOptionIndexes(rawAnswer, options)
		if len(indexes) > 0 {
			return string(rune('A' + indexes[0]))
		}
	}

	// Try matching with raw answer first
	for _, opt := range options {
		if MatchOption(rawAnswer, opt) {
			return strings.TrimSpace(opt)
		}
	}

	// Try matching after cleaning
	cleaned := CleanAnswer(rawAnswer)
	for _, opt := range options {
		if MatchOption(cleaned, opt) {
			return strings.TrimSpace(opt)
		}
	}

	// Return cleaned answer as-is
	if cleaned != "" {
		return cleaned
	}
	return rawAnswer
}

func processMultipleChoice(rawAnswer string, options []string, useOptionLabels bool) string {
	if len(options) == 0 {
		return CleanAnswer(rawAnswer)
	}

	if useOptionLabels {
		indexes := ExtractOptionIndexes(rawAnswer, options)
		if len(indexes) > 0 {
			parts := []string{}
			for _, idx := range indexes {
				parts = append(parts, string(rune('A'+idx)))
			}
			return strings.Join(parts, "#")
		}
	}

	// Try matching with raw answer
	matchedIndexes := []int{}
	for i, opt := range options {
		if MatchOption(rawAnswer, opt) {
			matchedIndexes = append(matchedIndexes, i)
		}
	}
	if len(matchedIndexes) > 0 {
		return OptionIndexesToAnswer(matchedIndexes, options)
	}

	// Try matching after cleaning
	cleaned := CleanAnswer(rawAnswer)
	for i, opt := range options {
		if MatchOption(cleaned, opt) {
			matchedIndexes = append(matchedIndexes, i)
		}
	}
	if len(matchedIndexes) > 0 {
		return OptionIndexesToAnswer(matchedIndexes, options)
	}

	// Return cleaned answer as-is
	if cleaned != "" {
		return cleaned
	}
	return rawAnswer
}

func processJudgement(rawAnswer string, options []string) string {
	if len(options) == 0 {
		return CleanAnswer(rawAnswer)
	}

	cleaned := CleanAnswer(rawAnswer)

	for _, opt := range options {
		if MatchOption(cleaned, opt) {
			return strings.TrimSpace(opt)
		}
	}

	return cleaned
}

func contains(arr []int, val int) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
