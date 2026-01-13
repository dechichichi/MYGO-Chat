package philosopher

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Memory 记忆结构
type Memory struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"` // 用户会话
	Character string    `json:"character"`  // 角色
	Content   string    `json:"content"`    // 记忆内容
	Type      string    `json:"type"`       // short_term / long_term
	Embedding []float32 `json:"embedding"`  // 向量嵌入（可选）
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"` // 短期记忆过期时间
	Metadata  string    `json:"metadata"`   // 额外元数据
}

// MemoryStore 记忆存储接口
type MemoryStore interface {
	// 基础操作
	Save(memory *Memory) error
	GetByID(id string) (*Memory, error)
	GetBySession(sessionID, character string, limit int) ([]*Memory, error)
	DeleteByID(id string) error
	DeleteExpired() error

	// 检索操作
	SearchByKeyword(sessionID, keyword string, limit int) ([]*Memory, error)
	SearchSimilar(content string, topK int) ([]*Memory, error) // 语义检索
	GetRecentConversations(sessionID string, limit int) ([]*Memory, error)

	// 统计操作
	GetMemoryStats(sessionID string) (*MemoryStats, error)
	Close() error
}

// MemoryStats 记忆统计
type MemoryStats struct {
	TotalMemories  int `json:"total_memories"`
	ShortTermCount int `json:"short_term_count"`
	LongTermCount  int `json:"long_term_count"`
	RecentActivity int `json:"recent_activity"` // 最近7天活动
}

// SQLiteMemoryStore SQLite实现
type SQLiteMemoryStore struct {
	db *sql.DB
}

// NewSQLiteMemoryStore 创建SQLite存储实例
func NewSQLiteMemoryStore(dbPath string) (*SQLiteMemoryStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteMemoryStore{db: db}
	if err := store.initTables(); err != nil {
		return nil, fmt.Errorf("failed to init tables: %w", err)
	}

	return store, nil
}

// initTables 初始化数据库表
func (s *SQLiteMemoryStore) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS memories (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			character TEXT NOT NULL,
			content TEXT NOT NULL,
			type TEXT NOT NULL,
			embedding BLOB,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME,
			metadata TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_session_character ON memories(session_id, character)`,
		`CREATE INDEX IF NOT EXISTS idx_created_at ON memories(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_expires_at ON memories(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_type ON memories(type)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query %s: %w", query, err)
		}
	}

	return nil
}

