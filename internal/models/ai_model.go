package models

// AiUsage 对话 token 用量统计。
type AiUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// AiModelStats 模型聚合统计信息。
type AiModelStats struct {
	Model           string  `json:"model"`
	TotalCalls      int64   `json:"totalCalls"`
	AvgTokens       float64 `json:"avgTokens"`
	AvgFirstTokenMs float64 `json:"avgFirstTokenMs"`
}

// AiModelOption 模型下拉选项。
type AiModelOption struct {
	Value     string `json:"value"`
	Label     string `json:"label"`
	Source    string `json:"source,omitempty"`
	Available bool   `json:"available"`
	Status    string `json:"status,omitempty"`
	Hint      string `json:"hint,omitempty"`
}

// AiModelListResponse 模型列表响应，支持回退状态提示。
type AiModelListResponse struct {
	Models   []AiModelOption `json:"models"`
	Fallback bool            `json:"fallback"`
	Message  string          `json:"message,omitempty"`
}

// AiChatMessage 前端传入的聊天消息
type AiChatMessage struct {
	Role    string `json:"role"`    // system / user / assistant
	Content string `json:"content"` // 消息内容
}

// AiChatRequest 前端发起的对话请求
type AiChatRequest struct {
	Messages    []AiChatMessage `json:"messages"`
	Model       string          `json:"model,omitempty"`       // 可选，覆盖默认模型
	Temperature *float32        `json:"temperature,omitempty"` // 可选
	MaxTokens   *int            `json:"maxTokens,omitempty"`   // 可选
	Stream      bool            `json:"stream"`                // 是否流式
	Mode        string          `json:"mode,omitempty"`        // single / compare
	SessionId   string          `json:"sessionId,omitempty"`   // 流式会话标识
	RoundId     string          `json:"roundId,omitempty"`     // 同轮对比问题标识
}

// AiChatHistoryMessage 历史会话中的单条消息。
type AiChatHistoryMessage struct {
	Role             string        `json:"role"`
	Content          string        `json:"content"`
	ReasoningContent *string       `json:"reasoningContent,omitempty"`
	Model            string        `json:"model,omitempty"`
	Usage            *AiUsage      `json:"usage,omitempty"`
	DurationMs       *int64        `json:"durationMs,omitempty"`
	FirstTokenMs     *int64        `json:"firstTokenMs,omitempty"`
	ModelStats       *AiModelStats `json:"modelStats,omitempty"`
	Error            string        `json:"error,omitempty"`
}

// AiChatSessionSummary 会话列表摘要。
type AiChatSessionSummary struct {
	SessionId        string `json:"sessionId"`
	Title            string `json:"title"`
	Preview          string `json:"preview,omitempty"`
	Model            string `json:"model,omitempty"`
	RoundCount       int64  `json:"roundCount"`
	PromptTokens     int64  `json:"promptTokens"`
	CompletionTokens int64  `json:"completionTokens"`
	TotalTokens      int64  `json:"totalTokens"`
	AvgDurationMs    int64  `json:"avgDurationMs"`
	AvgFirstTokenMs  int64  `json:"avgFirstTokenMs"`
	LastActiveTime   string `json:"lastActiveTime"`
}

// AiChatSessionDetail 会话详情。
type AiChatSessionDetail struct {
	SessionId string                 `json:"sessionId"`
	Messages  []AiChatHistoryMessage `json:"messages"`
}

// AiChatStreamChunk 流式推送给前端的单条消息块
type AiChatStreamChunk struct {
	SessionId        string        `json:"sessionId"`
	RoundId          string        `json:"roundId,omitempty"`
	Model            string        `json:"model,omitempty"`
	Content          string        `json:"content"`
	ReasoningContent *string       `json:"reasoningContent,omitempty"` // 思维链内容
	Usage            *AiUsage      `json:"usage,omitempty"`
	DurationMs       *int64        `json:"durationMs,omitempty"`
	FirstTokenMs     *int64        `json:"firstTokenMs,omitempty"`
	ModelStats       *AiModelStats `json:"modelStats,omitempty"`
	Done             bool          `json:"done"`
	Error            string        `json:"error,omitempty"`
}

// AiChatResponse 非流式对话完整响应
type AiChatResponse struct {
	Content          string        `json:"content"`
	ReasoningContent *string       `json:"reasoningContent,omitempty"`
	Model            string        `json:"model"`
	Usage            AiUsage       `json:"usage"`
	DurationMs       int64         `json:"durationMs"`
	FirstTokenMs     *int64        `json:"firstTokenMs,omitempty"`
	ModelStats       *AiModelStats `json:"modelStats,omitempty"`
}
