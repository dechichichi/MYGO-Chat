# MyGO!!!!! Chat

基于 LLM 的多 Agent 角色对话系统，和 MyGO!!!!! 乐队成员聊天。

> 迷子でもいい、迷子でも進め。
> 
> 迷路也没关系，迷路也要前进。

## 功能特性

- **一对一对话** - 和五位乐队成员进行深度对话
- **情绪感知** - AI 会根据你的情绪状态调整回应方式
- **乐队讨论会** - 观看成员们围绕话题展开讨论
- **Agent 能力** - 工具调用、记忆系统、反思机制
- **多 API 容错** - 支持多个 API 源自动切换

## 乐队成员

| 成员 | 代号 | 担当 | 性格特点 |
|------|------|------|----------|
| 高松灯 | `tomori` | 主唱 | 感性细腻的"羽丘怪女生"，直觉敏锐 |
| 千早爱音 | `anon` | 吉他 | 元气满满的优等生，渴望闪闪发光 |
| 要乐奈 | `rana` | 鼓手 | 神出鬼没的古怪少女，觉得一切都很有趣 |
| 长崎素世 | `soyo` | 贝斯 | 温柔的大姐姐，内心渴望真正的连接 |
| 椎名立希 | `taki` | 吉他 | 傲娇的独狼，嘴硬心软 |

## 快速开始

### 1. 配置

编辑 `config/config.yaml`，填入你的 API 密钥：

```yaml
base_url: "https://api.deepseek.com/v1"
token: "your-api-key"
model_name: "deepseek-chat"
temperature: 0.7
```

### 2. 运行

```bash
# 和高松灯聊天
go run main.go -mode=cli -member=tomori

# 和椎名立希聊天
go run main.go -mode=cli -member=taki

# 乐队讨论会
go run main.go -mode=debate

# 启动 API 服务器
go run main.go -mode=server -port=:8080
```

### 3. 对话示例

```
╔══════════════════════════════════════════════════════════════╗
║                     MyGO!!!!! Chat                           ║
║                   迷子でもいい v1.0                          ║
╚══════════════════════════════════════════════════════════════╝

🎸 你正在与 高松灯 (Takamatsu Tomori) 对话

你: 最近感觉有点迷茫，不知道自己想要什么

高松灯: 那个...迷茫的感觉，我也懂呢。
就像...站在十字路口，每条路都看不到尽头。
但是...我觉得，不知道想要什么，也没关系的。
有时候...走着走着，就会看到想要的东西了。
星星...也不是一开始就知道自己会发光的吧。
[心之所向...]
```

## 技术亮点

### 1. 四层架构 Prompt 工程 + 两层情绪感知系统

设计并实现**四层角色 Prompt 框架**（核心身份→思维框架→语言风格→回复规则），支持 5 个差异化角色的一致性表达；构建**两层情绪分析机制**：第一层通过关键词规则快速判断（痛苦/迷茫/抱怨/找借口），第二层调用 AI 进行深度情绪分析，实现动态调整回复风格。

```go
// 四层 Prompt 结构
type PhilosopherPrompt struct {
    Name              string   // 角色名称
    CoreIdentity      string   // 核心身份
    ThinkingFramework string   // 思维框架
    LinguisticStyle   string   // 语言风格
    ResponseRules     string   // 回复规则
}

// 两层情绪分析
func (a *EmotionAnalyzer) Analyze(text string) EmotionLevel {
    level := a.quickAnalyze(text)  // 第一层：关键词快速判断
    if level != EmotionNeutral {
        return level
    }
    return a.aiAnalyze(text)       // 第二层：AI 深度分析
}
```

### 2. 完整 Agent 能力：工具调用 + 记忆系统 + 反思机制

每个角色都是一个完整的 Agent，具备：

**工具调用能力**：
- `recall_memory` - 回忆与用户的过往对话
- `save_memory` - 保存重要信息到记忆
- `search_lyrics` - 搜索歌词找灵感
- `sense_atmosphere` - 感知当前对话氛围
- `reflect_response` - 反思自己的回复

**记忆系统**：
- 短期记忆：当前会话的对话历史
- 长期记忆：跨会话的重要信息（用户偏好、重要事件等）

**反思机制**：
- 生成回复后进行自我评估
- 检查性格一致性、情感恰当性、表达自然度
- 支持迭代优化直到达到目标质量

```go
// Agent 工具调用循环
func (a *Agent) Chat(userMessage string, history []Message) (*AgentResponse, error) {
    // 1. 情绪分析
    emotionLevel := a.EmotionAnalyzer.Analyze(userMessage)
    
    // 2. 调用模型（可能触发工具调用）
    response, toolResults := a.invokeWithTools(messages, tools)
    
    // 3. 反思并优化
    if a.EnableReflection {
        response = a.ReflectionEngine.ReflectAndRefine(response, ...)
    }
    
    // 4. 保存到记忆
    a.Context.ShortTermMemory = append(...)
    
    return response
}
```

