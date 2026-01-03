package philosopher

import (
	"regexp"
	"strings"

	"agent/config"
)

// EmotionLevel 情绪级别
type EmotionLevel string

const (
	EmotionPain        EmotionLevel = "pain"        // 痛苦
	EmotionConfused    EmotionLevel = "confused"    // 迷茫
	EmotionComplaining EmotionLevel = "complaining" // 抱怨
	EmotionExcusing    EmotionLevel = "excusing"    // 找借口
	EmotionNeutral     EmotionLevel = "neutral"     // 正常
)

// EmotionAnalyzer 情绪分析器
type EmotionAnalyzer struct {
	model *config.ChatModel

	// 关键词规则（快速判断）
	painKeywords        []string
	confusedKeywords    []string
	complainingKeywords []string
	excusingKeywords    []string
}

// NewEmotionAnalyzer 创建情绪分析器
func NewEmotionAnalyzer(model *config.ChatModel) *EmotionAnalyzer {
	return &EmotionAnalyzer{
		model: model,
		painKeywords: []string{
			"好痛苦", "受不了", "活不下去", "想死", "崩溃", "绝望",
			"太难了", "撑不住", "心碎", "无法承受", "痛不欲生",
			"失去了一切", "没有意义", "好累", "精疲力竭",
		},
		confusedKeywords: []string{
			"不知道", "迷茫", "困惑", "该怎么办", "怎么选",
			"纠结", "犹豫", "不确定", "找不到方向", "人生意义",
			"不明白", "为什么", "想不通",
		},
		complainingKeywords: []string{
			"凭什么", "不公平", "太过分", "受够了", "烦死了",
			"讨厌", "恶心", "垃圾", "傻逼", "操",
			"都怪", "要不是", "本来可以",
		},
		excusingKeywords: []string{
			"没办法", "不得不", "被迫", "没有选择", "环境所迫",
			"别人都", "大家都这样", "我也想但是", "条件不允许",
			"不是我的错", "我尽力了", "我没有能力",
		},
	}
}

// Analyze 分析用户输入的情绪
func (a *EmotionAnalyzer) Analyze(text string) EmotionLevel {
	// 第一层：关键词快速判断
	level := a.quickAnalyze(text)
	if level != EmotionNeutral {
		return level
	}

	// 如果关键词无法判断，使用 AI 分析
	return a.aiAnalyze(text)
}

// quickAnalyze 快速关键词分析
func (a *EmotionAnalyzer) quickAnalyze(text string) EmotionLevel {
	text = strings.ToLower(text)

	// 检查痛苦关键词
	painScore := a.countKeywords(text, a.painKeywords)
	if painScore >= 2 || a.containsStrongPainIndicator(text) {
		return EmotionPain
	}

	// 检查找借口关键词
	excusingScore := a.countKeywords(text, a.excusingKeywords)
	if excusingScore >= 2 {
		return EmotionExcusing
	}

	// 检查抱怨关键词
	complainingScore := a.countKeywords(text, a.complainingKeywords)
	if complainingScore >= 2 {
		return EmotionComplaining
	}

	// 检查迷茫关键词
	confusedScore := a.countKeywords(text, a.confusedKeywords)
	if confusedScore >= 2 {
		return EmotionConfused
	}

	return EmotionNeutral
}

// countKeywords 计算关键词命中数
func (a *EmotionAnalyzer) countKeywords(text string, keywords []string) int {
	count := 0
	for _, kw := range keywords {
		if strings.Contains(text, kw) {
			count++
		}
	}
	return count
}

// containsStrongPainIndicator 检查是否包含强烈痛苦指标
func (a *EmotionAnalyzer) containsStrongPainIndicator(text string) bool {
	strongIndicators := []string{
		"想死", "自杀", "活不下去", "没有意义",
	}
	for _, indicator := range strongIndicators {
		if strings.Contains(text, indicator) {
			return true
		}
	}
	return false
}

// aiAnalyze 使用 AI 进行深度情绪分析
func (a *EmotionAnalyzer) aiAnalyze(text string) EmotionLevel {
	if a.model == nil {
		return EmotionNeutral
	}

	prompt := `你是一个情绪分析专家。请分析以下用户输入的情绪状态。

用户输入：
"""
` + text + `
"""

请判断用户当前的情绪状态，只输出以下五个选项之一：
- pain（痛苦：用户正在经历真实的痛苦、悲伤、绝望）
- confused（迷茫：用户感到困惑、不知所措、需要方向）
- complaining（抱怨：用户在发泄不满、抱怨他人或环境）
- excusing（找借口：用户在为自己的行为或不作为找借口）
- neutral（正常：用户情绪正常，在进行普通的讨论或提问）

只输出一个词，不要有任何其他内容。`

	messages := []config.Message{
		{Role: "user", Content: prompt},
	}

	response, _, err := a.model.Invoke(messages, nil)
	if err != nil {
		return EmotionNeutral
	}

	// 解析响应
	response = strings.TrimSpace(strings.ToLower(response))

	switch {
	case strings.Contains(response, "pain"):
		return EmotionPain
	case strings.Contains(response, "confused"):
		return EmotionConfused
	case strings.Contains(response, "complaining"):
		return EmotionComplaining
	case strings.Contains(response, "excusing"):
		return EmotionExcusing
	default:
		return EmotionNeutral
	}
}

