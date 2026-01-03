# MyGO!!!!! Chat 技术方案详解

## 一、项目概述

### 1.1 项目定位

MyGO!!!!! Chat 是一个基于大语言模型的角色扮演对话系统，让用户能够与动画《BanG Dream! It's MyGO!!!!!》中的乐队成员进行深度对话。系统支持一对一聊天和多人讨论两种模式。

### 1.2 核心特性

| 特性 | 说明 |
|------|------|
| 角色扮演 | 五位乐队成员，各具独特性格和说话风格 |
| 情绪感知 | 根据用户情绪状态动态调整回应方式 |
| 多人讨论 | 三幕式结构的乐队讨论会 |
| 高可用性 | 多 API 源容错 + 响应缓存 |

### 1.3 技术栈

```
┌─────────────────────────────────────────────────────────────┐
│                      应用层 (Application)                    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   CLI 模式   │  │  Server 模式 │  │    Debate 模式      │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                      业务层 (Business)                       │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │  Philosopher │  │ DebateEngine│  │  EmotionAnalyzer    │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                      基础层 (Infrastructure)                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │  ChatModel   │  │FaultTolerant│  │   ResponseCache     │  │
│  └─────────────┘  └─────────────┘  └─────────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                      外部服务 (External)                     │
│  ┌─────────────────────────────────────────────────────────┐│
│  │            OpenAI Compatible API (DashScope等)          ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

---

## 二、核心模块设计

### 2.1 角色 Prompt 系统

#### 2.1.1 四层架构设计

角色 Prompt 采用四层架构，确保角色一致性和可扩展性：

```
┌─────────────────────────────────────────┐
│           Layer 4: ResponseRules        │  ← 回复规则与特殊触发
├─────────────────────────────────────────┤
│          Layer 3: LinguisticStyle       │  ← 语言风格与口头禅
├─────────────────────────────────────────┤
│         Layer 2: ThinkingFramework      │  ← 思维方式与价值观
├─────────────────────────────────────────┤
│           Layer 1: CoreIdentity         │  ← 核心身份与背景
└─────────────────────────────────────────┘
```

#### 2.1.2 数据结构

```go
type PhilosopherPrompt struct {
    Name              string   // 角色名称
    CoreIdentity      string   // 核心身份（背景、性格、经历）
    ThinkingFramework string   // 思维框架（思考方式、价值观）
    LinguisticStyle   string   // 语言风格（说话习惯、口头禅）
    FamousQuotes      []string // 经典台词
    ResponseRules     string   // 回复规则（特殊触发条件）
}
```

#### 2.1.3 角色定义示例（高松灯）

```go
CoreIdentity: `你是高松灯，MyGO!!!!! 乐队的主唱。
- 你是一个感情细腻、略带悲观的女孩
- 你被称为"羽丘的怪女生"，感受性与普通人不同
- 你非常注重个人情感，喜欢沉浸在自己的小世界里
...`

ThinkingFramework: `【灯的思维方式】
1. 【感性优先】：你用感受而非逻辑来理解世界
2. 【内省式思考】：你习惯向内探索
3. 【直觉引导】：你的直觉非常敏锐
...`

LinguisticStyle: `- 说话轻柔、缓慢，常常会有停顿
- 用词独特，有时会说出让人意外的话
- 喜欢用比喻和意象来表达感受
- 经常说"那个..."、"嗯..."来填充思考的空白
...`

