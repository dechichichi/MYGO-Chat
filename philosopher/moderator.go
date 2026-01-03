package philosopher

import (
	"fmt"
	"strings"

	"agent/config"
)

// ==================== 主持人 Agent（自主驱动讨论）====================

// ModeratorAgent 主持人 Agent，负责自主驱动讨论流程
type ModeratorAgent struct {
	model      *config.ChatModel
	context    *DebateContext
	members    map[PhilosopherType]*Philosopher
	roundCount int
	maxRounds  int
	onDecision func(decision *ModeratorDecision) // 决策回调
}

// ModeratorDecision 主持人的决策
type ModeratorDecision struct {
	Action       ModeratorAction `json:"action"`        // 决策动作
	NextSpeaker  PhilosopherType `json:"next_speaker"`  // 下一个发言者
	TargetMember PhilosopherType `json:"target_member"` // 目标成员（如果是质询）
	Instruction  string          `json:"instruction"`   // 给发言者的指令
	Reason       string          `json:"reason"`        // 决策理由
	ShouldEnd    bool            `json:"should_end"`    // 是否应该结束讨论
	Phase        DebatePhase     `json:"phase"`         // 当前阶段
}

// ModeratorAction 主持人动作类型
type ModeratorAction string

const (
	ActionOpeningSpeech  ModeratorAction = "opening_speech"  // 开场发言
	ActionAskQuestion    ModeratorAction = "ask_question"    // 让某人提问
	ActionRequestAnswer  ModeratorAction = "request_answer"  // 让某人回答
	ActionInviteComment  ModeratorAction = "invite_comment"  // 邀请评论
	ActionFreeDiscussion ModeratorAction = "free_discussion" // 自由讨论
	ActionRequestSummary ModeratorAction = "request_summary" // 请求总结
	ActionEndDiscussion  ModeratorAction = "end_discussion"  // 结束讨论
)

// NewModeratorAgent 创建主持人 Agent
func NewModeratorAgent(model *config.ChatModel, topic string, members map[PhilosopherType]*Philosopher) *ModeratorAgent {
	return &ModeratorAgent{
		model:     model,
		members:   members,
		maxRounds: 10, // 默认最多10轮
		context: &DebateContext{
			Topic:              topic,
			CurrentPhase:       PhaseOpening,
			History:            []DebateRecord{},
			OpeningStatements:  make(map[PhilosopherType]string),
			QuestioningRecords: []QuestionRecord{},
			ClosingStatements:  make(map[PhilosopherType]string),
		},
	}
}

// SetOnDecision 设置决策回调
func (m *ModeratorAgent) SetOnDecision(callback func(decision *ModeratorDecision)) {
	m.onDecision = callback
}

// SetMaxRounds 设置最大轮数
func (m *ModeratorAgent) SetMaxRounds(rounds int) {
	m.maxRounds = rounds
}

// Think 主持人思考下一步决策
func (m *ModeratorAgent) Think() (*ModeratorDecision, error) {
	// 构建主持人的思考 Prompt
	systemPrompt := m.buildModeratorPrompt()

	// 构建当前状态描述
	stateDesc := m.buildStateDescription()

	messages := []config.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: stateDesc},
	}

	// 调用模型进行决策
	response, _, err := m.model.Invoke(messages, nil)
	if err != nil {
		return nil, fmt.Errorf("主持人思考失败: %w", err)
	}

	// 解析决策
	decision := m.parseDecision(response)

	return decision, nil
}

