package philosopher

import (
	"encoding/json"
	"fmt"

	"agent/config"
	"agent/react"
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
	MemoryManager    *MemoryManager // 记忆管理器

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
	MemoryStorePath  string // SQLite 数据库路径
}

// DefaultAgentConfig 默认配置
func DefaultAgentConfig() *AgentConfig {
	return &AgentConfig{
		EnableTools:      true,
		EnableReflection: true,
		EnableRefinement: false, // 默认关闭迭代优化（消耗较大）
		MaxToolCalls:     3,
		MemoryStorePath:  "./memories.db",
	}
}

// NewAgent 创建完整的 Agent
func NewAgent(pType PhilosopherType, model *config.ChatModel, cfg *AgentConfig) (*Agent, error) {
	if cfg == nil {
		cfg = DefaultAgentConfig()
	}

	prompts := GetPhilosopherPrompts()
	prompt := prompts[pType]

	// 创建持久化记忆存储
	store, err := NewSQLiteMemoryStore(cfg.MemoryStorePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory store: %w", err)
	}

	memoryManager := NewMemoryManager(store)

	agent := &Agent{
		Type:   pType,
		Name:   prompt.Name,
		Prompt: prompt,
		Model:  model,
		Context: &AgentContext{
			PhilosopherType: pType,
			PhilosopherName: prompt.Name,
			CurrentMood:     "neutral",
			MemoryManager:   memoryManager,
		},
		EmotionAnalyzer:  NewEmotionAnalyzer(model),
		ReflectionEngine: NewReflectionEngine(model),
		Evaluator:        NewSelfEvaluator(model),
		Refiner:          NewIterativeRefiner(model),
		MemoryManager:    memoryManager,
		EnableTools:      cfg.EnableTools,
		EnableReflection: cfg.EnableReflection,
		EnableRefinement: cfg.EnableRefinement,
		MaxToolCalls:     cfg.MaxToolCalls,
	}

	return agent, nil
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

	// 4. 使用 ReAct 框架：Thought -> Action -> Observation 循环
	var tools []map[string]interface{}
	if a.EnableTools {
		tools = ToOpenAITools(GetAgentTools())
	}

	reactInput := &react.RunInput{
		Model:       a.Model,
		Executor:    a.reactToolExecutor(),
		Tools:       tools,
		Messages:    messages,
		MaxSteps:    a.MaxToolCalls,
		ReActPrompt: react.DefaultReActInstruction,
	}
	// 未启用工具时仍走 ReAct，但 tools 为空，模型会直接回复
	if !a.EnableTools {
		reactInput.Tools = nil
	}

	runResult, err := react.Run(reactInput)
	if err != nil {
		return nil, err
	}
	response := runResult.FinalAnswer

	// 从 ReAct 步骤中提取工具调用记录，供兼容原有 ToolResults 字段
	toolResults := a.toolResultsFromReActSteps(runResult.Steps)

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

	// 8. 保存对话到记忆系统（持久化）
	if a.MemoryManager != nil && a.Context.SessionID != "" {
		// 使用新的持久化记忆系统保存对话
		err := a.MemoryManager.SaveConversation(a.Context.SessionID, a.Context.PhilosopherName, userMessage, response)
		if err != nil {
			// 记录错误但继续流程
			fmt.Printf("保存记忆失败: %v\n", err)
		}
	}

	return &AgentResponse{
		Content:          response,
		EmotionLevel:     emotionLevel,
		ToolResults:      toolResults,
		ReActSteps:       runResult.Steps,
		ReflectionResult: reflectionResult,
		Evaluations:      evaluations,
	}, nil
}