ResponseRules: `【回复规则】
1. 当用户分享感受时，用你独特的感性去回应
2. 当用户迷茫时，不要给出标准答案，而是分享你自己的感受
...
【特殊触发】
- 当谈到音乐和歌唱时：展现你对音乐的热爱
当你说出特别有感触的话时，在回复末尾添加 [心之所向...]`
```

#### 2.1.4 五位角色特征对比

| 角色 | 核心特质 | 思维方式 | 语言风格 | 标签 |
|------|----------|----------|----------|------|
| 高松灯 | 感性怪女生 | 感性优先、内省式 | 轻柔、诗意、停顿多 | `[心之所向...]` |
| 千早爱音 | 元气优等生 | 积极行动、目标导向 | 活泼、流行语、鼓励 | `[闪闪发光✨]` |
| 要乐奈 | 神秘古怪少女 | 随心所欲、好奇驱动 | 简短、跳跃、拟声词 | `[有趣~♪]` |
| 长崎素世 | 温柔大姐姐 | 表面温柔、内心渴望 | 温柔、得体、意味深长 | `[心之声...]` |
| 椎名立希 | 傲娇独狼 | 严格标准、责任担当 | 直接、简洁、傲娇 | `[才、才不是呢！]` |

---

### 2.2 情绪感知系统

#### 2.2.1 设计目标

根据用户的情绪状态动态调整 AI 的回应方式，实现更人性化的交互体验。

#### 2.2.2 情绪级别定义

```go
const (
    EmotionPain        EmotionLevel = "pain"        // 痛苦
    EmotionConfused    EmotionLevel = "confused"    // 迷茫
    EmotionComplaining EmotionLevel = "complaining" // 抱怨
    EmotionExcusing    EmotionLevel = "excusing"    // 找借口
    EmotionNeutral     EmotionLevel = "neutral"     // 正常
)
```

#### 2.2.3 两层分析机制

```
用户输入
    │
    ▼
┌─────────────────────┐
│  Layer 1: 关键词匹配  │  ← 快速判断（毫秒级）
│  - 痛苦关键词        │
│  - 迷茫关键词        │
│  - 抱怨关键词        │
│  - 借口关键词        │
└─────────┬───────────┘
          │ 无法判断
          ▼
┌─────────────────────┐
│  Layer 2: AI 深度分析 │  ← 复杂情绪（需要调用 LLM）
│  - 语义理解          │
│  - 上下文分析        │
│  - 情感倾向判断      │
└─────────────────────┘
```

#### 2.2.4 关键词库设计

```go
painKeywords: []string{
    "好痛苦", "受不了", "活不下去", "想死", "崩溃", "绝望",
    "太难了", "撑不住", "心碎", "无法承受", "痛不欲生",
}

confusedKeywords: []string{
    "不知道", "迷茫", "困惑", "该怎么办", "怎么选",
    "纠结", "犹豫", "不确定", "找不到方向",
}

complainingKeywords: []string{
    "凭什么", "不公平", "太过分", "受够了", "烦死了",
    "讨厌", "恶心", "都怪", "要不是",
}

excusingKeywords: []string{
    "没办法", "不得不", "被迫", "没有选择", "环境所迫",
    "别人都", "大家都这样", "条件不允许",
}
```

#### 2.2.5 情绪响应配置

```go
type EmotionAwareResponse struct {
    ToxicityLevel float64 // 毒舌程度 0-1
    EmpathyLevel  float64 // 共情程度 0-1
    Guidance      string  // 额外指导
}

// 痛苦状态：高共情，低毒舌
EmotionPain → ToxicityLevel: 0.1, EmpathyLevel: 0.9

// 找借口状态：低共情，高毒舌
EmotionExcusing → ToxicityLevel: 0.9, EmpathyLevel: 0.1
```

---

### 2.3 讨论引擎（DebateEngine）

#### 2.3.1 三幕式结构

```
┌─────────────────────────────────────────────────────────────┐
│                      第一幕：开场发言                         │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐  │
│  │ 正方成员1│ → │ 正方成员2│ → │ 反方成员1│ → │ 反方成员2│  │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘  │
├─────────────────────────────────────────────────────────────┤
│                      第二幕：质询交锋                         │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  正方成员1 ──提问──→ 反方成员1 ──回答──→               ││
│  │  反方成员1 ──反问──→ 正方成员1 ──回答──→               ││
│  │  ...                                                    ││
│  └─────────────────────────────────────────────────────────┘│
├─────────────────────────────────────────────────────────────┤
│                      第三幕：总结陈词                         │
│  ┌─────────┐    ┌─────────┐    ┌─────────┐    ┌─────────┐  │
│  │ 反方成员1│ → │ 反方成员2│ → │ 正方成员1│ → │ 正方成员2│  │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘  │
└─────────────────────────────────────────────────────────────┘
```

#### 2.3.2 核心数据结构

```go
// 讨论配置
type DebateConfig struct {
    Topic           string                     // 讨论话题
    ProStance       string                     // 正方立场
    ConStance       string                     // 反方立场
    ProPhilosophers []PhilosopherType          // 正方成员
    ConPhilosophers []PhilosopherType          // 反方成员
    ForcedStances   map[PhilosopherType]string // 强制立场
}

