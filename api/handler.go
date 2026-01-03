package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"agent/config"
	"agent/philosopher"

	"github.com/rs/zerolog/log"
)

// Server HTTP API 服务器
type Server struct {
	model           *config.ChatModel
	faultTolerant   *config.FaultTolerantModel
	emotionAnalyzer *philosopher.EmotionAnalyzer
	deduplicator    *philosopher.ContentDeduplicator
	cache           *config.ResponseCache

	// 会话管理
	sessions     map[string]*Session
	sessionMutex sync.RWMutex

	// 辩论管理
	debates     map[string]*DebateSession
	debateMutex sync.RWMutex
}

// Session 用户会话
type Session struct {
	ID           string
	Messages     []config.Message
	Philosopher  philosopher.PhilosopherType
	LastActivity time.Time
}

// DebateSession 辩论会话
type DebateSession struct {
	ID           string                     `json:"id"`
	Status       DebateStatus               `json:"status"`
	Topic        string                     `json:"topic"`
	CurrentPhase philosopher.DebatePhase    `json:"current_phase"`
	Records      []philosopher.DebateRecord `json:"records"`
	StartTime    time.Time                  `json:"start_time"`
	EndTime      *time.Time                 `json:"end_time,omitempty"`
	Error        string                     `json:"error,omitempty"`
}

// DebateStatus 辩论状态
type DebateStatus string

const (
	DebateStatusPending   DebateStatus = "pending"   // 等待开始
	DebateStatusRunning   DebateStatus = "running"   // 进行中
	DebateStatusCompleted DebateStatus = "completed" // 已完成
	DebateStatusFailed    DebateStatus = "failed"    // 失败
)

// NewServer 创建 API 服务器
func NewServer(model *config.ChatModel) *Server {
	return &Server{
		model:           model,
		emotionAnalyzer: philosopher.NewEmotionAnalyzer(model),
		deduplicator:    philosopher.NewContentDeduplicator(0.7),
		cache:           config.NewResponseCache(100, 30*time.Minute),
		sessions:        make(map[string]*Session),
		debates:         make(map[string]*DebateSession),
	}
}

// NewServerWithFaultTolerant 创建带容错的 API 服务器
func NewServerWithFaultTolerant(ft *config.FaultTolerantModel) *Server {
	return &Server{
		faultTolerant:   ft,
		emotionAnalyzer: philosopher.NewEmotionAnalyzer(nil),
		deduplicator:    philosopher.NewContentDeduplicator(0.7),
		cache:           config.NewResponseCache(100, 30*time.Minute),
		sessions:        make(map[string]*Session),
		debates:         make(map[string]*DebateSession),
	}
}

// RegisterRoutes 注册路由
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// 一对一对话
	mux.HandleFunc("/api/chat", s.handleChat)

	// Agent 对话（带工具、反思）
	mux.HandleFunc("/api/agent/chat", s.handleAgentChat)

	// 主持人驱动的讨论
	mux.HandleFunc("/api/agent/discussion", s.handleAgentDiscussion)

	// 辩论模式
	mux.HandleFunc("/api/debate/start", s.handleDebateStart)
	mux.HandleFunc("/api/debate/status", s.handleDebateStatus)

	// 哲学家列表
	mux.HandleFunc("/api/philosophers", s.handlePhilosophers)

	// 健康检查
	mux.HandleFunc("/api/health", s.handleHealth)

	// 静态文件服务
	mux.HandleFunc("/", s.handleStatic)
}

// handleStatic 处理静态文件请求
func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	// 默认返回 index.html
	http.ServeFile(w, r, "static/index.html")
}

// ==================== 一对一对话 ====================

// ChatRequest 对话请求
type ChatRequest struct {
	SessionID   string                      `json:"session_id"`
	Message     string                      `json:"message"`
	Philosopher philosopher.PhilosopherType `json:"philosopher"`
}

