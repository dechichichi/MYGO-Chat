package philosopher

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// MemoryManager 记忆管理器
type MemoryManager struct {
	store     MemoryStore
	shortTerm *ShortTermMemory
}

// ShortTermMemory 短期记忆管理器
type ShortTermMemory struct {
	memories   []*Memory
	maxSize    int
	expiration time.Duration
}

// NewMemoryManager 创建记忆管理器
func NewMemoryManager(store MemoryStore) *MemoryManager {
	return &MemoryManager{
		store: store,
		shortTerm: &ShortTermMemory{
			memories:   []*Memory{},
			maxSize:    20,               // 保留最近20条短期记忆
			expiration: 30 * time.Minute, // 30分钟过期
		},
	}
}

// SaveConversation 保存对话记忆
func (m *MemoryManager) SaveConversation(sessionID, character, userMessage, response string) error {
	// 保存用户消息
	userMemory := &Memory{
		SessionID: sessionID,
		Character: character,
		Content:   fmt.Sprintf("用户: %s", userMessage),
		Type:      "short_term",
		ExpiresAt: time.Now().Add(m.shortTerm.expiration),
	}

	// 保存AI回复
	aiMemory := &Memory{
		SessionID: sessionID,
		Character: character,
		Content:   fmt.Sprintf("%s: %s", character, response),
		Type:      "short_term",
		ExpiresAt: time.Now().Add(m.shortTerm.expiration),
	}

	// 保存到数据库
	if err := m.store.Save(userMemory); err != nil {
		return fmt.Errorf("failed to save user memory: %w", err)
	}
	if err := m.store.Save(aiMemory); err != nil {
		return fmt.Errorf("failed to save AI memory: %w", err)
	}

	// 更新短期记忆
	m.shortTerm.memories = append(m.shortTerm.memories, userMemory, aiMemory)
	m.cleanShortTermMemory()

	// 检查是否需要保存为长期记忆
	if m.shouldSaveAsLongTerm(userMessage, response) {
		longTermMemory := &Memory{
			SessionID: sessionID,
			Character: character,
			Content:   fmt.Sprintf("用户: %s | %s: %s", userMessage, character, response),
			Type:      "long_term",
			Metadata:  m.extractMetadata(userMessage, response),
		}
		if err := m.store.Save(longTermMemory); err != nil {
			log.Warn().Err(err).Msg("Failed to save long-term memory")
		}
	}

	return nil
}

// RecallMemory 回忆相关记忆
func (m *MemoryManager) RecallMemory(sessionID, character, query string, limit int) ([]*Memory, error) {
	var results []*Memory

	// 1. 从短期记忆搜索
	shortTermResults := m.searchShortTermMemory(query)
	results = append(results, shortTermResults...)

	// 2. 从长期记忆搜索
	longTermResults, err := m.store.SearchByKeyword(sessionID, query, limit)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to search long-term memory")
	} else {
		results = append(results, longTermResults...)
	}

	// 3. 语义搜索（如果关键词搜索结果不足）
	if len(results) < limit/2 {
		semanticResults, err := m.store.SearchSimilar(query, limit-len(results))
		if err != nil {
			log.Warn().Err(err).Msg("Failed to perform semantic search")
		} else {
			results = append(results, semanticResults...)
		}
	}

	// 去重和排序
	results = m.deduplicateAndSort(results)

	// 限制结果数量
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// GetConversationHistory 获取对话历史
func (m *MemoryManager) GetConversationHistory(sessionID, character string, limit int) ([]*Memory, error) {
	// 获取短期记忆（最近对话）
	shortTerm := m.getRecentShortTermMemory()

	// 获取长期记忆（历史对话）
	longTerm, err := m.store.GetRecentConversations(sessionID, limit/2)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to get long-term history")
	}

	// 合并结果
	results := append(shortTerm, longTerm...)
	results = m.deduplicateAndSort(results)

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// SaveImportantMemory 保存重要信息到长期记忆
func (m *MemoryManager) SaveImportantMemory(sessionID, character, content, metadata string) error {
	memory := &Memory{
		SessionID: sessionID,
		Character: character,
		Content:   content,
		Type:      "long_term",
		Metadata:  metadata,
	}

	return m.store.Save(memory)
}

