package philosopher

import (
	"fmt"
	"sync"

	"agent/config"
)

// DebatePhase 辩论阶段
type DebatePhase string

const (
	PhaseOpening     DebatePhase = "opening"     // 开篇立论
	PhaseQuestioning DebatePhase = "questioning" // 质询交锋
	PhaseFreeDebate  DebatePhase = "free_debate" // 自由辩论
	PhaseClosing     DebatePhase = "closing"     // 总结陈词
)

// DebateConfig 辩论配置
type DebateConfig struct {
	Topic           string                     // 辩题
	ProStance       string                     // 正方立场
	ConStance       string                     // 反方立场
	ProPhilosophers []PhilosopherType          // 正方哲学家
	ConPhilosophers []PhilosopherType          // 反方哲学家
	ForcedStances   map[PhilosopherType]string // 强制立场（操纵阵营）
}

// DebateEngine 辩论流程引擎
type DebateEngine struct {
	config       *DebateConfig
	context      *DebateContext
	philosophers map[PhilosopherType]*Philosopher
	model        *config.ChatModel

	// 发言队列管理
	speakingQueue []PhilosopherType
	currentIndex  int
	mu            sync.Mutex

	// 回调函数
	onSpeech func(speaker string, content string, phase DebatePhase)
}

// DebateContext 辩论上下文（全局辩论纪要）
type DebateContext struct {
	Topic        string
	CurrentPhase DebatePhase
	History      []DebateRecord

	// 按阶段索引的历史记录
	OpeningStatements  map[PhilosopherType]string
	QuestioningRecords []QuestionRecord
	FreeDebateRecords  []DebateRecord
	ClosingStatements  map[PhilosopherType]string
}

// DebateRecord 辩论记录
type DebateRecord struct {
	Speaker       PhilosopherType `json:"speaker"`
	SpeakerName   string          `json:"speaker_name"`
	Content       string          `json:"content"`
	Phase         DebatePhase     `json:"phase"`
	TaskType      DebateTaskType  `json:"task_type"`
	TargetSpeaker PhilosopherType `json:"target_speaker"` // 如果是质询/回应，记录对象
}

// QuestionRecord 质询记录
type QuestionRecord struct {
	Questioner     PhilosopherType
	QuestionerName string
	Question       string
	Answerer       PhilosopherType
	AnswererName   string
	Answer         string
}

// NewDebateEngine 创建辩论引擎
func NewDebateEngine(cfg *DebateConfig, model *config.ChatModel) *DebateEngine {
	engine := &DebateEngine{
		config:       cfg,
		model:        model,
		philosophers: make(map[PhilosopherType]*Philosopher),
	}

	// 创建所有参与的哲学家
	allPhilosophers := append(cfg.ProPhilosophers, cfg.ConPhilosophers...)
	for _, pType := range allPhilosophers {
		p := NewPhilosopher(pType, model)

		// 设置立场
		if contains(cfg.ProPhilosophers, pType) {
			p.CurrentStance = cfg.ProStance
		} else {
			p.CurrentStance = cfg.ConStance
		}

		// 检查是否是强制立场
		if forcedStance, ok := cfg.ForcedStances[pType]; ok {
			p.CurrentStance = forcedStance
			p.IsForced = true
		}

		engine.philosophers[pType] = p
	}

	// 初始化上下文
	engine.context = &DebateContext{
		Topic:              cfg.Topic,
		CurrentPhase:       PhaseOpening,
		History:            []DebateRecord{},
		OpeningStatements:  make(map[PhilosopherType]string),
		QuestioningRecords: []QuestionRecord{},
		FreeDebateRecords:  []DebateRecord{},
		ClosingStatements:  make(map[PhilosopherType]string),
	}

	return engine
}

// SetOnSpeech 设置发言回调
func (e *DebateEngine) SetOnSpeech(callback func(speaker string, content string, phase DebatePhase)) {
	e.onSpeech = callback
}