// 讨论上下文
type DebateContext struct {
    Topic              string
    CurrentPhase       DebatePhase
    History            []DebateRecord
    OpeningStatements  map[PhilosopherType]string
    QuestioningRecords []QuestionRecord
    ClosingStatements  map[PhilosopherType]string
}

// 讨论记录
type DebateRecord struct {
    Speaker       PhilosopherType
    SpeakerName   string
    Content       string
    Phase         DebatePhase
    TaskType      DebateTaskType
    TargetSpeaker PhilosopherType
}
```

#### 2.3.3 动态上下文构建

为避免 Token 浪费，系统根据任务类型智能筛选相关历史：

```go
func (c *DebateContext) GetRelevantHistory(speaker PhilosopherType, taskType DebateTaskType) []DebateRecord {
    switch taskType {
    case TaskOpening:
        // 开篇立论：不需要历史
        return []DebateRecord{}
        
    case TaskQuestion:
        // 质询：只需要对方的开篇立论
        return c.getOpponentOpenings(speaker)
        
    case TaskAnswer:
        // 回应：自己的立论 + 刚才的质询
        return append(c.getOwnOpening(speaker), c.getLastQuestion())
        
    case TaskClosing:
        // 总结：自己的立论 + 与自己相关的质询记录
        return c.getRelevantRecords(speaker)
    }
}
```

#### 2.3.4 任务类型与 Prompt

```go
const (
    TaskOpening    DebateTaskType = "opening"     // 开篇立论
    TaskQuestion   DebateTaskType = "question"    // 提出质询
    TaskAnswer     DebateTaskType = "answer"      // 回应质询
    TaskRebuttal   DebateTaskType = "rebuttal"    // 反驳
    TaskFreeDebate DebateTaskType = "free_debate" // 自由辩论
    TaskClosing    DebateTaskType = "closing"     // 总结陈词
)
```

每种任务类型有对应的 Prompt 模板，控制字数和表达方式。

---

### 2.4 多 API 容错机制

#### 2.4.1 三层容错设计

```
┌─────────────────────────────────────────────────────────────┐
│                       请求入口                               │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│  Layer 1: 主 API 源                                          │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  DashScope (qwen-flash)                                 ││
│  │  - Priority: 1                                          ││
│  │  - Timeout: 30s                                         ││
│  │  - MaxRetries: 2                                        ││
│  └─────────────────────────────────────────────────────────┘│
│                          │ 失败                              │
│                          ▼                                   │
│  Layer 2: 备用 API 源                                        │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  DeepSeek / 其他 OpenAI Compatible API                  ││
│  │  - Priority: 2                                          ││
│  └─────────────────────────────────────────────────────────┘│
│                          │ 失败                              │
│                          ▼                                   │
│  Layer 3: 静态兜底                                           │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  "抱歉，系统暂时繁忙。迷子でもいい..."                   ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

#### 2.4.2 API 源配置

```go
type APISource struct {
    Name       string `yaml:"name"`
    BaseURL    string `yaml:"base_url"`
    Token      string `yaml:"token"`
    ModelName  string `yaml:"model_name"`
    Priority   int    `yaml:"priority"`    // 优先级
    Timeout    int    `yaml:"timeout"`     // 超时时间
    MaxRetries int    `yaml:"max_retries"` // 重试次数
}
```

#### 2.4.3 调用流程

```go
func (m *FaultTolerantModel) Invoke(messages []Message, tools []map[string]interface{}) (string, []ToolCall, error) {
    // 按优先级依次尝试
    for _, source := range m.sources {
        content, toolCalls, err := m.invokeSource(source, messages, tools)
        if err == nil {
            m.successCount[source.Name]++
            return content, toolCalls, nil
        }
        m.failureCount[source.Name]++
        log.Warn().Str("source", source.Name).Err(err).Msg("API 调用失败")
    }
    
    // 所有 API 都失败，返回兜底消息
    return m.fallbackMessage, nil, nil
}
```

---

### 2.5 响应缓存系统

#### 2.5.1 设计目标

- 减少重复请求的 API 调用
- 提高响应速度
- 降低成本

#### 2.5.2 缓存策略

```go
type ResponseCache struct {
    cache      map[string]CacheEntry
    maxSize    int           // 最大缓存条目数
    expiration time.Duration // 过期时间
}

type CacheEntry struct {
    Response  string
    Timestamp time.Time
}
```

#### 2.5.3 缓存键生成