// buildModeratorPrompt 构建主持人 Prompt
func (m *ModeratorAgent) buildModeratorPrompt() string {
	memberNames := []string{}
	for _, member := range m.members {
		memberNames = append(memberNames, member.Name)
	}

	return fmt.Sprintf(`你是一个讨论会的主持人，负责引导 MyGO!!!!! 乐队成员进行话题讨论。

【参与成员】
%s

【你的职责】
1. 决定谁下一个发言
2. 决定发言的类型（开场、提问、回答、评论、总结）
3. 判断讨论是否应该继续或结束
4. 确保每个成员都有发言机会
5. 在合适的时机推进讨论阶段

【讨论阶段】
1. opening（开场）：每个成员表达自己的初步想法
2. questioning（质询）：成员之间相互提问和回答
3. closing（总结）：每个成员做最后总结

【决策格式】
请用以下格式输出你的决策：
ACTION: [动作类型]
SPEAKER: [发言者代号]
TARGET: [目标成员代号，如果有的话]
INSTRUCTION: [给发言者的具体指令]
REASON: [你做出这个决策的理由]
SHOULD_END: [true/false]
PHASE: [当前阶段]

【动作类型】
- opening_speech: 开场发言
- ask_question: 让某人向另一人提问
- request_answer: 让某人回答问题
- invite_comment: 邀请某人评论
- free_discussion: 自由讨论
- request_summary: 请求总结发言
- end_discussion: 结束讨论

【成员代号】
- tomori: 高松灯
- anon: 千早爱音
- rana: 要乐奈
- soyo: 长崎素世
- taki: 椎名立希`, strings.Join(memberNames, "、"))
}

// buildStateDescription 构建当前状态描述
func (m *ModeratorAgent) buildStateDescription() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("【讨论话题】\n%s\n\n", m.context.Topic))
	sb.WriteString(fmt.Sprintf("【当前阶段】%s\n", m.context.CurrentPhase))
	sb.WriteString(fmt.Sprintf("【已进行轮数】%d / %d\n\n", m.roundCount, m.maxRounds))

	// 已发言情况
	sb.WriteString("【发言记录】\n")
	if len(m.context.History) == 0 {
		sb.WriteString("还没有人发言\n")
	} else {
		// 只显示最近5条
		start := 0
		if len(m.context.History) > 5 {
			start = len(m.context.History) - 5
			sb.WriteString(fmt.Sprintf("... 省略前 %d 条记录 ...\n", start))
		}
		for _, h := range m.context.History[start:] {
			content := h.Content
			if len(content) > 100 {
				content = content[:100] + "..."
			}
			sb.WriteString(fmt.Sprintf("- [%s][%s] %s\n", h.Phase, h.SpeakerName, content))
		}
	}

	// 开场发言情况
	sb.WriteString("\n【开场发言完成情况】\n")
	for pType, member := range m.members {
		if _, ok := m.context.OpeningStatements[pType]; ok {
			sb.WriteString(fmt.Sprintf("✓ %s 已完成开场\n", member.Name))
		} else {
			sb.WriteString(fmt.Sprintf("○ %s 尚未开场\n", member.Name))
		}
	}

	// 总结发言情况
	if m.context.CurrentPhase == PhaseClosing {
		sb.WriteString("\n【总结发言完成情况】\n")
		for pType, member := range m.members {
			if _, ok := m.context.ClosingStatements[pType]; ok {
				sb.WriteString(fmt.Sprintf("✓ %s 已完成总结\n", member.Name))
			} else {
				sb.WriteString(fmt.Sprintf("○ %s 尚未总结\n", member.Name))
			}
		}
	}

	sb.WriteString("\n请根据以上信息，决定下一步应该怎么做。")

	return sb.String()
}

// parseDecision 解析主持人的决策
func (m *ModeratorAgent) parseDecision(response string) *ModeratorDecision {
	decision := &ModeratorDecision{
		Phase: m.context.CurrentPhase,
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ACTION:") {
			action := strings.TrimSpace(strings.TrimPrefix(line, "ACTION:"))
			decision.Action = ModeratorAction(action)
		} else if strings.HasPrefix(line, "SPEAKER:") {
			speaker := strings.TrimSpace(strings.TrimPrefix(line, "SPEAKER:"))
			decision.NextSpeaker = PhilosopherType(speaker)
		} else if strings.HasPrefix(line, "TARGET:") {
			target := strings.TrimSpace(strings.TrimPrefix(line, "TARGET:"))
			if target != "" && target != "none" && target != "无" {
				decision.TargetMember = PhilosopherType(target)
			}
		} else if strings.HasPrefix(line, "INSTRUCTION:") {
			decision.Instruction = strings.TrimSpace(strings.TrimPrefix(line, "INSTRUCTION:"))
		} else if strings.HasPrefix(line, "REASON:") {
			decision.Reason = strings.TrimSpace(strings.TrimPrefix(line, "REASON:"))
		} else if strings.HasPrefix(line, "SHOULD_END:") {
			shouldEnd := strings.TrimSpace(strings.TrimPrefix(line, "SHOULD_END:"))
			decision.ShouldEnd = shouldEnd == "true"
		} else if strings.HasPrefix(line, "PHASE:") {
			phase := strings.TrimSpace(strings.TrimPrefix(line, "PHASE:"))
			decision.Phase = DebatePhase(phase)
		}
	}

	// 默认值处理
	if decision.Action == "" {
		decision.Action = ActionFreeDiscussion
	}
	if decision.Instruction == "" {
		decision.Instruction = "请分享你的想法"
	}

	return decision
}