// Run 运行完整辩论
func (e *DebateEngine) Run() (*DebateResult, error) {
	result := &DebateResult{
		Topic:   e.config.Topic,
		Records: []DebateRecord{},
	}

	// 第一幕：开篇立论
	if err := e.runOpeningPhase(); err != nil {
		return nil, fmt.Errorf("开篇立论失败: %w", err)
	}

	// 第二幕：质询交锋
	if err := e.runQuestioningPhase(); err != nil {
		return nil, fmt.Errorf("质询交锋失败: %w", err)
	}

	// 第三幕：总结陈词
	if err := e.runClosingPhase(); err != nil {
		return nil, fmt.Errorf("总结陈词失败: %w", err)
	}

	result.Records = e.context.History
	return result, nil
}

// runOpeningPhase 运行开篇立论阶段
func (e *DebateEngine) runOpeningPhase() error {
	e.context.CurrentPhase = PhaseOpening

	// 正方先发言，然后反方
	order := append(e.config.ProPhilosophers, e.config.ConPhilosophers...)

	for _, pType := range order {
		p := e.philosophers[pType]

		task := DebateTask{
			Type:        TaskOpening,
			Instruction: "请进行开篇立论",
		}

		content, err := p.Debate(e.context, task)
		if err != nil {
			return err
		}

		// 记录
		record := DebateRecord{
			Speaker:     pType,
			SpeakerName: p.Name,
			Content:     content,
			Phase:       PhaseOpening,
			TaskType:    TaskOpening,
		}
		e.context.History = append(e.context.History, record)
		e.context.OpeningStatements[pType] = content

		// 回调
		if e.onSpeech != nil {
			e.onSpeech(p.Name, content, PhaseOpening)
		}
	}

	return nil
}

// runQuestioningPhase 运行质询交锋阶段
func (e *DebateEngine) runQuestioningPhase() error {
	e.context.CurrentPhase = PhaseQuestioning

	// 交叉质询：正方质询反方，反方质询正方
	// 每个正方哲学家质询一个反方哲学家
	for i, proType := range e.config.ProPhilosophers {
		// 确定质询对象
		conIndex := i % len(e.config.ConPhilosophers)
		conType := e.config.ConPhilosophers[conIndex]

		// 正方提问
		if err := e.runQuestionExchange(proType, conType); err != nil {
			return err
		}

		// 反方反问
		if err := e.runQuestionExchange(conType, proType); err != nil {
			return err
		}
	}

	return nil
}

// runQuestionExchange 运行一次质询交换
func (e *DebateEngine) runQuestionExchange(questioner, answerer PhilosopherType) error {
	qp := e.philosophers[questioner]
	ap := e.philosophers[answerer]

	// 提问
	questionTask := DebateTask{
		Type:        TaskQuestion,
		TargetName:  ap.Name,
		Instruction: "请向 " + ap.Name + " 提出质询",
	}

	question, err := qp.Debate(e.context, questionTask)
	if err != nil {
		return err
	}

	// 记录提问
	questionRecord := DebateRecord{
		Speaker:       questioner,
		SpeakerName:   qp.Name,
		Content:       question,
		Phase:         PhaseQuestioning,
		TaskType:      TaskQuestion,
		TargetSpeaker: answerer,
	}
	e.context.History = append(e.context.History, questionRecord)

	if e.onSpeech != nil {
		e.onSpeech(qp.Name, question, PhaseQuestioning)
	}

	// 回答
	answerTask := DebateTask{
		Type:        TaskAnswer,
		TargetName:  qp.Name,
		Instruction: qp.Name + " 问你：" + question,
	}

	answer, err := ap.Debate(e.context, answerTask)
	if err != nil {
		return err
	}

	// 记录回答
	answerRecord := DebateRecord{
		Speaker:       answerer,
		SpeakerName:   ap.Name,
		Content:       answer,
		Phase:         PhaseQuestioning,
		TaskType:      TaskAnswer,
		TargetSpeaker: questioner,
	}
	e.context.History = append(e.context.History, answerRecord)

	// 记录质询对
	e.context.QuestioningRecords = append(e.context.QuestioningRecords, QuestionRecord{
		Questioner:     questioner,
		QuestionerName: qp.Name,
		Question:       question,
		Answerer:       answerer,
		AnswererName:   ap.Name,
		Answer:         answer,
	})

	if e.onSpeech != nil {
		e.onSpeech(ap.Name, answer, PhaseQuestioning)
	}

	return nil
}