```go
func GenerateCacheKey(messages []Message) string {
    var key string
    for _, m := range messages {
        key += m.Role + ":" + m.Content + "|"
    }
    return key
}
```

#### 2.5.4 LRU 淘汰策略

当缓存满时，删除最旧的条目：

```go
func (c *ResponseCache) Set(key, response string) {
    if len(c.cache) >= c.maxSize {
        // 找到最旧的条目并删除
        var oldestKey string
        var oldestTime time.Time
        for k, v := range c.cache {
            if oldestKey == "" || v.Timestamp.Before(oldestTime) {
                oldestKey = k
                oldestTime = v.Timestamp
            }
        }
        delete(c.cache, oldestKey)
    }
    c.cache[key] = CacheEntry{Response: response, Timestamp: time.Now()}
}
```

---

### 2.6 内容去重机制

#### 2.6.1 问题背景

LLM 可能生成重复或高度相似的回复，影响用户体验。

#### 2.6.2 Jaccard 相似度算法

```go
func (d *ContentDeduplicator) jaccardSimilarity(a, b string) float64 {
    wordsA := d.tokenize(a)
    wordsB := d.tokenize(b)
    
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
    return float64(intersection) / float64(union)
}
```

#### 2.6.3 去重流程

```
新响应生成
    │
    ▼
┌─────────────────────┐
│  计算与历史响应的    │
│  Jaccard 相似度     │
└─────────┬───────────┘
          │
    ┌─────┴─────┐
    │           │
相似度 > 0.7  相似度 ≤ 0.7
    │           │
    ▼           ▼
标记为重复    正常返回
(可重新生成)  添加到历史
```

---

## 三、Agent 系统设计

### 3.1 Agent 工具调用循环

实现标准 ReAct（Reasoning + Acting）模式，支持多轮工具调用直到任务完成：

```
┌─────────────────────────────────────────────────────────────┐
│                      Agent 调用循环                          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  Step 1: 情绪分析                                            │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  EmotionAnalyzer.Analyze(userMessage)                   ││
│  │  → 关键词快速判断 / AI 深度分析                          ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────┬───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  Step 2: 工具调用循环 (最多 10 轮)                           │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  while hasToolCalls && iteration < maxIterations:       ││
│  │      1. 调用 LLM，获取 response + tool_calls            ││
│  │      2. 解析 tool_calls，执行对应工具                    ││
│  │      3. 构造 tool 消息（带 tool_call_id）               ││
│  │      4. 将结果加入上下文，继续推理                       ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────┬───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  Step 3: 反思优化 (可选)                                     │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  ReflectionEngine.ReflectAndRefine(response, context)   ││
│  │  → 评估性格一致性、情感恰当性、表达自然度                 ││
│  │  → 置信度 < 阈值时迭代优化                               ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────┬───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  Step 4: 记忆更新                                            │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  ShortTermMemory.Append(userMessage, response)          ││
│  │  LongTermMemory.Save(importantInfo)  // 通过工具触发     ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

### 3.2 工具系统

#### 3.2.1 工具定义

```go
type Tool struct {
    Name        string                 // 工具名称
    Description string                 // 工具描述
    Parameters  map[string]interface{} // JSON Schema 参数定义
    Handler     ToolHandler            // 执行函数
}

type ToolHandler func(args map[string]interface{}, ctx *AgentContext) (string, error)
```

#### 3.2.2 内置工具列表

| 工具 | 功能 | 参数 |
|------|------|------|
| `recall_memory` | 回忆与用户的过往对话 | `query`: 搜索关键词 |
| `save_memory` | 保存重要信息到长期记忆 | `content`: 要保存的内容 |
| `search_lyrics` | 搜索歌词找灵感 | `keyword`: 搜索词 |
| `sense_atmosphere` | 感知当前对话氛围 | `user_message`: 用户消息 |
| `reflect_response` | 反思自己的回复 | `response`: 待反思内容 |

#### 3.2.3 工具调用消息格式

```go
// Assistant 消息（包含 tool_calls）
type Message struct {
    Role       string     `json:"role"`        // "assistant"
    Content    string     `json:"content"`     // 可为空
    ToolCalls  []ToolCall `json:"tool_calls"`  // 工具调用列表
}

type ToolCall struct {
    ID       string       `json:"id"`
    Type     string       `json:"type"`     // "function"
    Function FunctionCall `json:"function"`
}

