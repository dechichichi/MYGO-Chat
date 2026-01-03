package philosopher

import (
	"fmt"
	"strings"

	"agent/config"
)

// ==================== 反思系统 ====================

// ReflectionEngine 反思引擎
type ReflectionEngine struct {
	model *config.ChatModel
}

// ReflectionResult 反思结果
type ReflectionResult struct {
	OriginalResponse string            `json:"original_response"`
	IsAcceptable     bool              `json:"is_acceptable"`
	Issues           []ReflectionIssue `json:"issues"`
	Suggestions      []string          `json:"suggestions"`
	RevisedResponse  string            `json:"revised_response"`
	ConfidenceScore  float64           `json:"confidence_score"` // 0-1
}

// ReflectionIssue 反思发现的问题
type ReflectionIssue struct {
	Aspect      string `json:"aspect"`      // 问题方面
	Description string `json:"description"` // 问题描述
	Severity    string `json:"severity"`    // 严重程度: low, medium, high
}

// NewReflectionEngine 创建反思引擎
func NewReflectionEngine(model *config.ChatModel) *ReflectionEngine {
	return &ReflectionEngine{model: model}
}

// Reflect 对回复进行反思
func (r *ReflectionEngine) Reflect(
	response string,
	philosopherType PhilosopherType,
	context *AgentContext,
	userMessage string,
) (*ReflectionResult, error) {
	prompt := r.buildReflectionPrompt(response, philosopherType, context, userMessage)

	messages := []config.Message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: "请对以上回复进行反思和评估。"},
	}

	result, _, err := r.model.Invoke(messages, nil)
	if err != nil {
		return nil, fmt.Errorf("反思失败: %w", err)
	}

	return r.parseReflectionResult(response, result), nil
}

// buildReflectionPrompt 构建反思 Prompt
func (r *ReflectionEngine) buildReflectionPrompt(
	response string,
	philosopherType PhilosopherType,
	context *AgentContext,
	userMessage string,
) string {
	prompts := GetPhilosopherPrompts()
	characterPrompt := prompts[philosopherType]

	return fmt.Sprintf(`你是一个回复质量评估专家，需要评估以下 AI 角色的回复是否恰当。

【角色设定】
%s

【用户消息】
%s

【当前氛围】
%s

【AI 的回复】
%s

【评估维度】
1. 性格一致性：回复是否符合角色的性格特点和说话方式？
2. 情感恰当性：回复的情感温度是否与当前氛围匹配？
3. 内容相关性：回复是否针对用户的问题/话题？
4. 表达自然度：回复是否自然流畅，不像机器人？
5. 深度适当性：回复的深度是否合适（不过于肤浅也不过于复杂）？

【输出格式】
ACCEPTABLE: [true/false]
CONFIDENCE: [0-1之间的数字]

ISSUES:
- [方面]: [问题描述] (严重程度: low/medium/high)

SUGGESTIONS:
- [改进建议1]
- [改进建议2]

REVISED_RESPONSE:
[如果有问题，给出修改后的回复；如果没问题，输出"无需修改"]`,
		characterPrompt.BuildFullPrompt(),
		userMessage,
		context.CurrentMood,
		response,
	)
}

// parseReflectionResult 解析反思结果
func (r *ReflectionEngine) parseReflectionResult(originalResponse, result string) *ReflectionResult {
	reflection := &ReflectionResult{
		OriginalResponse: originalResponse,
		IsAcceptable:     true,
		ConfidenceScore:  0.8,
		Issues:           []ReflectionIssue{},
		Suggestions:      []string{},
	}

	lines := strings.Split(result, "\n")
	currentSection := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "ACCEPTABLE:") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "ACCEPTABLE:"))
			reflection.IsAcceptable = value == "true"
		} else if strings.HasPrefix(line, "CONFIDENCE:") {
			value := strings.TrimSpace(strings.TrimPrefix(line, "CONFIDENCE:"))
			fmt.Sscanf(value, "%f", &reflection.ConfidenceScore)
		} else if strings.HasPrefix(line, "ISSUES:") {
			currentSection = "issues"
		} else if strings.HasPrefix(line, "SUGGESTIONS:") {
			currentSection = "suggestions"
		} else if strings.HasPrefix(line, "REVISED_RESPONSE:") {
			currentSection = "revised"
		} else if strings.HasPrefix(line, "- ") {
			content := strings.TrimPrefix(line, "- ")
			switch currentSection {
			case "issues":
				issue := parseIssue(content)
				if issue.Aspect != "" {
					reflection.Issues = append(reflection.Issues, issue)
				}
			case "suggestions":
				reflection.Suggestions = append(reflection.Suggestions, content)
			}
		} else if currentSection == "revised" && line != "" && line != "无需修改" {
			if reflection.RevisedResponse == "" {
				reflection.RevisedResponse = line
			} else {
				reflection.RevisedResponse += "\n" + line
			}
		}
	}

	// 如果没有修改建议，使用原始回复
	if reflection.RevisedResponse == "" {
		reflection.RevisedResponse = originalResponse
	}

	return reflection
}