// AgentResponse Agent 响应
type AgentResponse struct {
	Content          string                 `json:"content"`
	EmotionLevel     EmotionLevel           `json:"emotion_level"`
	ToolResults      []ToolResult           `json:"tool_results,omitempty"`
	ReActSteps       []react.Step           `json:"react_steps,omitempty"` // ReAct 推理步骤（Thought/Action/Observation）
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
	// 检查是否启用了记忆系统
	if a.MemoryManager == nil || a.Context.SessionID == "" {
		return ""
	}

	context := "\n\n【记忆提示】\n"

	// 获取记忆统计
	stats, err := a.MemoryManager.GetMemoryStats(a.Context.SessionID)
	if err != nil {
		// 无法获取统计信息，返回空
		return ""
	}

	// 显示记忆统计
	context += fmt.Sprintf("与这位用户已有 %d 次对话记录（%d 条长期记忆）。\n",
		stats.RecentActivity, stats.LongTermCount)

	// 获取最近的对话历史
	if stats.RecentActivity > 0 {
		recentMemories, err := a.MemoryManager.GetConversationHistory(a.Context.SessionID, a.Context.PhilosopherName, 3)
		if err == nil && len(recentMemories) > 0 {
			context += "最近的对话：\n"
			for i, memory := range recentMemories {
				if i >= 2 { // 最多显示2条
					break
				}
				context += fmt.Sprintf("- %s\n", memory.Content)
			}
		}
	}

	return context
}

// reactToolExecutor 返回供 ReAct 循环使用的工具执行器
func (a *Agent) reactToolExecutor() react.ToolExecutor {
	return react.ToolExecutorFunc(func(toolName string, argsJSON string) (string, error) {
		tc := ToolCall{Function: struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		}{Name: toolName, Arguments: argsJSON}}
		return ExecuteTool(tc, a.Context)
	})
}

// toolResultsFromReActSteps 从 ReAct 步骤中提取 ToolResult 列表（兼容原有 API）
func (a *Agent) toolResultsFromReActSteps(steps []react.Step) []ToolResult {
	var out []ToolResult
	for _, s := range steps {
		if s.Action != nil {
			out = append(out, ToolResult{
				ToolName: s.Action.ToolName,
				Input:    s.Action.Input,
				Output:   s.Observation,
			})
		}
	}
	return out
}

// SetUserID 设置用户 ID（用于区分不同用户的记忆）
func (a *Agent) SetUserID(userID string) {
	a.Context.UserID = userID
}

// SetSessionID 设置会话 ID（用于区分不同会话的记忆）
func (a *Agent) SetSessionID(sessionID string) {
	a.Context.SessionID = sessionID
}

// LoadMemory 加载记忆（从外部存储）- 已废弃，使用持久化记忆系统
func (a *Agent) LoadMemory(longTerm, shortTerm []interface{}) {
	// 已废弃，记忆现在通过持久化系统自动管理
}

// ExportMemory 导出记忆（用于持久化）- 已废弃，使用持久化记忆系统
func (a *Agent) ExportMemory() (longTerm, shortTerm []interface{}) {
	// 已废弃，记忆现在通过持久化系统自动管理
	return nil, nil
}

// GetMemoryJSON 获取记忆的 JSON 格式
func (a *Agent) GetMemoryJSON() (string, error) {
	if a.MemoryManager == nil || a.Context.SessionID == "" {
		return "{}", nil
	}

	// 获取记忆统计
	stats, err := a.MemoryManager.GetMemoryStats(a.Context.SessionID)
	if err != nil {
		return "{}", err
	}

	// 获取最近的记忆
	recentMemories, _ := a.MemoryManager.GetConversationHistory(a.Context.SessionID, a.Context.PhilosopherName, 10)

	data := map[string]interface{}{
		"stats":           stats,
		"recent_memories": recentMemories,
	}
	bytes, err := json.Marshal(data)
	return string(bytes), err
}

// ClearShortTermMemory 清除短期记忆（新会话时调用）- 已废弃
func (a *Agent) ClearShortTermMemory() {
	// 已废弃，短期记忆现在通过过期时间自动管理
	// 如果需要清理，可以调用 a.MemoryManager.Cleanup()
}

// CleanupMemories 清理过期记忆
func (a *Agent) CleanupMemories() error {
	if a.MemoryManager != nil {
		return a.MemoryManager.Cleanup()
	}
	return nil
}

// GetMemoryStats 获取记忆统计
func (a *Agent) GetMemoryStats() (*MemoryStats, error) {
	if a.MemoryManager == nil || a.Context.SessionID == "" {
		return &MemoryStats{}, nil
	}
	return a.MemoryManager.GetMemoryStats(a.Context.SessionID)
}