// Execute 执行决策，让对应成员发言
func (m *ModeratorAgent) Execute(decision *ModeratorDecision) (*DebateRecord, error) {
	// 检查是否结束
	if decision.ShouldEnd || decision.Action == ActionEndDiscussion {
		return nil, nil
	}

	// 更新阶段
	m.context.CurrentPhase = decision.Phase

	// 获取发言者
	speaker, ok := m.members[decision.NextSpeaker]
	if !ok {
		return nil, fmt.Errorf("未找到成员: %s", decision.NextSpeaker)
	}

	// 构建任务
	task := m.buildTask(decision)

	// 让成员发言
	content, err := speaker.Debate(m.context, task)
	if err != nil {
		return nil, fmt.Errorf("%s 发言失败: %w", speaker.Name, err)
	}

	// 记录发言
	record := &DebateRecord{
		Speaker:     decision.NextSpeaker,
		SpeakerName: speaker.Name,
		Content:     content,
		Phase:       decision.Phase,
		TaskType:    task.Type,
	}

	if decision.TargetMember != "" {
		record.TargetSpeaker = decision.TargetMember
	}

	// 更新上下文
	m.context.History = append(m.context.History, *record)
	m.roundCount++

	// 更新特定阶段的记录
	switch decision.Action {
	case ActionOpeningSpeech:
		m.context.OpeningStatements[decision.NextSpeaker] = content
	case ActionRequestSummary:
		m.context.ClosingStatements[decision.NextSpeaker] = content
	}

	return record, nil
}

// buildTask 根据决策构建任务
func (m *ModeratorAgent) buildTask(decision *ModeratorDecision) DebateTask {
	task := DebateTask{
		Instruction: decision.Instruction,
	}

	switch decision.Action {
	case ActionOpeningSpeech:
		task.Type = TaskOpening
	case ActionAskQuestion:
		task.Type = TaskQuestion
		if decision.TargetMember != "" {
			if target, ok := m.members[decision.TargetMember]; ok {
				task.TargetName = target.Name
			}
		}
	case ActionRequestAnswer:
		task.Type = TaskAnswer
		if decision.TargetMember != "" {
			if target, ok := m.members[decision.TargetMember]; ok {
				task.TargetName = target.Name
			}
		}
	case ActionInviteComment, ActionFreeDiscussion:
		task.Type = TaskFreeDebate
	case ActionRequestSummary:
		task.Type = TaskClosing
	default:
		task.Type = TaskFreeDebate
	}

	return task
}

// RunAutonomous 自主运行完整讨论
func (m *ModeratorAgent) RunAutonomous(onSpeech func(speaker string, content string, phase DebatePhase)) (*DebateResult, error) {
	result := &DebateResult{
		Topic:   m.context.Topic,
		Records: []DebateRecord{},
	}

	for m.roundCount < m.maxRounds {
		// 主持人思考
		decision, err := m.Think()
		if err != nil {
			return nil, err
		}

		// 回调决策
		if m.onDecision != nil {
			m.onDecision(decision)
		}

		// 检查是否结束
		if decision.ShouldEnd {
			break
		}

		// 执行决策
		record, err := m.Execute(decision)
		if err != nil {
			return nil, err
		}

		if record != nil {
			result.Records = append(result.Records, *record)
			if onSpeech != nil {
				onSpeech(record.SpeakerName, record.Content, record.Phase)
			}
		}
	}

	return result, nil
}

// GetContext 获取当前上下文
func (m *ModeratorAgent) GetContext() *DebateContext {
	return m.context
}