// EmotionAwareResponse 情绪感知响应配置
type EmotionAwareResponse struct {
	ToxicityLevel float64 // 毒舌程度 0-1
	EmpathyLevel  float64 // 共情程度 0-1
	Guidance      string  // 额外指导
}

// GetResponseConfig 根据情绪级别获取响应配置
func GetResponseConfig(level EmotionLevel) *EmotionAwareResponse {
	switch level {
	case EmotionPain:
		return &EmotionAwareResponse{
			ToxicityLevel: 0.1,
			EmpathyLevel:  0.9,
			Guidance: `用户正在经历痛苦。
- 暂时收起毒舌，展现智慧和温度
- 不要说空话和鸡汤，要说有深度的话
- 帮助用户从更高的视角看待处境
- 可以分享哲学家面对苦难的智慧`,
		}

	case EmotionConfused:
		return &EmotionAwareResponse{
			ToxicityLevel: 0.4,
			EmpathyLevel:  0.6,
			Guidance: `用户感到迷茫。
- 用你的方法论帮助他理清思路
- 可以适度毒舌，但重点是引导思考
- 提出有建设性的问题`,
		}

	case EmotionComplaining:
		return &EmotionAwareResponse{
			ToxicityLevel: 0.7,
			EmpathyLevel:  0.3,
			Guidance: `用户在抱怨。
- 可以适度毒舌，指出抱怨的无用
- 引导他思考：抱怨能改变什么？
- 挑战他采取行动`,
		}

	case EmotionExcusing:
		return &EmotionAwareResponse{
			ToxicityLevel: 0.9,
			EmpathyLevel:  0.1,
			Guidance: `用户在找借口。
- 毫不留情地戳破借口
- 用最犀利的语言揭示自欺欺人
- 逼迫他面对真相`,
		}

	default:
		return &EmotionAwareResponse{
			ToxicityLevel: 0.6,
			EmpathyLevel:  0.4,
			Guidance: `用户情绪正常。
- 正常发挥毒舌风格
- 该犀利时犀利，该深刻时深刻`,
		}
	}
}

// ContentDeduplicator 内容去重器
// 使用 Jaccard 相似度检测重复内容
type ContentDeduplicator struct {
	previousResponses []string
	threshold         float64 // 相似度阈值，超过则认为重复
}

// NewContentDeduplicator 创建内容去重器
func NewContentDeduplicator(threshold float64) *ContentDeduplicator {
	return &ContentDeduplicator{
		previousResponses: []string{},
		threshold:         threshold,
	}
}

// IsDuplicate 检查内容是否与之前的响应重复
func (d *ContentDeduplicator) IsDuplicate(content string) bool {
	for _, prev := range d.previousResponses {
		similarity := d.jaccardSimilarity(content, prev)
		if similarity > d.threshold {
			return true
		}
	}
	return false
}

// AddResponse 添加响应到历史
func (d *ContentDeduplicator) AddResponse(content string) {
	d.previousResponses = append(d.previousResponses, content)
	// 只保留最近10条
	if len(d.previousResponses) > 10 {
		d.previousResponses = d.previousResponses[1:]
	}
}

// jaccardSimilarity 计算 Jaccard 相似度
func (d *ContentDeduplicator) jaccardSimilarity(a, b string) float64 {
	// 分词
	wordsA := d.tokenize(a)
	wordsB := d.tokenize(b)

	// 计算交集和并集
	setA := make(map[string]bool)
	for _, w := range wordsA {
		setA[w] = true
	}

	setB := make(map[string]bool)
	for _, w := range wordsB {
		setB[w] = true
	}

	intersection := 0
	for w := range setA {
		if setB[w] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// tokenize 简单分词
func (d *ContentDeduplicator) tokenize(text string) []string {
	// 移除标点符号
	reg := regexp.MustCompile(`[^\p{L}\p{N}\s]`)
	text = reg.ReplaceAllString(text, " ")

	// 分割
	words := strings.Fields(text)

	// 过滤短词
	var result []string
	for _, w := range words {
		if len(w) >= 2 {
			result = append(result, w)
		}
	}

	return result
}