### 3. 主持人 Agent 自主驱动讨论

讨论引擎由"主持人 Agent"自主驱动，而非硬编码流程：

- **自主决策**：主持人根据当前状态决定谁下一个发言、发言类型、是否结束
- **动态调整**：根据讨论进展灵活调整流程，而非固定的"开场→质询→总结"
- **意图识别**：识别成员发言的意图（质询、反驳、补充），触发对应的响应模式

```go
// 主持人自主决策
func (m *ModeratorAgent) Think() (*ModeratorDecision, error) {
    // 分析当前状态：谁发言了、话题走向、是否需要推进
    stateDesc := m.buildStateDescription()
    
    // 调用 AI 进行决策
    response := m.model.Invoke(systemPrompt, stateDesc)
    
    // 返回决策：下一个发言者、发言类型、指令
    return m.parseDecision(response)
}

// 决策类型
type ModeratorAction string
const (
    ActionOpeningSpeech   // 开场发言
    ActionAskQuestion     // 让某人提问
    ActionRequestAnswer   // 让某人回答
    ActionInviteComment   // 邀请评论
    ActionFreeDiscussion  // 自由讨论
    ActionRequestSummary  // 请求总结
    ActionEndDiscussion   // 结束讨论
)
```

### 4. 三层容错高可用架构 + LRU 缓存 + Jaccard 去重

实现**三层容错机制**（主 API→备用 API→静态兜底），支持多 API 源优先级配置、超时重试和自动故障转移；配合 **LRU 响应缓存**减少重复调用，**Jaccard 相似度去重**避免 AI 输出重复内容。

```go
// 三层容错调用
func (m *FaultTolerantModel) Invoke(messages []Message, tools []map[string]interface{}) (string, []ToolCall, error) {
    for _, source := range m.sources {  // 依次尝试每个 API 源
        content, toolCalls, err := m.invokeSource(source, messages, tools)
        if err == nil {
            return content, toolCalls, nil
        }
    }
    return m.fallbackMessage, nil, nil  // 全部失败返回兜底
}
```

## API 接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/chat` | POST | 一对一对话 |
| `/api/agent/chat` | POST | Agent 对话（支持工具调用、反思） |
| `/api/agent/discussion` | POST | 主持人 Agent 驱动讨论 |
| `/api/debate/start` | POST | 开始乐队讨论 |
| `/api/debate/status` | GET | 获取讨论状态 |
| `/api/philosophers` | GET | 获取成员列表 |
| `/api/health` | GET | 健康检查 |

### 对话请求示例

```bash
# 普通对话
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "user123",
    "message": "你好，灯",
    "philosopher": "tomori"
  }'

# Agent 对话（带工具调用和反思）
curl -X POST http://localhost:8080/api/agent/chat \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "user123",
    "message": "我最近感觉很迷茫",
    "philosopher": "tomori",
    "enable_tools": true,
    "enable_reflection": true
  }'

# 主持人驱动讨论
curl -X POST http://localhost:8080/api/agent/discussion \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "discussion1",
    "topic": "音乐的意义是什么"
  }'
```

## 项目结构

```
mygo-chat/
├── main.go              # 入口文件
├── config/
│   ├── config.go        # 配置加载
│   ├── config.yaml      # 配置文件
│   ├── model.go         # LLM 模型客户端
│   └── multi_api.go     # 多 API 源容错
├── philosopher/
│   ├── prompts.go       # 角色 Prompt 定义
│   ├── philosopher.go   # 角色基础实现
│   ├── agent.go         # 完整 Agent 实现
│   ├── tools.go         # Agent 工具系统
│   ├── reflection.go    # 反思机制
│   ├── moderator.go     # 主持人 Agent
│   ├── debate_engine.go # 讨论引擎
│   └── emotion.go       # 情绪分析
├── api/
│   └── handler.go       # HTTP API
├── web/                 # React 前端
│   ├── src/
│   │   ├── components/  # UI 组件
│   │   ├── hooks/       # 状态管理
│   │   └── types/       # 类型定义
│   └── package.json
└── utils/
    └── vars.go          # 常量定义
```

## 技术栈

- **后端**: Go, Gin, Viper, Resty
- **前端**: React, TypeScript, TailwindCSS, Vite
- **AI**: OpenAI Compatible API (支持 DeepSeek, 阿里云 DashScope 等)

## License

MIT