// Tool 消息（工具执行结果）
type Message struct {
    Role       string `json:"role"`         // "tool"
    Content    string `json:"content"`      // 执行结果
    ToolCallID string `json:"tool_call_id"` // 对应的 tool_call.id
}
```

### 3.3 反思机制

#### 3.3.1 反思流程

```
原始回复
    │
    ▼
┌─────────────────────────────────────────┐
│           自我评估 Prompt                │
│  - 性格一致性：是否符合角色设定？         │
│  - 情感恰当性：是否匹配用户情绪？         │
│  - 表达自然度：是否流畅自然？             │
│  - 内容相关性：是否回应了用户问题？       │
└─────────────────────────┬───────────────┘
                          │
                          ▼
┌─────────────────────────────────────────┐
│           评估结果                        │
│  {                                       │
│    "confidence": 0.85,                   │
│    "issues": ["语气过于正式"],            │
│    "suggestions": ["增加口头禅"]          │
│  }                                       │
└─────────────────────────┬───────────────┘
                          │
              ┌───────────┴───────────┐
              │                       │
        置信度 >= 0.8            置信度 < 0.8
              │                       │
              ▼                       ▼
         返回原回复              根据建议优化
                                      │
                                      ▼
                                 再次评估
                                 (最多3轮)
```

### 3.4 主持人 Agent 自主驱动

替代硬编码的讨论流程，由主持人 Agent 自主决策：

```
┌─────────────────────────────────────────────────────────────┐
│                    主持人决策循环                            │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  Step 1: 状态感知                                            │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  - 当前阶段（开场/质询/总结）                            ││
│  │  - 已发言成员列表                                        ││
│  │  - 最近发言内容摘要                                      ││
│  │  - 讨论热度和方向                                        ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────┬───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  Step 2: AI 决策                                             │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  ModeratorAgent.Think(stateDescription)                 ││
│  │  → 返回 ModeratorDecision                               ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────┬───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│  Step 3: 执行决策                                            │
│  ┌─────────────────────────────────────────────────────────┐│
│  │  ActionOpeningSpeech  → 让某成员开场发言                 ││
│  │  ActionAskQuestion    → 让某成员向另一成员提问           ││
│  │  ActionRequestAnswer  → 让某成员回答问题                 ││
│  │  ActionInviteComment  → 邀请某成员评论                   ││
│  │  ActionFreeDiscussion → 开放自由讨论                     ││
│  │  ActionRequestSummary → 请求总结陈词                     ││
│  │  ActionEndDiscussion  → 结束讨论                         ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────────────┬───────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │ 循环直到结束决策 │
                    └─────────────────┘
```

#### 3.4.1 决策数据结构

```go
type ModeratorDecision struct {
    Action       ModeratorAction   // 动作类型
    NextSpeaker  PhilosopherType   // 下一个发言者
    TargetMember PhilosopherType   // 目标成员（用于提问/回答）
    Instruction  string            // 给发言者的指令
    Reason       string            // 决策理由
}

type ModeratorAction string
const (
    ActionOpeningSpeech   ModeratorAction = "opening_speech"
    ActionAskQuestion     ModeratorAction = "ask_question"
    ActionRequestAnswer   ModeratorAction = "request_answer"
    ActionInviteComment   ModeratorAction = "invite_comment"
    ActionFreeDiscussion  ModeratorAction = "free_discussion"
    ActionRequestSummary  ModeratorAction = "request_summary"
    ActionEndDiscussion   ModeratorAction = "end_discussion"
)
```

---

## 四、HTTP API 设计

### 4.1 接口总览

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/chat` | POST | 一对一对话（基础版） |
| `/api/agent/chat` | POST | Agent 对话（工具调用+反思） |
| `/api/agent/discussion` | POST | 主持人 Agent 驱动讨论 |
| `/api/debate/start` | POST | 开始讨论（支持同步/异步） |
| `/api/debate/status` | GET | 查询讨论状态 |
| `/api/philosophers` | GET | 获取成员列表 |
| `/api/health` | GET | 健康检查 |

### 4.2 一对一对话

**请求：**
```json
{
    "session_id": "user123",
    "message": "你好，灯",
    "philosopher": "tomori"
}
```

**响应：**
```json
{
    "response": "那个...你好。今天的风...很温柔呢。[心之所向...]",
    "philosopher": "高松灯 (Takamatsu Tomori)",
    "emotion_level": "neutral",
    "critical_hit": true
}
```