// Save 保存记忆
func (s *SQLiteMemoryStore) Save(memory *Memory) error {
	if memory.ID == "" {
		memory.ID = generateMemoryID()
	}
	if memory.CreatedAt.IsZero() {
		memory.CreatedAt = time.Now()
	}

	// 序列化向量嵌入
	var embeddingBlob []byte
	if len(memory.Embedding) > 0 {
		var err error
		embeddingBlob, err = json.Marshal(memory.Embedding)
		if err != nil {
			return fmt.Errorf("failed to marshal embedding: %w", err)
		}
	}

	query := `INSERT OR REPLACE INTO memories 
		(id, session_id, character, content, type, embedding, created_at, expires_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		memory.ID, memory.SessionID, memory.Character, memory.Content,
		memory.Type, embeddingBlob, memory.CreatedAt, memory.ExpiresAt,
		memory.Metadata)

	return err
}

// GetBySession 获取会话记忆
func (s *SQLiteMemoryStore) GetBySession(sessionID, character string, limit int) ([]*Memory, error) {
	query := `SELECT id, session_id, character, content, type, embedding, created_at, expires_at, metadata
		FROM memories 
		WHERE session_id = ? AND character = ? 
		ORDER BY created_at DESC 
		LIMIT ?`

	rows, err := s.db.Query(query, sessionID, character, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanMemories(rows)
}

// SearchByKeyword 关键词搜索
func (s *SQLiteMemoryStore) SearchByKeyword(sessionID, keyword string, limit int) ([]*Memory, error) {
	query := `SELECT id, session_id, character, content, type, embedding, created_at, expires_at, metadata
		FROM memories 
		WHERE session_id = ? AND content LIKE ? 
		ORDER BY created_at DESC 
		LIMIT ?`

	rows, err := s.db.Query(query, sessionID, "%"+keyword+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanMemories(rows)
}

// SearchSimilar 语义相似度搜索（简化版，基于关键词匹配）
func (s *SQLiteMemoryStore) SearchSimilar(content string, topK int) ([]*Memory, error) {
	// 提取关键词进行简单匹配
	keywords := extractKeywords(content)
	if len(keywords) == 0 {
		return []*Memory{}, nil
	}

	// 构建OR条件
	conditions := []string{}
	args := []interface{}{}
	for _, keyword := range keywords {
		conditions = append(conditions, "content LIKE ?")
		args = append(args, "%"+keyword+"%")
	}

	query := fmt.Sprintf(`SELECT id, session_id, character, content, type, embedding, created_at, expires_at, metadata
		FROM memories 
		WHERE (%s) 
		ORDER BY created_at DESC 
		LIMIT ?`, strings.Join(conditions, " OR "))

	args = append(args, topK)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanMemories(rows)
}

// GetRecentConversations 获取最近对话
func (s *SQLiteMemoryStore) GetRecentConversations(sessionID string, limit int) ([]*Memory, error) {
	query := `SELECT id, session_id, character, content, type, embedding, created_at, expires_at, metadata
		FROM memories 
		WHERE session_id = ? AND type = 'short_term'
		ORDER BY created_at DESC 
		LIMIT ?`

	rows, err := s.db.Query(query, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanMemories(rows)
}

// GetMemoryStats 获取记忆统计
func (s *SQLiteMemoryStore) GetMemoryStats(sessionID string) (*MemoryStats, error) {
	stats := &MemoryStats{}

	// 总记忆数
	err := s.db.QueryRow(`SELECT COUNT(*) FROM memories WHERE session_id = ?`, sessionID).
		Scan(&stats.TotalMemories)
	if err != nil {
		return nil, err
	}

	// 短期记忆数
	err = s.db.QueryRow(`SELECT COUNT(*) FROM memories WHERE session_id = ? AND type = 'short_term'`, sessionID).
		Scan(&stats.ShortTermCount)
	if err != nil {
		return nil, err
	}

	// 长期记忆数
	err = s.db.QueryRow(`SELECT COUNT(*) FROM memories WHERE session_id = ? AND type = 'long_term'`, sessionID).
		Scan(&stats.LongTermCount)
	if err != nil {
		return nil, err
	}

	// 最近7天活动
	err = s.db.QueryRow(`SELECT COUNT(*) FROM memories WHERE session_id = ? AND created_at > datetime('now', '-7 days')`, sessionID).
		Scan(&stats.RecentActivity)

	return stats, err
}

// DeleteExpired 删除过期记忆
func (s *SQLiteMemoryStore) DeleteExpired() error {
	_, err := s.db.Exec(`DELETE FROM memories WHERE expires_at IS NOT NULL AND expires_at < datetime('now')`)
	return err
}

// Close 关闭数据库连接
func (s *SQLiteMemoryStore) Close() error {
	return s.db.Close()
}

// 辅助函数
func (s *SQLiteMemoryStore) scanMemories(rows *sql.Rows) ([]*Memory, error) {
	var memories []*Memory

	for rows.Next() {
		var memory Memory
		var embeddingBlob []byte

		err := rows.Scan(
			&memory.ID, &memory.SessionID, &memory.Character, &memory.Content,
			&memory.Type, &embeddingBlob, &memory.CreatedAt, &memory.ExpiresAt,
			&memory.Metadata)
		if err != nil {
			return nil, err
		}

		// 反序列化向量嵌入
		if len(embeddingBlob) > 0 {
			if err := json.Unmarshal(embeddingBlob, &memory.Embedding); err != nil {
				return nil, fmt.Errorf("failed to unmarshal embedding: %w", err)
			}
		}

		memories = append(memories, &memory)
	}

	return memories, nil
}

func generateMemoryID() string {
	return fmt.Sprintf("mem_%d_%s", time.Now().UnixNano(), randomString(8))
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}

func extractKeywords(content string) []string {
	// 简单的关键词提取（实际项目中可以使用更复杂的NLP技术）
	words := strings.Fields(content)
	var keywords []string

	// 过滤停用词和短词
	stopWords := map[string]bool{
		"的": true, "了": true, "在": true, "是": true, "我": true,
		"你": true, "他": true, "她": true, "它": true, "这": true,
		"那": true, "和": true, "与": true, "或": true, "但": true,
	}

	for _, word := range words {
		if len(word) > 1 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	// 限制关键词数量
	if len(keywords) > 5 {
		return keywords[:5]
	}
	return keywords
}

// 实现其他接口方法
func (s *SQLiteMemoryStore) GetByID(id string) (*Memory, error) {
	query := `SELECT id, session_id, character, content, type, embedding, created_at, expires_at, metadata
		FROM memories WHERE id = ?`

	row := s.db.QueryRow(query, id)
	var memory Memory
	var embeddingBlob []byte

	err := row.Scan(
		&memory.ID, &memory.SessionID, &memory.Character, &memory.Content,
		&memory.Type, &embeddingBlob, &memory.CreatedAt, &memory.ExpiresAt,
		&memory.Metadata)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// 反序列化向量嵌入
	if len(embeddingBlob) > 0 {
		if err := json.Unmarshal(embeddingBlob, &memory.Embedding); err != nil {
			return nil, fmt.Errorf("failed to unmarshal embedding: %w", err)
		}
	}

	return &memory, nil
}

func (s *SQLiteMemoryStore) DeleteByID(id string) error {
	_, err := s.db.Exec("DELETE FROM memories WHERE id = ?", id)
	return err
}