// parseIssue 解析问题
func parseIssue(content string) ReflectionIssue {
	issue := ReflectionIssue{Severity: "low"}

	// 格式: [方面]: [描述] (严重程度: xxx)
	if idx := strings.Index(content, ":"); idx > 0 {
		issue.Aspect = strings.TrimSpace(content[:idx])
		rest := strings.TrimSpace(content[idx+1:])

		// 提取严重程度
		if sevIdx := strings.Index(rest, "(严重程度:"); sevIdx > 0 {
			issue.Description = strings.TrimSpace(rest[:sevIdx])
			sevPart := rest[sevIdx:]
			if strings.Contains(sevPart, "high") {
				issue.Severity = "high"
			} else if strings.Contains(sevPart, "medium") {
				issue.Severity = "medium"
			}
		} else {
			issue.Description = rest
		}
	}

	return issue
}

// ReflectAndRefine 反思并优化回复（一体化方法）
func (r *ReflectionEngine) ReflectAndRefine(
	response string,
	philosopherType PhilosopherType,
	context *AgentContext,
	userMessage string,
) (string, *ReflectionResult, error) {
	result, err := r.Reflect(response, philosopherType, context, userMessage)
	if err != nil {
		return response, nil, err
	}

	// 如果可接受且置信度高，直接返回原始回复
	if result.IsAcceptable && result.ConfidenceScore >= 0.7 {
		return response, result, nil
	}

	// 否则返回修改后的回复
	return result.RevisedResponse, result, nil
}

// ==================== 自我评估系统 ====================

// SelfEvaluator 自我评估器
type SelfEvaluator struct {
	model *config.ChatModel
}

// EvaluationCriteria 评估标准
type EvaluationCriteria struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Weight      float64 `json:"weight"` // 权重 0-1
}

// EvaluationScore 评估分数
type EvaluationScore struct {
	Criteria string  `json:"criteria"`
	Score    float64 `json:"score"`    // 0-10
	Feedback string  `json:"feedback"` // 具体反馈
}

// SelfEvaluationResult 自我评估结果
type SelfEvaluationResult struct {
	TotalScore float64           `json:"total_score"` // 加权总分
	Scores     []EvaluationScore `json:"scores"`
	Summary    string            `json:"summary"`
}

// NewSelfEvaluator 创建自我评估器
func NewSelfEvaluator(model *config.ChatModel) *SelfEvaluator {
	return &SelfEvaluator{model: model}
}

// GetDefaultCriteria 获取默认评估标准
func GetDefaultCriteria() []EvaluationCriteria {
	return []EvaluationCriteria{
		{Name: "角色一致性", Description: "回复是否符合角色的性格和说话方式", Weight: 0.25},
		{Name: "情感共鸣", Description: "回复是否能与用户产生情感连接", Weight: 0.20},
		{Name: "内容深度", Description: "回复是否有思考深度，而非泛泛而谈", Weight: 0.20},
		{Name: "表达自然", Description: "回复是否自然流畅，像真人对话", Weight: 0.20},
		{Name: "问题针对", Description: "回复是否准确回应了用户的问题", Weight: 0.15},
	}
}

// Evaluate 进行自我评估
func (e *SelfEvaluator) Evaluate(
	response string,
	philosopherType PhilosopherType,
	userMessage string,
	criteria []EvaluationCriteria,
) (*SelfEvaluationResult, error) {
	if len(criteria) == 0 {
		criteria = GetDefaultCriteria()
	}

	prompt := e.buildEvaluationPrompt(response, philosopherType, userMessage, criteria)

	messages := []config.Message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: "请对回复进行评分。"},
	}

	result, _, err := e.model.Invoke(messages, nil)
	if err != nil {
		return nil, fmt.Errorf("自我评估失败: %w", err)
	}

	return e.parseEvaluationResult(result, criteria), nil
}