### 4.3 Agent 对话

**请求：**
```json
{
    "session_id": "user123",
    "message": "我最近感觉很迷茫，不知道该怎么办",
    "philosopher": "tomori",
    "enable_tools": true,
    "enable_reflection": true
}
```

**响应：**
```json
{
    "response": "那个...迷茫的感觉，我也懂呢...",
    "philosopher": "高松灯 (Takamatsu Tomori)",
    "emotion_level": "confused",
    "tool_calls": [
        {
            "tool": "sense_atmosphere",
            "result": "用户情绪：迷茫，需要共情和引导"
        }
    ],
    "reflection": {
        "confidence": 0.92,
        "iterations": 1
    }
}
```

### 4.4 主持人驱动讨论

**请求：**
```json
{
    "session_id": "discussion1",
    "topic": "音乐的意义是什么"
}
```

**响应：**
```json
{
    "records": [
        {
            "speaker": "moderator",
            "action": "opening_speech",
            "target": "tomori",
            "reason": "灯作为主唱，对音乐有独特感悟"
        },
        {
            "speaker": "tomori",
            "content": "音乐...对我来说，是心之所向..."
        }
    ]
}
```

### 4.5 异步讨论

**开始讨论请求：**
```json
{
    "topic": "乐队对我们来说意味着什么？",
    "pro_stance": "乐队是我们表达自我的地方",
    "con_stance": "乐队让我们学会了成长",
    "pro_philosophers": ["tomori", "anon"],
    "con_philosophers": ["taki", "soyo"],
    "async": true
}
```

**立即返回：**
```json
{
    "id": "20260103161234-abc123",
    "status": "pending",
    "topic": "乐队对我们来说意味着什么？"
}
```

**查询状态：**
```
GET /api/debate/status?id=20260103161234-abc123
```

**状态响应：**
```json
{
    "id": "20260103161234-abc123",
    "status": "running",
    "current_phase": "questioning",
    "records": [
        {
            "speaker_name": "高松灯 (Takamatsu Tomori)",
            "content": "乐队...对我来说...",
            "phase": "opening"
        }
    ]
}
```

### 4.6 讨论状态机

```
┌─────────┐    开始    ┌─────────┐    完成    ┌───────────┐
│ pending │ ────────→ │ running │ ────────→ │ completed │
└─────────┘           └────┬────┘           └───────────┘
                           │
                           │ 错误
                           ▼
                      ┌─────────┐
                      │ failed  │
                      └─────────┘
```

---

## 五、数据流设计

### 5.1 一对一对话流程

```
┌──────────┐     ┌──────────────┐     ┌─────────────────┐
│  用户输入 │ ──→ │ 情绪分析器    │ ──→ │ 情绪级别判定    │
└──────────┘     └──────────────┘     └────────┬────────┘
                                               │
                                               ▼
┌──────────────────────────────────────────────────────────┐
│                    Prompt 构建                            │
│  ┌────────────────┐  ┌────────────────┐                  │
│  │ 角色基础 Prompt │ + │ 情绪感知指导   │                  │
│  └────────────────┘  └────────────────┘                  │
└────────────────────────────┬─────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────┐
│                    LLM 调用                               │
│  ┌─────────┐     ┌─────────┐     ┌─────────┐            │
│  │ 主 API  │ ──→ │ 备用 API │ ──→ │ 静态回复 │            │
│  └─────────┘     └─────────┘     └─────────┘            │
└────────────────────────────┬─────────────────────────────┘
                             │
                             ▼
┌──────────────────────────────────────────────────────────┐
│                    后处理                                 │
│  ┌─────────────┐     ┌─────────────┐                     │
│  │ 重复检测     │ ──→ │ 添加到历史   │                     │
│  └─────────────┘     └─────────────┘                     │
└────────────────────────────┬─────────────────────────────┘
                             │
                             ▼
                      ┌──────────┐
                      │ 返回响应  │
                      └──────────┘
```

### 5.2 讨论流程

