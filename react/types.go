package react

import "agent/config"

// Model 模型接口：与 config.ChatModel / config.FaultTolerantModel 兼容
type Model interface {
	Invoke(messages []config.Message, tools []map[string]interface{}) (string, []config.ToolCall, error)
}

// ToolExecutor 工具执行器：根据工具名和参数执行并返回观察结果
type ToolExecutor interface {
	Execute(toolName string, argsJSON string) (observation string, err error)
}

// Step 表示 ReAct 的一步：Thought（思考）-> Action（动作）-> Observation（观察）
type Step struct {
	Thought     string  `json:"thought"`               // 模型的本轮推理/思考
	Action      *Action `json:"action,omitempty"`      // 本轮调用的工具（可为空表示直接回答）
	Observation string  `json:"observation,omitempty"` // 工具执行后的观察结果
}

// Action 表示一次工具调用
type Action struct {
	ToolName string `json:"tool_name"`
	Input    string `json:"input"` // JSON 参数
}

// RunInput ReAct 运行输入
type RunInput struct {
	Model       Model
	Executor    ToolExecutor
	Tools       []map[string]interface{} // OpenAI 格式的 tools
	Messages    []config.Message         // 初始消息（含 system + 历史 + 当前 user）
	MaxSteps    int                      // 最大 Thought-Action-Observation 轮数
	ReActPrompt string                   // 追加到 system 的 ReAct 行为说明
}

// RunResult ReAct 运行结果
type RunResult struct {
	FinalAnswer string `json:"final_answer"` // 最终回复内容
	Steps       []Step `json:"steps"`        // 各轮 Thought/Action/Observation
}