// ChatResponse 对话响应
type ChatResponse struct {
	Response     string                   `json:"response"`
	Philosopher  string                   `json:"philosopher"`
	EmotionLevel philosopher.EmotionLevel `json:"emotion_level"`
	CriticalHit  bool                     `json:"critical_hit"` // 是否触发毒舌标签
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 获取或创建会话
	session := s.getOrCreateSession(req.SessionID, req.Philosopher)

	// 分析情绪
	emotionLevel := s.emotionAnalyzer.Analyze(req.Message)

	// 添加用户消息
	session.Messages = append(session.Messages, config.Message{
		Role:    "user",
		Content: req.Message,
	})

	// 创建哲学家并获取响应
	p := philosopher.NewPhilosopher(req.Philosopher, s.model)
	response, err := p.Chat(session.Messages, emotionLevel)
	if err != nil {
		log.Error().Err(err).Msg("Chat failed")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 检查是否重复，如果重复则提高 temperature 重新生成
	if s.deduplicator.IsDuplicate(response) {
		// 这里可以实现重新生成逻辑
		log.Warn().Msg("Detected duplicate response")
	}
	s.deduplicator.AddResponse(response)

	// 添加助手消息
	session.Messages = append(session.Messages, config.Message{
		Role:    "assistant",
		Content: response,
	})
	session.LastActivity = time.Now()

	// 检查是否有毒舌标签
	criticalHit := containsCriticalHit(response)

	resp := ChatResponse{
		Response:     response,
		Philosopher:  p.Name,
		EmotionLevel: emotionLevel,
		CriticalHit:  criticalHit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func containsCriticalHit(response string) bool {
	tags := []string{
		"[致命追问!]", "[权力意志!]", "[绝对命令!]",
		"[潜意识暴露!]", "[逻辑利刃!]", "[Critical Hit!]",
	}
	for _, tag := range tags {
		if containsSubstring(response, tag) {
			return true
		}
	}
	return false
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (s *Server) getOrCreateSession(id string, pType philosopher.PhilosopherType) *Session {
	s.sessionMutex.Lock()
	defer s.sessionMutex.Unlock()

	if session, ok := s.sessions[id]; ok {
		return session
	}

	session := &Session{
		ID:           id,
		Messages:     []config.Message{},
		Philosopher:  pType,
		LastActivity: time.Now(),
	}
	s.sessions[id] = session
	return session
}

// ==================== 辩论模式 ====================

// DebateStartRequest 开始辩论请求
type DebateStartRequest struct {
	Topic           string                                 `json:"topic"`
	ProStance       string                                 `json:"pro_stance"`
	ConStance       string                                 `json:"con_stance"`
	ProPhilosophers []philosopher.PhilosopherType          `json:"pro_philosophers"`
	ConPhilosophers []philosopher.PhilosopherType          `json:"con_philosophers"`
	ForcedStances   map[philosopher.PhilosopherType]string `json:"forced_stances,omitempty"`
	Async           bool                                   `json:"async,omitempty"` // 是否异步执行
}

// DebateResponse 辩论响应
type DebateResponse struct {
	ID           string                     `json:"id,omitempty"`
	Status       DebateStatus               `json:"status"`
	Topic        string                     `json:"topic,omitempty"`
	CurrentPhase philosopher.DebatePhase    `json:"current_phase,omitempty"`
	Records      []philosopher.DebateRecord `json:"records,omitempty"`
	Error        string                     `json:"error,omitempty"`
}

func (s *Server) handleDebateStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DebateStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 创建辩论配置
	debateConfig := &philosopher.DebateConfig{
		Topic:           req.Topic,
		ProStance:       req.ProStance,
		ConStance:       req.ConStance,
		ProPhilosophers: req.ProPhilosophers,
		ConPhilosophers: req.ConPhilosophers,
		ForcedStances:   req.ForcedStances,
	}

	// 生成辩论 ID
	debateID := generateDebateID()

	// 异步模式
	if req.Async {
		// 创建辩论会话
		session := &DebateSession{
			ID:           debateID,
			Status:       DebateStatusPending,
			Topic:        req.Topic,
			CurrentPhase: philosopher.PhaseOpening,
			Records:      []philosopher.DebateRecord{},
			StartTime:    time.Now(),
		}

		s.debateMutex.Lock()
		s.debates[debateID] = session
		s.debateMutex.Unlock()

		// 异步执行辩论
		go s.runDebateAsync(debateID, debateConfig)

		// 立即返回辩论 ID
		resp := DebateResponse{
			ID:     debateID,
			Status: DebateStatusPending,
			Topic:  req.Topic,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// 同步模式（原有逻辑）
	engine := philosopher.NewDebateEngine(debateConfig, s.model)
	result, err := engine.Run()
	if err != nil {
		log.Error().Err(err).Msg("Debate failed")
		resp := DebateResponse{
			Status: DebateStatusFailed,
			Error:  err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := DebateResponse{
		Status:  DebateStatusCompleted,
		Records: result.Records,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// runDebateAsync 异步执行辩论
func (s *Server) runDebateAsync(debateID string, config *philosopher.DebateConfig) {
	// 更新状态为运行中
	s.debateMutex.Lock()
	session := s.debates[debateID]
	session.Status = DebateStatusRunning
	s.debateMutex.Unlock()

	// 创建辩论引擎
	engine := philosopher.NewDebateEngine(config, s.model)

	// 设置发言回调，实时更新记录
	engine.SetOnSpeech(func(speaker string, content string, phase philosopher.DebatePhase) {
		s.debateMutex.Lock()
		session.CurrentPhase = phase
		session.Records = append(session.Records, philosopher.DebateRecord{
			SpeakerName: speaker,
			Content:     content,
			Phase:       phase,
		})
		s.debateMutex.Unlock()
	})

	// 运行辩论
	result, err := engine.Run()

	// 更新最终状态
	s.debateMutex.Lock()
	now := time.Now()
	session.EndTime = &now
	if err != nil {
		session.Status = DebateStatusFailed
		session.Error = err.Error()
		log.Error().Err(err).Str("debate_id", debateID).Msg("Async debate failed")
	} else {
		session.Status = DebateStatusCompleted
		session.Records = result.Records
	}
	s.debateMutex.Unlock()
}

func (s *Server) handleDebateStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取辩论 ID
	debateID := r.URL.Query().Get("id")
	if debateID == "" {
		http.Error(w, "Missing debate id", http.StatusBadRequest)
		return
	}

	// 查找辩论会话
	s.debateMutex.RLock()
	session, ok := s.debates[debateID]
	s.debateMutex.RUnlock()

	if !ok {
		http.Error(w, "Debate not found", http.StatusNotFound)
		return
	}

	resp := DebateResponse{
		ID:           session.ID,
		Status:       session.Status,
		Topic:        session.Topic,
		CurrentPhase: session.CurrentPhase,
		Records:      session.Records,
		Error:        session.Error,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// generateDebateID 生成辩论 ID
func generateDebateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

// randomString 生成随机字符串
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}

// ==================== 哲学家列表 ====================

// PhilosopherInfo 哲学家信息
type PhilosopherInfo struct {
	Type        philosopher.PhilosopherType `json:"type"`
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	Quotes      []string                    `json:"quotes"`
}

func (s *Server) handlePhilosophers(w http.ResponseWriter, r *http.Request) {
	prompts := philosopher.GetPhilosopherPrompts()

	var philosophers []PhilosopherInfo
	for pType, prompt := range prompts {
		philosophers = append(philosophers, PhilosopherInfo{
			Type:        pType,
			Name:        prompt.Name,
			Description: prompt.CoreIdentity,
			Quotes:      prompt.FamousQuotes,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(philosophers)
}

// ==================== Agent 对话 ====================

// AgentChatRequest Agent 对话请求
type AgentChatRequest struct {
	SessionID        string                      `json:"session_id"`
	Message          string                      `json:"message"`
	Philosopher      philosopher.PhilosopherType `json:"philosopher"`
	EnableTools      bool                        `json:"enable_tools"`
	EnableReflection bool                        `json:"enable_reflection"`
}

// AgentChatResponse Agent 对话响应
type AgentChatResponse struct {
	Response         string                        `json:"response"`
	Philosopher      string                        `json:"philosopher"`
	EmotionLevel     philosopher.EmotionLevel      `json:"emotion_level"`
	ToolResults      []philosopher.ToolResult      `json:"tool_results,omitempty"`
	ReflectionResult *philosopher.ReflectionResult `json:"reflection_result,omitempty"`
	AgentEnabled     bool                          `json:"agent_enabled"`
}

func (s *Server) handleAgentChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AgentChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 创建 Agent 配置
	agentConfig := &philosopher.AgentConfig{
		EnableTools:      req.EnableTools,
		EnableReflection: req.EnableReflection,
		EnableRefinement: false,
		MaxToolCalls:     3,
	}

	// 创建 Agent
	agent := philosopher.NewAgent(req.Philosopher, s.model, agentConfig)

	// 获取历史消息
	session := s.getOrCreateSession(req.SessionID, req.Philosopher)

	// 调用 Agent
	result, err := agent.Chat(req.Message, session.Messages)
	if err != nil {
		log.Error().Err(err).Msg("Agent chat failed")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 更新会话
	session.Messages = append(session.Messages, config.Message{
		Role:    "user",
		Content: req.Message,
	})
	session.Messages = append(session.Messages, config.Message{
		Role:    "assistant",
		Content: result.Content,
	})
	session.LastActivity = time.Now()

	resp := AgentChatResponse{
		Response:         result.Content,
		Philosopher:      agent.Name,
		EmotionLevel:     result.EmotionLevel,
		ToolResults:      result.ToolResults,
		ReflectionResult: result.ReflectionResult,
		AgentEnabled:     true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ==================== 主持人驱动讨论 ====================

// AgentDiscussionRequest 主持人讨论请求
type AgentDiscussionRequest struct {
	Topic        string                        `json:"topic"`
	Participants []philosopher.PhilosopherType `json:"participants"`
	MaxRounds    int                           `json:"max_rounds"`
}

// AgentDiscussionResponse 主持人讨论响应
type AgentDiscussionResponse struct {
	Status    string                     `json:"status"`
	Topic     string                     `json:"topic"`
	Records   []philosopher.DebateRecord `json:"records"`
	Decisions []ModeratorDecisionInfo    `json:"decisions,omitempty"`
	Error     string                     `json:"error,omitempty"`
}

// ModeratorDecisionInfo 主持人决策信息
type ModeratorDecisionInfo struct {
	Action      string `json:"action"`
	NextSpeaker string `json:"next_speaker"`
	Reason      string `json:"reason"`
}

func (s *Server) handleAgentDiscussion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AgentDiscussionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 默认参与者
	if len(req.Participants) == 0 {
		req.Participants = []philosopher.PhilosopherType{
			philosopher.TakamatsuTomori,
			philosopher.ChihayaAnon,
		}
	}

	// 默认轮数
	if req.MaxRounds == 0 {
		req.MaxRounds = 6
	}

	// 创建成员
	members := make(map[philosopher.PhilosopherType]*philosopher.Philosopher)
	for _, pType := range req.Participants {
		members[pType] = philosopher.NewPhilosopher(pType, s.model)
	}

	// 创建主持人 Agent
	moderator := philosopher.NewModeratorAgent(s.model, req.Topic, members)
	moderator.SetMaxRounds(req.MaxRounds)

	// 收集决策信息
	var decisions []ModeratorDecisionInfo
	moderator.SetOnDecision(func(decision *philosopher.ModeratorDecision) {
		decisions = append(decisions, ModeratorDecisionInfo{
			Action:      string(decision.Action),
			NextSpeaker: string(decision.NextSpeaker),
			Reason:      decision.Reason,
		})
	})

	// 运行讨论
	result, err := moderator.RunAutonomous(nil)
	if err != nil {
		log.Error().Err(err).Msg("Agent discussion failed")
		resp := AgentDiscussionResponse{
			Status: "failed",
			Error:  err.Error(),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := AgentDiscussionResponse{
		Status:    "completed",
		Topic:     req.Topic,
		Records:   result.Records,
		Decisions: decisions,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ==================== 健康检查 ====================

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "mygo-chat",
	})
}

// Start 启动服务器
func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)

	// 添加 CORS 中间件
	handler := corsMiddleware(mux)

	log.Info().Str("addr", addr).Msg("Starting API server")
	return http.ListenAndServe(addr, handler)
}

// corsMiddleware CORS 中间件
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