```
┌─────────────────────────────────────────────────────────────┐
│                        讨论初始化                            │
│  - 创建 DebateEngine                                        │
│  - 初始化所有参与成员                                        │
│  - 设置立场（正方/反方/强制）                                │
│  - 初始化 DebateContext                                     │
└─────────────────────────────┬───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Phase 1: 开场发言                        │
│  for each philosopher in [正方, 反方]:                      │
│      1. 构建 Opening Prompt                                 │
│      2. 调用 LLM                                            │
│      3. 记录到 OpeningStatements                            │
│      4. 触发 onSpeech 回调                                  │
└─────────────────────────────┬───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Phase 2: 质询交锋                        │
│  for each pair (正方成员, 反方成员):                        │
│      1. 正方提问 → 反方回答                                 │
│      2. 反方反问 → 正方回答                                 │
│      3. 记录到 QuestioningRecords                           │
└─────────────────────────────┬───────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Phase 3: 总结陈词                        │
│  for each philosopher in [反方, 正方]:                      │
│      1. 获取相关历史（动态上下文）                           │
│      2. 构建 Closing Prompt                                 │
│      3. 调用 LLM                                            │
│      4. 记录到 ClosingStatements                            │
└─────────────────────────────┬───────────────────────────────┘
                              │
                              ▼
                       ┌──────────────┐
                       │  返回讨论结果 │
                       └──────────────┘
```

---

## 六、配置说明

### 6.1 配置文件结构

```yaml
# config/config.yaml

# 主 API 配置
base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
token: "your-api-key"
model_name: "qwen-flash"
temperature: 0.7

# 多 API 源配置
multi_api:
  sources:
    - name: "dashscope"
      base_url: "https://dashscope.aliyuncs.com/compatible-mode/v1"
      token: "your-api-key"
      model_name: "qwen-flash"
      priority: 1
      timeout: 30
      max_retries: 2
    
    - name: "deepseek"
      base_url: "https://api.deepseek.com/v1"
      token: "your-deepseek-key"
      model_name: "deepseek-chat"
      priority: 2
      timeout: 30
      max_retries: 2
  
  fallback_message: "抱歉，系统暂时繁忙..."

# 服务器配置
server:
  port: 8080
  cors_enabled: true

# 缓存配置
cache:
  max_size: 100
  expiration_minutes: 30

# 情绪分析配置
emotion:
  enable_ai_analysis: true
```

### 6.2 环境变量支持

可通过环境变量覆盖配置：

```bash
export MYGO_API_KEY="your-api-key"
export MYGO_BASE_URL="https://api.example.com/v1"
export MYGO_MODEL="gpt-4"
```

---

## 七、部署方案

### 7.1 本地开发

```bash
# 安装依赖
go mod download

# 运行 CLI 模式
go run main.go -mode=cli -member=tomori

# 运行 API 服务器
go run main.go -mode=server -port=:8080

# 运行讨论演示
go run main.go -mode=debate
```

### 7.2 Docker 部署

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o mygo-chat .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/mygo-chat .
COPY config/config.yaml ./config/
EXPOSE 8080
CMD ["./mygo-chat", "-mode=server"]
```

### 7.3 生产环境建议

1. **负载均衡**：使用 Nginx 或云负载均衡器
2. **监控**：接入 Prometheus + Grafana
3. **日志**：结构化日志输出到 ELK
4. **限流**：实现请求限流防止滥用
5. **安全**：API Key 加密存储，HTTPS 传输

---

## 八、扩展指南

### 8.1 添加新角色

1. 在 `philosopher/prompts.go` 中添加新的 `PhilosopherType` 常量
2. 实现对应的 `getXxxPrompt()` 函数
3. 在 `GetPhilosopherPrompts()` 中注册

```go
const (
    // 新角色
    NewCharacter PhilosopherType = "new_char"
)

