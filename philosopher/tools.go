package philosopher

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ==================== Agent 工具系统 ====================

// Tool Agent 可调用的工具
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Handler     ToolHandler            `json:"-"`
}

// ToolHandler 工具处理函数
type ToolHandler func(args map[string]interface{}, ctx *AgentContext) (string, error)

// ToolCall 工具调用
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// AgentContext Agent 上下文，包含记忆、状态等
type AgentContext struct {
	PhilosopherType PhilosopherType
	PhilosopherName string
	UserID          string
	CurrentMood     string // 当前对话氛围

	// 记忆系统
	ShortTermMemory []MemoryItem // 短期记忆（当前会话）
	LongTermMemory  []MemoryItem // 长期记忆（跨会话）

	// 对话历史摘要
	ConversationSummary string
}

// MemoryItem 记忆条目
type MemoryItem struct {
	Timestamp  time.Time `json:"timestamp"`
	Type       string    `json:"type"` // "fact", "emotion", "preference", "event"
	Content    string    `json:"content"`
	Importance float64   `json:"importance"` // 0-1，重要程度
	RelatedTo  string    `json:"related_to"` // 关联的话题或人
}

// ==================== 工具定义 ====================

// GetAgentTools 获取 Agent 可用的工具列表
func GetAgentTools() []Tool {
	return []Tool{
		{
			Name:        "recall_memory",
			Description: "回忆与用户的过往对话和重要信息。当你想要提及之前聊过的内容，或者想了解用户的喜好、经历时使用。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "想要回忆的内容关键词，比如'上次聊的话题'、'用户的烦恼'、'喜欢的事物'",
					},
					"memory_type": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"all", "fact", "emotion", "preference", "event"},
						"description": "记忆类型：all-全部，fact-事实，emotion-情感，preference-偏好，event-事件",
					},
				},
				"required": []string{"query"},
			},
			Handler: handleRecallMemory,
		},
		{
			Name:        "save_memory",
			Description: "保存重要的信息到记忆中。当用户分享了重要的事情、表达了偏好、或者发生了值得记住的对话时使用。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "要记住的内容",
					},
					"memory_type": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"fact", "emotion", "preference", "event"},
						"description": "记忆类型",
					},
					"importance": map[string]interface{}{
						"type":        "number",
						"description": "重要程度 0-1，1 最重要",
					},
				},
				"required": []string{"content", "memory_type"},
			},
			Handler: handleSaveMemory,
		},
		{
			Name:        "search_lyrics",
			Description: "搜索歌词找灵感。当你想引用歌词、或者想用音乐来表达情感时使用。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"mood": map[string]interface{}{
						"type":        "string",
						"description": "想要表达的情绪或氛围，比如'迷茫'、'希望'、'悲伤'、'勇气'",
					},
					"theme": map[string]interface{}{
						"type":        "string",
						"description": "主题关键词，比如'前进'、'友情'、'梦想'",
					},
				},
				"required": []string{"mood"},
			},
			Handler: handleSearchLyrics,
		},
		{
			Name:        "sense_atmosphere",
			Description: "感知当前的对话氛围。当你想要更好地理解当前对话的情绪基调，以便做出更恰当的回应时使用。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"aspect": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"emotion", "topic", "relationship", "all"},
						"description": "想感知的方面：emotion-情绪，topic-话题走向，relationship-关系状态，all-全部",
					},
				},
				"required": []string{"aspect"},
			},
			Handler: handleSenseAtmosphere,
		},
		{
			Name:        "reflect_response",
			Description: "反思自己即将给出的回复。在给出重要回复前，检查是否符合自己的性格，是否恰当。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"draft_response": map[string]interface{}{
						"type":        "string",
						"description": "准备给出的回复草稿",
					},
					"check_aspects": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "要检查的方面，如 ['性格一致性', '情感恰当性', '表达方式']",
					},
				},
				"required": []string{"draft_response"},
			},
			Handler: handleReflectResponse,
		},
	}
}

// ==================== 工具处理函数 ====================

// handleRecallMemory 处理记忆回忆
func handleRecallMemory(args map[string]interface{}, ctx *AgentContext) (string, error) {
	query, _ := args["query"].(string)
	memoryType, _ := args["memory_type"].(string)
	if memoryType == "" {
		memoryType = "all"
	}

	var results []MemoryItem

	// 搜索短期记忆
	for _, m := range ctx.ShortTermMemory {
		if memoryType != "all" && m.Type != memoryType {
			continue
		}
		if strings.Contains(strings.ToLower(m.Content), strings.ToLower(query)) {
			results = append(results, m)
		}
	}

	// 搜索长期记忆
	for _, m := range ctx.LongTermMemory {
		if memoryType != "all" && m.Type != memoryType {
			continue
		}
		if strings.Contains(strings.ToLower(m.Content), strings.ToLower(query)) {
			results = append(results, m)
		}
	}

	if len(results) == 0 {
		return "没有找到相关的记忆。这可能是我们第一次聊到这个话题。", nil
	}

	// 构建回忆结果
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("找到 %d 条相关记忆：\n", len(results)))
	for i, m := range results {
		if i >= 5 { // 最多返回5条
			break
		}
		sb.WriteString(fmt.Sprintf("- [%s] %s (重要度: %.1f)\n",
			m.Type, m.Content, m.Importance))
	}

	return sb.String(), nil
}