// runClosingPhase 运行总结陈词阶段
func (e *DebateEngine) runClosingPhase() error {
	e.context.CurrentPhase = PhaseClosing

	// 反方先总结，正方最后
	order := append(e.config.ConPhilosophers, e.config.ProPhilosophers...)

	for _, pType := range order {
		p := e.philosophers[pType]

		task := DebateTask{
			Type:        TaskClosing,
			Instruction: "请进行总结陈词",
		}

		content, err := p.Debate(e.context, task)
		if err != nil {
			return err
		}

		// 记录
		record := DebateRecord{
			Speaker:     pType,
			SpeakerName: p.Name,
			Content:     content,
			Phase:       PhaseClosing,
			TaskType:    TaskClosing,
		}
		e.context.History = append(e.context.History, record)
		e.context.ClosingStatements[pType] = content

		if e.onSpeech != nil {
			e.onSpeech(p.Name, content, PhaseClosing)
		}
	}

	return nil
}

// GetRelevantHistory 获取与当前任务相关的历史记录
// 这是"动态上下文构建"的核心实现
func (c *DebateContext) GetRelevantHistory(speaker PhilosopherType, taskType DebateTaskType) []DebateRecord {
	var relevant []DebateRecord

	switch taskType {
	case TaskOpening:
		// 开篇立论：只需要知道辩题，不需要历史
		return relevant

	case TaskQuestion:
		// 质询：需要知道对方的开篇立论
		for pType, statement := range c.OpeningStatements {
			if pType != speaker {
				relevant = append(relevant, DebateRecord{
					Speaker:     pType,
					SpeakerName: GetPhilosopherPrompts()[pType].Name,
					Content:     statement,
					Phase:       PhaseOpening,
				})
			}
		}

	case TaskAnswer:
		// 回应质询：需要知道自己的立论 + 刚才的质询
		if statement, ok := c.OpeningStatements[speaker]; ok {
			relevant = append(relevant, DebateRecord{
				Speaker:     speaker,
				SpeakerName: GetPhilosopherPrompts()[speaker].Name,
				Content:     statement,
				Phase:       PhaseOpening,
			})
		}
		// 最近的质询记录
		if len(c.History) > 0 {
			lastRecord := c.History[len(c.History)-1]
			if lastRecord.TaskType == TaskQuestion {
				relevant = append(relevant, lastRecord)
			}
		}

	case TaskClosing:
		// 总结陈词：需要关键的交锋记录
		// 1. 自己的开篇立论
		if statement, ok := c.OpeningStatements[speaker]; ok {
			relevant = append(relevant, DebateRecord{
				Speaker:     speaker,
				SpeakerName: GetPhilosopherPrompts()[speaker].Name,
				Content:     statement,
				Phase:       PhaseOpening,
			})
		}
		// 2. 与自己相关的质询记录
		for _, qr := range c.QuestioningRecords {
			if qr.Questioner == speaker || qr.Answerer == speaker {
				relevant = append(relevant, DebateRecord{
					Speaker:     qr.Questioner,
					SpeakerName: qr.QuestionerName,
					Content:     qr.Question,
					Phase:       PhaseQuestioning,
				})
				relevant = append(relevant, DebateRecord{
					Speaker:     qr.Answerer,
					SpeakerName: qr.AnswererName,
					Content:     qr.Answer,
					Phase:       PhaseQuestioning,
				})
			}
		}

	case TaskFreeDebate:
		// 自由辩论：最近3条记录
		start := len(c.History) - 3
		if start < 0 {
			start = 0
		}
		relevant = c.History[start:]
	}

	return relevant
}

// DebateResult 辩论结果
type DebateResult struct {
	Topic   string
	Records []DebateRecord
}

// helper function
func contains(slice []PhilosopherType, item PhilosopherType) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