// GetMemoryStats 获取记忆统计
func (m *MemoryManager) GetMemoryStats(sessionID string) (*MemoryStats, error) {
	return m.store.GetMemoryStats(sessionID)
}

// Cleanup 清理过期记忆
func (m *MemoryManager) Cleanup() error {
	// 清理短期过期记忆
	m.cleanShortTermMemory()

	// 清理数据库过期记忆
	return m.store.DeleteExpired()
}

// 内部方法

func (m *MemoryManager) searchShortTermMemory(query string) []*Memory {
	var results []*Memory
	keywords := extractKeywords(query)

	for _, memory := range m.shortTerm.memories {
		for _, keyword := range keywords {
			if strings.Contains(memory.Content, keyword) {
				results = append(results, memory)
				break
			}
		}
	}

	return results
}

func (m *MemoryManager) getRecentShortTermMemory() []*Memory {
	// 返回最新的短期记忆
	count := len(m.shortTerm.memories)
	if count > m.shortTerm.maxSize/2 {
		return m.shortTerm.memories[count-min(count, m.shortTerm.maxSize/2):]
	}
	return m.shortTerm.memories
}

func (m *MemoryManager) cleanShortTermMemory() {
	now := time.Now()
	var validMemories []*Memory

	for _, memory := range m.shortTerm.memories {
		if memory.ExpiresAt.IsZero() || memory.ExpiresAt.After(now) {
			validMemories = append(validMemories, memory)
		}
	}

	// 限制大小
	if len(validMemories) > m.shortTerm.maxSize {
		validMemories = validMemories[len(validMemories)-m.shortTerm.maxSize:]
	}

	m.shortTerm.memories = validMemories
}

func (m *MemoryManager) shouldSaveAsLongTerm(userMessage, response string) bool {
	// 判断是否应该保存为长期记忆的简单规则

	// 包含重要关键词
	importantKeywords := []string{
		"喜欢", "不喜欢", "爱好", "梦想", "目标",
		"重要", "难忘", "特别", "第一次", "改变",
		"家人", "朋友", "爱情", "工作", "学习",
		"成功", "失败", "困难", "挑战", "成长",
	}

	content := userMessage + " " + response
	for _, keyword := range importantKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}

	// 对话长度超过阈值
	if len(content) > 100 {
		return true
	}

	// 包含情感表达
	emotionWords := []string{
		"开心", "难过", "生气", "惊讶", "害怕",
		"感动", "失望", "希望", "担心", "期待",
	}

	for _, word := range emotionWords {
		if strings.Contains(content, word) {
			return true
		}
	}

	return false
}

func (m *MemoryManager) extractMetadata(userMessage, response string) string {
	metadata := map[string]interface{}{
		"user_message_length": len(userMessage),
		"response_length":     len(response),
		"has_emotion":         m.containsEmotion(userMessage + " " + response),
		"timestamp":           time.Now().Format(time.RFC3339),
	}

	data, _ := json.Marshal(metadata)
	return string(data)
}

func (m *MemoryManager) containsEmotion(text string) bool {
	emotionWords := []string{
		"开心", "高兴", "快乐", "幸福", "满意",
		"难过", "伤心", "痛苦", "失望", "沮丧",
		"生气", "愤怒", "恼火", "不满", "烦躁",
		"惊讶", "震惊", "意外", "惊喜", "害怕",
	}

	for _, word := range emotionWords {
		if strings.Contains(text, word) {
			return true
		}
	}
	return false
}

func (m *MemoryManager) deduplicateAndSort(memories []*Memory) []*Memory {
	seen := make(map[string]bool)
	var unique []*Memory

	for _, memory := range memories {
		if !seen[memory.ID] {
			seen[memory.ID] = true
			unique = append(unique, memory)
		}
	}

	// 按创建时间排序（最新的在前）
	for i, j := 0, len(unique)-1; i < j; i, j = i+1, j-1 {
		unique[i], unique[j] = unique[j], unique[i]
	}

	return unique
}

// 工具函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