// handleSaveMemory 处理记忆保存
func handleSaveMemory(args map[string]interface{}, ctx *AgentContext) (string, error) {
	content, _ := args["content"].(string)
	memoryType, _ := args["memory_type"].(string)
	importance, _ := args["importance"].(float64)
	if importance == 0 {
		importance = 0.5
	}

	memory := MemoryItem{
		Timestamp:  time.Now(),
		Type:       memoryType,
		Content:    content,
		Importance: importance,
	}

	// 高重要度的存入长期记忆
	if importance >= 0.7 {
		ctx.LongTermMemory = append(ctx.LongTermMemory, memory)
	}
	// 所有记忆都存入短期记忆
	ctx.ShortTermMemory = append(ctx.ShortTermMemory, memory)

	return fmt.Sprintf("已记住：%s（类型：%s，重要度：%.1f）", content, memoryType, importance), nil
}

// handleSearchLyrics 处理歌词搜索
func handleSearchLyrics(args map[string]interface{}, ctx *AgentContext) (string, error) {
	mood, _ := args["mood"].(string)
	theme, _ := args["theme"].(string)

	// MyGO!!!!! 歌词库
	lyrics := map[string][]struct {
		Song   string
		Lyrics string
		Mood   []string
		Theme  []string
	}{
		"迷路": {
			{Song: "迷路日々", Lyrics: "迷子でもいい、迷子でも進め（迷路也没关系，迷路也要前进）", Mood: []string{"迷茫", "希望", "勇气"}, Theme: []string{"前进", "迷茫", "成长"}},
			{Song: "迷路日々", Lyrics: "どこにいるかわからなくても、歩き続ける（即使不知道身在何处，也要继续走下去）", Mood: []string{"迷茫", "坚定"}, Theme: []string{"前进", "坚持"}},
		},
		"羁绊": {
			{Song: "春日影", Lyrics: "春日影に抱かれて、君と歩いた道（被春日的影子拥抱，与你一起走过的路）", Mood: []string{"温暖", "怀念", "悲伤"}, Theme: []string{"回忆", "友情", "离别"}},
			{Song: "碧天伴走", Lyrics: "碧い空の下、一緒に歩こう（在碧蓝的天空下，一起走吧）", Mood: []string{"希望", "温暖"}, Theme: []string{"友情", "陪伴", "前进"}},
		},
		"音乐": {
			{Song: "音一会", Lyrics: "音が繋ぐ、この一瞬を（用音乐连接，这一瞬间）", Mood: []string{"激动", "感动"}, Theme: []string{"音乐", "连接", "瞬间"}},
			{Song: "影色舞", Lyrics: "影と光が踊る、この舞台で（在这个舞台上，影与光共舞）", Mood: []string{"激情", "期待"}, Theme: []string{"舞台", "表演", "梦想"}},
		},
		"孤独": {
			{Song: "栞", Lyrics: "一人じゃないって、知ってるはずなのに（明明知道不是一个人）", Mood: []string{"孤独", "悲伤", "渴望"}, Theme: []string{"孤独", "陪伴", "理解"}},
		},
		"梦想": {
			{Song: "詩超絆", Lyrics: "言葉を超えて、想いを届けたい（想要超越语言，传达心意）", Mood: []string{"渴望", "热情"}, Theme: []string{"表达", "连接", "梦想"}},
		},
	}

	var results []string

	// 根据 mood 和 theme 搜索
	for _, category := range lyrics {
		for _, l := range category {
			matched := false
			for _, m := range l.Mood {
				if strings.Contains(m, mood) || strings.Contains(mood, m) {
					matched = true
					break
				}
			}
			if theme != "" && !matched {
				for _, t := range l.Theme {
					if strings.Contains(t, theme) || strings.Contains(theme, t) {
						matched = true
						break
					}
				}
			}
			if matched {
				results = append(results, fmt.Sprintf("「%s」- %s", l.Song, l.Lyrics))
			}
		}
	}

	if len(results) == 0 {
		return "没有找到完全匹配的歌词，但音乐的力量在于它能表达无法言说的情感。", nil
	}

	var sb strings.Builder
	sb.WriteString("找到以下歌词可以表达这种感觉：\n")
	for i, r := range results {
		if i >= 3 {
			break
		}
		sb.WriteString("♪ " + r + "\n")
	}

	return sb.String(), nil
}