// buildEvaluationPrompt 构建评估 Prompt
func (e *SelfEvaluator) buildEvaluationPrompt(
	response string,
	philosopherType PhilosopherType,
	userMessage string,
	criteria []EvaluationCriteria,
) string {
	prompts := GetPhilosopherPrompts()
	characterPrompt := prompts[philosopherType]

	var criteriaDesc strings.Builder
	for i, c := range criteria {
		criteriaDesc.WriteString(fmt.Sprintf("%d. %s（权重 %.0f%%）：%s\n",
			i+1, c.Name, c.Weight*100, c.Description))
	}

	return fmt.Sprintf(`你是一个对话质量评估专家。请对以下 AI 角色的回复进行评分。

【角色设定摘要】
角色：%s

【用户消息】
%s

【AI 回复】
%s

【评估标准】
%s

【输出格式】
请为每个标准打分（0-10分），并给出具体反馈：

SCORE_1: [分数]
FEEDBACK_1: [反馈]

SCORE_2: [分数]
FEEDBACK_2: [反馈]

... 以此类推 ...

SUMMARY: [整体评价总结]`,
		characterPrompt.Name,
		userMessage,
		response,
		criteriaDesc.String(),
	)
}

// parseEvaluationResult 解析评估结果
func (e *SelfEvaluator) parseEvaluationResult(result string, criteria []EvaluationCriteria) *SelfEvaluationResult {
	evaluation := &SelfEvaluationResult{
		Scores: make([]EvaluationScore, len(criteria)),
	}

	lines := strings.Split(result, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		for i := range criteria {
			scoreKey := fmt.Sprintf("SCORE_%d:", i+1)
			feedbackKey := fmt.Sprintf("FEEDBACK_%d:", i+1)

			if strings.HasPrefix(line, scoreKey) {
				value := strings.TrimSpace(strings.TrimPrefix(line, scoreKey))
				fmt.Sscanf(value, "%f", &evaluation.Scores[i].Score)
				evaluation.Scores[i].Criteria = criteria[i].Name
			} else if strings.HasPrefix(line, feedbackKey) {
				evaluation.Scores[i].Feedback = strings.TrimSpace(strings.TrimPrefix(line, feedbackKey))
			}
		}

		if strings.HasPrefix(line, "SUMMARY:") {
			evaluation.Summary = strings.TrimSpace(strings.TrimPrefix(line, "SUMMARY:"))
		}
	}

	// 计算加权总分
	totalWeight := 0.0
	weightedSum := 0.0
	for i, score := range evaluation.Scores {
		if i < len(criteria) {
			weightedSum += score.Score * criteria[i].Weight
			totalWeight += criteria[i].Weight
		}
	}
	if totalWeight > 0 {
		evaluation.TotalScore = weightedSum / totalWeight
	}

	return evaluation
}

// ==================== 迭代优化器 ====================

// IterativeRefiner 迭代优化器
type IterativeRefiner struct {
	model            *config.ChatModel
	reflectionEngine *ReflectionEngine
	evaluator        *SelfEvaluator
	maxIterations    int
	targetScore      float64
}

// NewIterativeRefiner 创建迭代优化器
func NewIterativeRefiner(model *config.ChatModel) *IterativeRefiner {
	return &IterativeRefiner{
		model:            model,
		reflectionEngine: NewReflectionEngine(model),
		evaluator:        NewSelfEvaluator(model),
		maxIterations:    3,
		targetScore:      8.0,
	}
}

// SetMaxIterations 设置最大迭代次数
func (r *IterativeRefiner) SetMaxIterations(n int) {
	r.maxIterations = n
}

// SetTargetScore 设置目标分数
func (r *IterativeRefiner) SetTargetScore(score float64) {
	r.targetScore = score
}

// RefineResponse 迭代优化回复
func (r *IterativeRefiner) RefineResponse(
	initialResponse string,
	philosopherType PhilosopherType,
	context *AgentContext,
	userMessage string,
) (string, []SelfEvaluationResult, error) {
	currentResponse := initialResponse
	evaluations := []SelfEvaluationResult{}

	for i := 0; i < r.maxIterations; i++ {
		// 评估当前回复
		eval, err := r.evaluator.Evaluate(currentResponse, philosopherType, userMessage, nil)
		if err != nil {
			return currentResponse, evaluations, err
		}
		evaluations = append(evaluations, *eval)

		// 如果达到目标分数，停止迭代
		if eval.TotalScore >= r.targetScore {
			break
		}

		// 反思并优化
		refined, _, err := r.reflectionEngine.ReflectAndRefine(
			currentResponse, philosopherType, context, userMessage)
		if err != nil {
			return currentResponse, evaluations, err
		}

		// 如果没有变化，停止迭代
		if refined == currentResponse {
			break
		}

		currentResponse = refined
	}

	return currentResponse, evaluations, nil
}
