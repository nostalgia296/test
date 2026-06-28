package prompt

import "github.com/nostalgia296/ocs-ai/internal/config"

// Build generates a question-specific prompt based on the question type.
func Build(question string, options []string, qType string, useOptionLabels bool) string {
	switch qType {
	case config.QuestionTypeSingle:
		return buildSingleChoice(question, options, useOptionLabels)
	case config.QuestionTypeMultiple:
		return buildMultipleChoice(question, options, useOptionLabels)
	case config.QuestionTypeJudgement:
		return buildJudgement(question, options)
	case config.QuestionTypeCompletion:
		return buildCompletion(question)
	default:
		return buildDefault(question, options)
	}
}

func buildSingleChoice(question string, options []string, useOptionLabels bool) string {
	optionsText := formatOptions(options)
	answerFormat := `4. 回答格式：直接输出选项字母，例如 A、B、C、D
5. 如果选项是图片，必须根据图片标签选择对应字母，不要输出图片里的数值、公式或文字
6. 只输出一个字母，不要有任何解释、分析或额外文字`
	example := `如果正确答案是 A 选项，则只输出：A`
	if !useOptionLabels {
		answerFormat = `4. 回答格式：直接输出选项内容，不要包含A、B、C等标识符
5. 只输出答案内容，不要有任何解释、分析或额外文字`
		example = `如果正确答案是选项'北京'，则只输出：北京`
	}

	return "你是一个专业的在线考试答题助手，请严格按照要求回答。\n\n【题目类型】单选题（只能选择一个正确答案）\n\n【题目】\n" + question + "\n\n【选项】\n" + optionsText + "\n\n【回答要求】\n1. 仔细分析题目和所有选项\n2. 只选择一个最正确的答案\n3. 必须从给定的选项中选择，不能自己编造\n" + answerFormat + "\n\n【示例】\n" + example + "\n\n现在请回答上述题目："
}

func buildMultipleChoice(question string, options []string, useOptionLabels bool) string {
	optionsText := formatOptions(options)
	answerFormat := `5. 回答格式：A#B#C（只包含选项字母，多个答案之间用井号#分隔）
6. 如果选项是图片，必须根据图片标签选择对应字母，不要输出图片里的数值、公式或文字
7. 只输出选项字母，不要有任何解释、分析或额外文字`
	example := `如果正确答案是 A 和 C 两个选项，则输出：A#C`
	if !useOptionLabels {
		answerFormat = `5. 回答格式：选项1#选项2#选项3（不要包含A、B、C等标识符）
6. 只输出答案内容，不要有任何解释、分析或额外文字`
		example = `如果正确答案是'北京'和'上海'两个选项，则输出：北京#上海`
	}

	return "你是一个专业的在线考试答题助手，请严格按照要求回答。\n\n【题目类型】多选题（可能有多个正确答案）\n\n【题目】\n" + question + "\n\n【选项】\n" + optionsText + "\n\n【回答要求】\n1. 仔细分析题目，找出所有正确的选项\n2. 多选题通常有2个或以上的正确答案\n3. 必须从给定的选项中选择，不能自己编造\n4. 多个答案之间用井号#分隔\n" + answerFormat + "\n\n【示例】\n" + example + "\n\n现在请回答上述题目："
}

func buildJudgement(question string, options []string) string {
	optionsText := "无固定选项"
	if len(options) > 0 {
		optionsText = joinOptions(options)
	}

	return "你是一个专业的在线考试答题助手，请严格按照要求回答。\n\n【题目类型】判断题（判断对错/是否）\n\n【题目】\n" + question + "\n\n【可选答案】\n" + optionsText + "\n\n【回答要求】\n1. 仔细分析题目陈述是否正确\n2. 必须从给定的选项中选择（如：正确/错误、对/错、是/否、√/×等）\n3. 只输出一个判断结果\n4. 不要有任何解释、分析或额外文字\n\n【示例】\n如果题目陈述正确，且选项中有'正确'，则输出：正确\n\n现在请判断上述题目："
}

func buildCompletion(question string) string {
	return "你是一个专业的在线考试答题助手，请严格按照要求回答。\n\n【题目类型】填空题\n\n【题目】\n" + question + "\n\n【回答要求】\n1. 仔细理解题目要求\n2. 给出准确、简洁的答案\n3. 如果有多个空，答案之间用井号#分隔\n4. 答案要具体、准确，避免模糊表述\n5. 只输出答案内容，不要有序号、解释或额外文字\n\n【示例】\n- 单空题：如果答案是'北京'，则输出：北京\n- 多空题：如果答案是'氢'和'氧'，则输出：氢#氧\n\n现在请回答上述填空题："
}

func buildDefault(question string, options []string) string {
	optionsText := "无固定选项"
	if len(options) > 0 {
		var parts []string
		for _, opt := range options {
			parts = append(parts, "- "+opt)
		}
		optionsText = joinOptions(parts)
	}

	return "请回答以下问题：\n\n【题目】\n" + question + "\n\n【选项】\n" + optionsText + "\n\n【要求】\n1. 给出准确的答案\n2. 如果有多个答案，用#分隔\n3. 只输出答案，不要解释\n\n请回答："
}

func formatOptions(options []string) string {
	parts := make([]string, len(options))
	for i, opt := range options {
		parts[i] = string(rune('A'+i)) + ". " + opt
	}
	return joinOptions(parts)
}

func joinOptions(parts []string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "\n"
		}
		result += p
	}
	return result
}