// handleSenseAtmosphere 处理氛围感知
func handleSenseAtmosphere(args map[string]interface{}, ctx *AgentContext) (string, error) {
	aspect, _ := args["aspect"].(string)

	var result strings.Builder
	result.WriteString("【氛围感知结果】\n")

	switch aspect {
	case "emotion":
		result.WriteString(fmt.Sprintf("当前情绪基调：%s\n", ctx.CurrentMood))
		result.WriteString("建议：根据用户的情绪状态调整回应的温度和方式。")

	case "topic":
		// 从短期记忆中分析话题走向
		topics := analyzeTopic(ctx.ShortTermMemory)
		result.WriteString(fmt.Sprintf("话题走向：%s\n", topics))
		result.WriteString("建议：可以顺着当前话题深入，或者适时引导到新的方向。")

	case "relationship":
		// 分析关系状态
		relationshipLevel := analyzeRelationship(ctx)
		result.WriteString(fmt.Sprintf("关系状态：%s\n", relationshipLevel))
		result.WriteString("建议：根据关系亲密度调整说话方式。")

	case "all":
		result.WriteString(fmt.Sprintf("情绪基调：%s\n", ctx.CurrentMood))
		result.WriteString(fmt.Sprintf("话题走向：%s\n", analyzeTopic(ctx.ShortTermMemory)))
		result.WriteString(fmt.Sprintf("关系状态：%s\n", analyzeRelationship(ctx)))
	}

	return result.String(), nil
}

// handleReflectResponse 处理回复反思
func handleReflectResponse(args map[string]interface{}, ctx *AgentContext) (string, error) {
	draftResponse, _ := args["draft_response"].(string)
	checkAspects, _ := args["check_aspects"].([]interface{})

	var result strings.Builder
	result.WriteString("【回复反思】\n")
	result.WriteString(fmt.Sprintf("草稿：%s\n\n", draftResponse))

	// 默认检查项
	if len(checkAspects) == 0 {
		checkAspects = []interface{}{"性格一致性", "情感恰当性", "表达方式"}
	}

	for _, aspect := range checkAspects {
		aspectStr, _ := aspect.(string)
		switch aspectStr {
		case "性格一致性":
			result.WriteString("✓ 性格一致性：检查回复是否符合角色设定\n")
		case "情感恰当性":
			result.WriteString(fmt.Sprintf("✓ 情感恰当性：当前氛围是「%s」，检查回复的情感温度是否合适\n", ctx.CurrentMood))
		case "表达方式":
			result.WriteString("✓ 表达方式：检查用词和句式是否符合角色的说话习惯\n")
		}
	}

	result.WriteString("\n建议：如果有不符合的地方，请调整后再回复。")

	return result.String(), nil
}

// ==================== 辅助函数 ====================

func analyzeTopic(memories []MemoryItem) string {
	if len(memories) == 0 {
		return "刚开始对话，话题尚未展开"
	}

	// 简单分析最近的记忆
	recentTopics := []string{}
	for i := len(memories) - 1; i >= 0 && i >= len(memories)-3; i-- {
		recentTopics = append(recentTopics, memories[i].Content)
	}

	if len(recentTopics) > 0 {
		return fmt.Sprintf("最近在聊：%s", strings.Join(recentTopics, " → "))
	}
	return "话题比较分散"
}

func analyzeRelationship(ctx *AgentContext) string {
	// 根据长期记忆数量和互动深度判断关系
	memoryCount := len(ctx.LongTermMemory)

	if memoryCount == 0 {
		return "初次见面，还在相互了解"
	} else if memoryCount < 5 {
		return "有过几次交流，开始熟悉"
	} else if memoryCount < 15 {
		return "已经比较熟悉，可以聊一些深入的话题"
	} else {
		return "老朋友了，可以很自然地交流"
	}
}

// ==================== 工具转换为 OpenAI 格式 ====================

// ToOpenAITools 将工具转换为 OpenAI API 格式
func ToOpenAITools(tools []Tool) []map[string]interface{} {
	result := make([]map[string]interface{}, len(tools))
	for i, t := range tools {
		result[i] = map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.Parameters,
			},
		}
	}
	return result
}

// ExecuteTool 执行工具调用
func ExecuteTool(toolCall ToolCall, ctx *AgentContext) (string, error) {
	tools := GetAgentTools()

	// 找到对应的工具
	var handler ToolHandler
	for _, t := range tools {
		if t.Name == toolCall.Function.Name {
			handler = t.Handler
			break
		}
	}

	if handler == nil {
		return "", fmt.Errorf("unknown tool: %s", toolCall.Function.Name)
	}

	// 解析参数
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	// 执行工具
	return handler(args, ctx)
}
