package philosopher

import (
	"encoding/json"
	"fmt"

	"agent/config"
)

// ==================== 完整 Agent 实现 ====================

// Agent 完整的 Agent 实现，集成所有能力
type Agent struct {
	// 基础信息
	Type   PhilosopherType
	Name   string
	Prompt *PhilosopherPrompt
	Model  *config.ChatModel

	// Agent 上下文
	Context *AgentContext

	// 子系统
	EmotionAnalyzer  *EmotionAnalyzer
	ReflectionEngine *ReflectionEngine
	Evaluator        *SelfEvaluator
	Refiner          *IterativeRefiner

	// 配置
	EnableTools      bool // 是否启用工具调用
	EnableReflection bool // 是否启用反思
	EnableRefinement bool // 是否启用迭代优化
	MaxToolCalls     int  // 最大工具调用次数
}

// AgentConfig Agent 配置
type AgentConfig struct {
	EnableTools      bool
	EnableReflection bool
	EnableRefinement bool
	MaxToolCalls     int
}

// DefaultAgentConfig 默认配置
func DefaultAgentConfig() *AgentConfig {
	return &AgentConfig{
		EnableTools:      true,
		EnableReflection: true,
		EnableRefinement: false, // 默认关闭迭代优化（消耗较大）
		MaxToolCalls:     3,
	}
}

// NewAgent 创建完整的 Agent
func NewAgent(pType PhilosopherType, model *config.ChatModel, cfg *AgentConfig) *Agent {
	if cfg == nil {
		cfg = DefaultAgentConfig()
	}

	prompts := GetPhilosopherPrompts()
	prompt := prompts[pType]

	agent := &Agent{
		Type:   pType,
		Name:   prompt.Name,
		Prompt: prompt,
		Model:  model,
		Context: &AgentContext{
			PhilosopherType: pType,
			PhilosopherName: prompt.Name,
			ShortTermMemory: []MemoryItem{},
			LongTermMemory:  []MemoryItem{},
			CurrentMood:     "neutral",
		},
		EmotionAnalyzer:  NewEmotionAnalyzer(model),
		ReflectionEngine: NewReflectionEngine(model),
		Evaluator:        NewSelfEvaluator(model),
		Refiner:          NewIterativeRefiner(model),
		EnableTools:      cfg.EnableTools,
		EnableReflection: cfg.EnableReflection,
		EnableRefinement: cfg.EnableRefinement,
		MaxToolCalls:     cfg.MaxToolCalls,
	}

	return agent
}

// Chat Agent 的主对话方法
func (a *Agent) Chat(userMessage string, history []config.Message) (*AgentResponse, error) {
	// 1. 情绪分析
	emotionLevel := a.EmotionAnalyzer.Analyze(userMessage)
	a.Context.CurrentMood = string(emotionLevel)

	// 2. 构建系统 Prompt
	systemPrompt := a.buildAgentPrompt(emotionLevel)

	// 3. 构建消息
	messages := []config.Message{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)
	messages = append(messages, config.Message{Role: "user", Content: userMessage})

	// 4. 准备工具（如果启用）
	var tools []map[string]interface{}
	if a.EnableTools {
		tools = ToOpenAITools(GetAgentTools())
	}

	// 5. 调用模型（可能包含工具调用循环）
	response, toolResults, err := a.invokeWithTools(messages, tools)
	if err != nil {
		return nil, err
	}

	// 6. 反思（如果启用）
	var reflectionResult *ReflectionResult
	if a.EnableReflection {
		response, reflectionResult, err = a.ReflectionEngine.ReflectAndRefine(
			response, a.Type, a.Context, userMessage)
		if err != nil {
			// 反思失败不影响主流程，记录日志即可
			reflectionResult = nil
		}
	}

	// 7. 迭代优化（如果启用）
	var evaluations []SelfEvaluationResult
	if a.EnableRefinement {
		response, evaluations, _ = a.Refiner.RefineResponse(
			response, a.Type, a.Context, userMessage)
	}

	// 8. 保存对话到短期记忆
	a.Context.ShortTermMemory = append(a.Context.ShortTermMemory, MemoryItem{
		Type:       "event",
		Content:    fmt.Sprintf("用户说：%s", userMessage),
		Importance: 0.5,
	})

	return &AgentResponse{
		Content:          response,
		EmotionLevel:     emotionLevel,
		ToolResults:      toolResults,
		ReflectionResult: reflectionResult,
		Evaluations:      evaluations,
	}, nil
}

// AgentResponse Agent 响应
type AgentResponse struct {
	Content          string                 `json:"content"`
	EmotionLevel     EmotionLevel           `json:"emotion_level"`
	ToolResults      []ToolResult           `json:"tool_results,omitempty"`
	ReflectionResult *ReflectionResult      `json:"reflection_result,omitempty"`
	Evaluations      []SelfEvaluationResult `json:"evaluations,omitempty"`
}

// ToolResult 工具调用结果
type ToolResult struct {
	ToolName string `json:"tool_name"`
	Input    string `json:"input"`
	Output   string `json:"output"`
}

