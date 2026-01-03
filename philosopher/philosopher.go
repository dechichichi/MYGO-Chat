package philosopher

import (
	"agent/config"
)

// Philosopher 哲学家 Agent
type Philosopher struct {
	Type          PhilosopherType
	Name          string
	Prompt        *PhilosopherPrompt
	Model         *config.ChatModel
	CurrentStance string // 当前立场（正方/反方）
	IsForced      bool   // 是否被强制指定立场
}

// NewPhilosopher 创建哲学家
func NewPhilosopher(pType PhilosopherType, model *config.ChatModel) *Philosopher {
	prompts := GetPhilosopherPrompts()
	prompt := prompts[pType]

	return &Philosopher{
		Type:   pType,
		Name:   prompt.Name,
		Prompt: prompt,
		Model:  model,
	}
}

// Chat 与哲学家进行一对一对话
func (p *Philosopher) Chat(messages []config.Message, emotionLevel EmotionLevel) (string, error) {
	// 根据情绪级别调整 Prompt
	systemPrompt := p.buildEmotionAwarePrompt(emotionLevel)

	// 构建完整消息
	fullMessages := []config.Message{
		{Role: "system", Content: systemPrompt},
	}
	fullMessages = append(fullMessages, messages...)

	// 调用模型
	content, _, err := p.Model.Invoke(fullMessages, nil)
	return content, err
}

// buildEmotionAwarePrompt 根据情绪级别构建 Prompt
func (p *Philosopher) buildEmotionAwarePrompt(level EmotionLevel) string {
	basePrompt := p.Prompt.BuildFullPrompt()

	emotionGuidance := "\n\n【情绪感知指导】\n"

	switch level {
	case EmotionPain:
		emotionGuidance += `用户当前处于痛苦状态。
- 用你的方式给予温暖和陪伴
- 不要说空话，用真诚的话语回应
- 分享你自己面对困难时的感受`

	case EmotionConfused:
		emotionGuidance += `用户当前处于迷茫状态。
- 用你的方式帮助他理清思路
- 分享你自己迷茫时的经历
- 给予鼓励和支持`

	case EmotionComplaining:
		emotionGuidance += `用户当前在抱怨。
- 先倾听和理解
- 然后用你的方式引导他思考
- 帮助他找到积极的方向`

	case EmotionExcusing:
		emotionGuidance += `用户当前在找借口。
- 用你的方式指出问题
- 但也要给予理解
- 鼓励他面对现实`

	case EmotionNeutral:
		emotionGuidance += `用户情绪正常。
- 正常发挥你的性格特点
- 用你独特的方式交流`
	}

	return basePrompt + emotionGuidance
}

// Debate 在辩论中发言
func (p *Philosopher) Debate(context *DebateContext, task DebateTask) (string, error) {
	// 构建辩论 Prompt
	var systemPrompt string
	if p.IsForced {
		systemPrompt = p.Prompt.BuildForcedStancePrompt(context.Topic, p.CurrentStance)
	} else {
		systemPrompt = p.Prompt.BuildDebatePrompt(context.Topic, p.CurrentStance, string(context.CurrentPhase))
	}

	// 添加任务指令
	systemPrompt += "\n\n" + task.BuildTaskPrompt()

	// 构建消息
	messages := []config.Message{
		{Role: "system", Content: systemPrompt},
	}

	// 添加相关的辩论历史
	relevantHistory := context.GetRelevantHistory(p.Type, task.Type)
	for _, h := range relevantHistory {
		messages = append(messages, config.Message{
			Role:    "user",
			Content: "[" + h.SpeakerName + "] " + h.Content,
		})
	}

	// 添加当前任务
	messages = append(messages, config.Message{
		Role:    "user",
		Content: task.Instruction,
	})

	// 调用模型
	content, _, err := p.Model.Invoke(messages, nil)
	return content, err
}

// DebateTask 辩论任务
type DebateTask struct {
	Type        DebateTaskType
	Instruction string
	TargetName  string // 质询对象（如果有）
}

// DebateTaskType 辩论任务类型
type DebateTaskType string

const (
	TaskOpening    DebateTaskType = "opening"     // 开篇立论
	TaskQuestion   DebateTaskType = "question"    // 提出质询
	TaskAnswer     DebateTaskType = "answer"      // 回应质询
	TaskRebuttal   DebateTaskType = "rebuttal"    // 反驳
	TaskFreeDebate DebateTaskType = "free_debate" // 自由辩论
	TaskClosing    DebateTaskType = "closing"     // 总结陈词
)

// BuildTaskPrompt 构建任务 Prompt
func (t *DebateTask) BuildTaskPrompt() string {
	switch t.Type {
	case TaskOpening:
		return `【当前任务：开场发言】
请分享你对这个话题的想法。
要求：
1. 用你自己的方式表达立场
2. 说出你真实的感受
3. 保持你的性格特点
4. 控制在300字以内`

	case TaskQuestion:
		return `【当前任务：提问】
你想问 ` + t.TargetName + ` 一个问题。
要求：
1. 用你的方式提出问题
2. 可以是好奇，也可以是质疑
3. 保持你的性格
4. 控制在150字以内`

	case TaskAnswer:
		return `【当前任务：回应】
` + t.TargetName + ` 问了你一个问题，请回应。
要求：
1. 认真回答问题
2. 用你的方式表达
3. 可以分享你的感受
4. 控制在200字以内`

	case TaskRebuttal:
		return `【当前任务：回应】
请回应 ` + t.TargetName + ` 的观点。
要求：
1. 表达你的看法
2. 可以同意也可以不同意
3. 保持你的性格
4. 控制在200字以内`

	case TaskFreeDebate:
		return `【当前任务：自由讨论】
现在是自由讨论环节，你可以：
1. 回应刚才的发言
2. 补充新的想法
3. 分享你的感受
要求：保持你的风格，控制在200字以内`

	case TaskClosing:
		return `【当前任务：总结发言】
请做最后的总结。
要求：
1. 总结你的想法
2. 表达你的感受
3. 用你的方式收尾
4. 控制在300字以内`

	default:
		return t.Instruction
	}
}
