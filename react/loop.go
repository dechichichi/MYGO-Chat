package react

import (
	"agent/config"
	"fmt"
)

// Run 执行 ReAct 循环：Thought -> Action -> Observation，直到模型返回最终答案或达到最大步数
func Run(input *RunInput) (*RunResult, error) {
	if input.MaxSteps <= 0 {
		input.MaxSteps = 10
	}

	// 若有 ReAct 说明，注入到第一条 system 消息末尾
	messages := make([]config.Message, len(input.Messages))
	copy(messages, input.Messages)
	if input.ReActPrompt != "" && len(messages) > 0 && messages[0].Role == "system" {
		messages[0].Content = messages[0].Content + input.ReActPrompt
	}

	var steps []Step

	for i := 0; i < input.MaxSteps; i++ {
		content, toolCalls, err := input.Model.Invoke(messages, input.Tools)
		if err != nil {
			return nil, fmt.Errorf("react step %d invoke: %w", i+1, err)
		}

		step := Step{Thought: content}

		// 无工具调用 -> 视为最终答案
		if len(toolCalls) == 0 {
			steps = append(steps, step)
			return &RunResult{FinalAnswer: content, Steps: steps}, nil
		}

		// 有工具调用 -> 记录 Action，执行并得到 Observation，并收集 tool_call_id -> observation
		toolResults := make(map[string]string)
		for _, tc := range toolCalls {
			name := tc.Function.Name
			args := tc.Function.Arguments
			obs, err := input.Executor.Execute(name, args)
			if err != nil {
				obs = fmt.Sprintf("执行失败: %s", err.Error())
			}
			toolResults[tc.ID] = obs
			steps = append(steps, Step{
				Thought:     content,
				Action:      &Action{ToolName: name, Input: args},
				Observation: obs,
			})
		}

		// 将 assistant 消息（含 content + tool_calls）加入历史
		assistantMsg := config.Message{
			Role:    "assistant",
			Content: content,
		}
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
		messages = append(messages, assistantMsg)

		// 将每个工具结果作为 tool 消息追加
		for _, tc := range toolCalls {
			obs := toolResults[tc.ID]
			messages = append(messages, config.Message{
				Role:       "tool",
				Content:    obs,
				ToolCallID: tc.ID,
			})
		}
	}

	// 达到最大步数仍未得到无工具调用的回复，再请求一次“仅文字”的最终回答
	finalMessages := make([]config.Message, len(messages))
	copy(finalMessages, messages)
	// 不传 tools，强制模型只输出文字
	finalContent, _, err := input.Model.Invoke(finalMessages, nil)
	if err != nil {
		return nil, fmt.Errorf("react final answer: %w", err)
	}
	return &RunResult{FinalAnswer: finalContent, Steps: steps}, nil
}

// ToolExecutorFunc 将函数实现为 ToolExecutor
type ToolExecutorFunc func(toolName string, argsJSON string) (string, error)

func (f ToolExecutorFunc) Execute(toolName string, argsJSON string) (string, error) {
	return f(toolName, argsJSON)
}

// ValidateTools 校验 tools 格式（可选）
func ValidateTools(tools []map[string]interface{}) error {
	for i, t := range tools {
		if _, ok := t["type"]; !ok {
			return fmt.Errorf("tool %d: missing type", i)
		}
		fn, ok := t["function"].(map[string]interface{})
		if !ok {
			return fmt.Errorf("tool %d: missing function", i)
		}
		if _, ok := fn["name"]; !ok {
			return fmt.Errorf("tool %d: function missing name", i)
		}
	}
	return nil
}