// buildAgentPrompt 构建 Agent Prompt
func (a *Agent) buildAgentPrompt(emotionLevel EmotionLevel) string {
	basePrompt := a.Prompt.BuildFullPrompt()

	// 添加情绪感知指导
	emotionGuidance := a.buildEmotionGuidance(emotionLevel)

	// 添加工具使用说明（如果启用）
	toolGuidance := ""
	if a.EnableTools {
		toolGuidance = `

【可用工具】
你可以使用以下工具来增强你的回复：
1. recall_memory - 回忆与用户的过往对话
2. save_memory - 保存重要信息到记忆
3. search_lyrics - 搜索歌词找灵感
4. sense_atmosphere - 感知当前对话氛围
5. reflect_response - 反思自己的回复

在需要时，你可以调用这些工具来获取更多信息或进行自我检查。`
	}

	// 添加记忆上下文
	memoryContext := a.buildMemoryContext()

	return basePrompt + emotionGuidance + toolGuidance + memoryContext
}

// buildEmotionGuidance 构建情绪指导
func (a *Agent) buildEmotionGuidance(level EmotionLevel) string {
	guidance := "\n\n【情绪感知】\n"

	switch level {
	case EmotionPain:
		guidance += `检测到用户可能处于痛苦状态。
- 用你的方式给予温暖和陪伴
- 不要说空话，用真诚的话语回应
- 可以分享你自己面对困难时的感受`

	case EmotionConfused:
		guidance += `检测到用户可能处于迷茫状态。
- 用你的方式帮助他理清思路
- 分享你自己迷茫时的经历
- 给予鼓励和支持`

	case EmotionComplaining:
		guidance += `检测到用户可能在抱怨。
- 先倾听和理解
- 然后用你的方式引导他思考
- 帮助他找到积极的方向`

	case EmotionExcusing:
		guidance += `检测到用户可能在找借口。
- 用你的方式指出问题
- 但也要给予理解
- 鼓励他面对现实`

	default:
		guidance += `用户情绪正常。
- 正常发挥你的性格特点
- 用你独特的方式交流`
	}

	return guidance
}

// buildMemoryContext 构建记忆上下文
func (a *Agent) buildMemoryContext() string {
	if len(a.Context.LongTermMemory) == 0 && len(a.Context.ShortTermMemory) == 0 {
		return ""
	}

	context := "\n\n【记忆提示】\n"

	// 长期记忆（重要的）
	if len(a.Context.LongTermMemory) > 0 {
		context += "你记得关于这位用户的一些事情：\n"
		for i, m := range a.Context.LongTermMemory {
			if i >= 3 {
				break
			}
			context += fmt.Sprintf("- %s\n", m.Content)
		}
	}

	// 短期记忆摘要
	if len(a.Context.ShortTermMemory) > 5 {
		context += fmt.Sprintf("\n这次对话已经进行了 %d 轮。\n", len(a.Context.ShortTermMemory)/2)
	}

	return context
}

// invokeWithTools 带工具调用的模型调用
func (a *Agent) invokeWithTools(messages []config.Message, tools []map[string]interface{}) (string, []ToolResult, error) {
	var toolResults []ToolResult
	currentMessages := make([]config.Message, len(messages))
	copy(currentMessages, messages)

	for i := 0; i < a.MaxToolCalls; i++ {
		// 调用模型
		content, toolCalls, err := a.Model.Invoke(currentMessages, tools)
		if err != nil {
			return "", toolResults, err
		}

		// 如果没有工具调用，返回内容
		if len(toolCalls) == 0 {
			return content, toolResults, nil
		}

		// 添加带有 tool_calls 的 assistant 消息
		assistantMsg := config.Message{
			Role:    "assistant",
			Content: content,
		}
		// 转换 tool_calls
		for _, tc := range toolCalls {
			assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, config.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		currentMessages = append(currentMessages, assistantMsg)

		// 处理每个工具调用
		for _, tc := range toolCalls {
			// 转换工具调用格式
			toolCall := ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
			}
			toolCall.Function.Name = tc.Function.Name
			toolCall.Function.Arguments = tc.Function.Arguments

			// 执行工具
			result, err := ExecuteTool(toolCall, a.Context)
			if err != nil {
				result = fmt.Sprintf("工具执行失败: %s", err.Error())
			}

			toolResults = append(toolResults, ToolResult{
				ToolName: tc.Function.Name,
				Input:    tc.Function.Arguments,
				Output:   result,
			})

			// 添加工具结果消息（需要 tool_call_id）
			currentMessages = append(currentMessages, config.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	// 达到最大工具调用次数，最后再调用一次获取最终回复
	finalContent, _, err := a.Model.Invoke(currentMessages, nil)
	return finalContent, toolResults, err
}

// SetUserID 设置用户 ID（用于区分不同用户的记忆）
func (a *Agent) SetUserID(userID string) {
	a.Context.UserID = userID
}

// LoadMemory 加载记忆（从外部存储）
func (a *Agent) LoadMemory(longTerm, shortTerm []MemoryItem) {
	a.Context.LongTermMemory = longTerm
	a.Context.ShortTermMemory = shortTerm
}

// ExportMemory 导出记忆（用于持久化）
func (a *Agent) ExportMemory() (longTerm, shortTerm []MemoryItem) {
	return a.Context.LongTermMemory, a.Context.ShortTermMemory
}

// GetMemoryJSON 获取记忆的 JSON 格式
func (a *Agent) GetMemoryJSON() (string, error) {
	data := map[string]interface{}{
		"long_term":  a.Context.LongTermMemory,
		"short_term": a.Context.ShortTermMemory,
	}
	bytes, err := json.Marshal(data)
	return string(bytes), err
}

// ClearShortTermMemory 清除短期记忆（新会话时调用）
func (a *Agent) ClearShortTermMemory() {
	a.Context.ShortTermMemory = []MemoryItem{}
}
