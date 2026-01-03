package config

import (
	"encoding/json"
	"net/http"
	"time"

	"agent/utils"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

// APISource API 源配置
type APISource struct {
	Name       string `yaml:"name"`
	BaseURL    string `yaml:"base_url"`
	Token      string `yaml:"token"`
	ModelName  string `yaml:"model_name"`
	Priority   int    `yaml:"priority"`    // 优先级，数字越小优先级越高
	Timeout    int    `yaml:"timeout"`     // 超时时间（秒）
	MaxRetries int    `yaml:"max_retries"` // 最大重试次数
}

// MultiAPIConfig 多 API 源配置
type MultiAPIConfig struct {
	Sources         []APISource `yaml:"sources"`
	FallbackMessage string      `yaml:"fallback_message"` // 所有 API 都失败时的兜底消息
}

// FaultTolerantModel 容错模型
// 实现三层容错机制：主 API -> 备用 API -> 静态回复
type FaultTolerantModel struct {
	sources         []APISource
	fallbackMessage string
	client          *http.Client

	// 统计信息
	successCount map[string]int
	failureCount map[string]int
}

// NewFaultTolerantModel 创建容错模型
func NewFaultTolerantModel(cfg *MultiAPIConfig) *FaultTolerantModel {
	// 按优先级排序
	sources := make([]APISource, len(cfg.Sources))
	copy(sources, cfg.Sources)

	// 简单冒泡排序
	for i := 0; i < len(sources)-1; i++ {
		for j := 0; j < len(sources)-i-1; j++ {
			if sources[j].Priority > sources[j+1].Priority {
				sources[j], sources[j+1] = sources[j+1], sources[j]
			}
		}
	}

	fallback := cfg.FallbackMessage
	if fallback == "" {
		fallback = "抱歉，系统暂时繁忙，请稍后再试。不过，真正的哲学家不会因为技术问题而停止思考。"
	}

	return &FaultTolerantModel{
		sources:         sources,
		fallbackMessage: fallback,
		client:          &http.Client{},
		successCount:    make(map[string]int),
		failureCount:    make(map[string]int),
	}
}

// Invoke 调用模型（带容错）
func (m *FaultTolerantModel) Invoke(messages []Message, tools []map[string]interface{}) (string, []ToolCall, error) {
	var lastErr error

	// 依次尝试每个 API 源
	for _, source := range m.sources {
		content, toolCalls, err := m.invokeSource(source, messages, tools)
		if err == nil {
			m.successCount[source.Name]++
			return content, toolCalls, nil
		}

		log.Warn().
			Str("source", source.Name).
			Err(err).
			Msg("API 调用失败，尝试下一个源")

		m.failureCount[source.Name]++
		lastErr = err
	}

	// 所有 API 都失败，返回兜底消息
	log.Error().Err(lastErr).Msg("所有 API 源都失败，返回兜底消息")
	return m.fallbackMessage, nil, nil
}

// invokeSource 调用单个 API 源
func (m *FaultTolerantModel) invokeSource(source APISource, messages []Message, tools []map[string]interface{}) (string, []ToolCall, error) {
	reqBody := map[string]interface{}{
		"model":       source.ModelName,
		"messages":    messages,
		"temperature": 0.7,
	}
	if len(tools) > 0 {
		reqBody["tools"] = tools
	}

	timeout := source.Timeout
	if timeout == 0 {
		timeout = 30
	}

	maxRetries := source.MaxRetries
	if maxRetries == 0 {
		maxRetries = 2
	}

	client := resty.New().
		SetTimeout(time.Duration(timeout) * time.Second).
		SetRetryCount(maxRetries).
		SetRetryWaitTime(1 * time.Second)

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", "Bearer "+source.Token).
		SetBody(reqBody).
		Post(source.BaseURL + utils.ChatCompletionsPath)

	if err != nil {
		return "", nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return "", nil, &APIError{
			StatusCode: resp.StatusCode(),
			Body:       string(resp.Body()),
		}
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content   string     `json:"content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return "", nil, err
	}

	if len(result.Choices) == 0 {
		return "", nil, &APIError{Message: "no response from model"}
	}

	msg := result.Choices[0].Message
	return msg.Content, msg.ToolCalls, nil
}

// GetStats 获取统计信息
func (m *FaultTolerantModel) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"success": m.successCount,
		"failure": m.failureCount,
	}
}

// APIError API 错误
type APIError struct {
	StatusCode int
	Body       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "API error: " + e.Body
}

// IntelligentModelRouter 智能模型路由器
// 根据任务复杂度选择合适的模型
type IntelligentModelRouter struct {
	heavyModel *FaultTolerantModel // 重型模型（用于复杂任务）
	lightModel *FaultTolerantModel // 轻型模型（用于简单任务）
}

// TaskComplexity 任务复杂度
type TaskComplexity string

const (
	ComplexityHigh   TaskComplexity = "high"   // 高复杂度：辩论、深度分析
	ComplexityMedium TaskComplexity = "medium" // 中复杂度：普通对话
	ComplexityLow    TaskComplexity = "low"    // 低复杂度：简单任务
)

// NewIntelligentModelRouter 创建智能路由器
func NewIntelligentModelRouter(heavyCfg, lightCfg *MultiAPIConfig) *IntelligentModelRouter {
	return &IntelligentModelRouter{
		heavyModel: NewFaultTolerantModel(heavyCfg),
		lightModel: NewFaultTolerantModel(lightCfg),
	}
}

// Route 根据复杂度路由到合适的模型
func (r *IntelligentModelRouter) Route(complexity TaskComplexity) *FaultTolerantModel {
	switch complexity {
	case ComplexityHigh:
		return r.heavyModel
	case ComplexityLow:
		return r.lightModel
	default:
		return r.heavyModel
	}
}

// AnalyzeComplexity 分析任务复杂度
func AnalyzeComplexity(task string) TaskComplexity {
	// 简单的启发式规则
	if len(task) > 500 {
		return ComplexityHigh
	}

	// 包含辩论相关关键词
	debateKeywords := []string{"辩论", "辩题", "正方", "反方", "立论", "质询"}
	for _, kw := range debateKeywords {
		if containsString(task, kw) {
			return ComplexityHigh
		}
	}

	// 包含深度分析关键词
	analysisKeywords := []string{"分析", "为什么", "如何", "意义", "本质"}
	for _, kw := range analysisKeywords {
		if containsString(task, kw) {
			return ComplexityMedium
		}
	}

	return ComplexityLow
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsString(s[1:], substr) || s[:len(substr)] == substr)
}

// ResponseCache 响应缓存
type ResponseCache struct {
	cache      map[string]CacheEntry
	maxSize    int
	expiration time.Duration
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Response  string
	Timestamp time.Time
}

// NewResponseCache 创建响应缓存
func NewResponseCache(maxSize int, expiration time.Duration) *ResponseCache {
	return &ResponseCache{
		cache:      make(map[string]CacheEntry),
		maxSize:    maxSize,
		expiration: expiration,
	}
}

// Get 获取缓存
func (c *ResponseCache) Get(key string) (string, bool) {
	entry, ok := c.cache[key]
	if !ok {
		return "", false
	}

	// 检查是否过期
	if time.Since(entry.Timestamp) > c.expiration {
		delete(c.cache, key)
		return "", false
	}

	return entry.Response, true
}

// Set 设置缓存
func (c *ResponseCache) Set(key, response string) {
	// 如果缓存已满，删除最旧的条目
	if len(c.cache) >= c.maxSize {
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

	c.cache[key] = CacheEntry{
		Response:  response,
		Timestamp: time.Now(),
	}
}

// GenerateCacheKey 生成缓存键
func GenerateCacheKey(messages []Message) string {
	// 简单地将所有消息内容拼接
	var key string
	for _, m := range messages {
		key += m.Role + ":" + m.Content + "|"
	}
	return key
}