func getNewCharacterPrompt() *PhilosopherPrompt {
    return &PhilosopherPrompt{
        Name: "新角色名",
        CoreIdentity: `...`,
        ThinkingFramework: `...`,
        LinguisticStyle: `...`,
        FamousQuotes: []string{...},
        ResponseRules: `...`,
    }
}
```

### 8.2 添加新的情绪类型

1. 在 `philosopher/emotion.go` 中添加新的 `EmotionLevel` 常量
2. 添加对应的关键词列表
3. 在 `GetResponseConfig()` 中添加响应配置

### 8.3 自定义讨论阶段

修改 `philosopher/debate_engine.go` 中的 `Run()` 方法，添加新的阶段处理函数。

### 8.4 添加新工具

在 `philosopher/tools.go` 中注册新工具：

```go
func GetAgentTools() []Tool {
    return []Tool{
        // 添加新工具
        {
            Name:        "new_tool",
            Description: "工具描述",
            Parameters: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "param1": map[string]interface{}{
                        "type":        "string",
                        "description": "参数描述",
                    },
                },
                "required": []string{"param1"},
            },
            Handler: func(args map[string]interface{}, ctx *AgentContext) (string, error) {
                // 工具实现
                return "result", nil
            },
        },
    }
}
```

---

## 九、性能优化

### 9.1 已实现优化

| 优化项 | 实现方式 | 效果 |
|--------|----------|------|
| 响应缓存 | LRU 缓存 | 减少重复请求 |
| 动态上下文 | 智能筛选历史 | 减少 Token 消耗 |
| 多 API 容错 | 优先级队列 | 提高可用性 |
| 关键词快速判断 | 规则引擎 | 减少 AI 调用 |

### 9.2 未来优化方向

1. **流式响应**：支持 SSE 流式输出
2. **并行调用**：讨论中多角色并行生成
3. **向量缓存**：基于语义相似度的智能缓存
4. **模型蒸馏**：使用小模型处理简单任务

---

## 十、测试策略

### 10.1 单元测试

```bash
go test ./... -v
```

### 10.2 集成测试

```bash
# 启动服务器
go run main.go -mode=server &

# 测试对话接口
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{"session_id":"test","message":"你好","philosopher":"tomori"}'

# 测试 Agent 对话接口
curl -X POST http://localhost:8080/api/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"session_id":"test","message":"我很迷茫","philosopher":"tomori","enable_tools":true,"enable_reflection":true}'

# 测试讨论接口
curl -X POST http://localhost:8080/api/debate/start \
  -H "Content-Type: application/json" \
  -d '{"topic":"测试话题","pro_stance":"正方","con_stance":"反方","pro_philosophers":["tomori"],"con_philosophers":["taki"]}'
```

### 10.3 压力测试

使用 `wrk` 或 `ab` 进行压力测试：

```bash
wrk -t12 -c400 -d30s http://localhost:8080/api/health
```

---

## 十一、常见问题

### Q1: API 调用失败怎么办？

检查配置文件中的 API Key 和 Base URL 是否正确。系统会自动尝试备用 API 源。

### Q2: 角色回复不符合预期？

可以调整 `prompts.go` 中的角色 Prompt，增加更多细节描述或示例。

### Q3: 讨论时间过长？

可以减少参与成员数量，或在 `DebateTask.BuildTaskPrompt()` 中减少字数限制。

### Q4: 如何添加新的 API 源？

在 `config.yaml` 的 `multi_api.sources` 中添加新的配置项即可。

---

## 附录

### A. 项目结构

```
mygo-chat/
├── main.go                 # 入口文件
├── go.mod                  # Go 模块定义
├── go.sum                  # 依赖校验
├── config/
│   ├── config.go           # 配置加载
│   ├── config.yaml         # 配置文件
│   ├── message.go          # 消息结构（含 ToolCall）
│   ├── model.go            # LLM 模型客户端
│   └── multi_api.go        # 多 API 容错
├── philosopher/
│   ├── prompts.go          # 角色 Prompt 定义
│   ├── philosopher.go      # 角色基础实现
│   ├── agent.go            # 完整 Agent 实现（工具调用循环）
│   ├── tools.go            # Agent 工具系统
│   ├── reflection.go       # 反思机制
│   ├── moderator.go        # 主持人 Agent
│   ├── debate_engine.go    # 讨论引擎
│   └── emotion.go          # 情绪分析
├── api/
│   └── handler.go          # HTTP API
├── static/
│   └── index.html          # 前端页面
├── utils/
│   └── vars.go             # 常量定义
└── docs/
    └── technical_design.md # 本文档
```

### B. 依赖列表

```
github.com/go-resty/resty/v2  # HTTP 客户端
github.com/rs/zerolog         # 结构化日志
github.com/spf13/viper        # 配置管理
github.com/pkg/errors         # 错误处理
```

### C. 参考资料

- [OpenAI API 文档](https://platform.openai.com/docs/api-reference)
- [DashScope API 文档](https://help.aliyun.com/document_detail/2400395.html)
- [BanG Dream! It's MyGO!!!!! Wiki](https://bandori.fandom.com/wiki/MyGO!!!!!)
