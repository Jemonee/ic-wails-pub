package models

import pkgmodels "ic-wails/pkg/models"

// AiChatRecordModel 持久化单次模型回答，支持后续统计与对比分析。
type AiChatRecordModel struct {
	*pkgmodels.BaseModel
	ChatMode            string  `gorm:"type:varchar(32);index;not null;default:'single'" json:"chatMode"`
	RoundId             string  `gorm:"type:varchar(128);index" json:"roundId"`
	SessionId           string  `gorm:"type:varchar(128);index;not null" json:"sessionId"`
	Model               string  `gorm:"type:varchar(255);index;not null" json:"model"`
	UserContent         string  `gorm:"type:text;not null" json:"userContent"`
	AssistantContent    string  `gorm:"type:text;not null" json:"assistantContent"`
	ReasoningContent    *string `gorm:"type:text" json:"reasoningContent"`
	PromptTokens        int     `gorm:"default:0" json:"promptTokens"`
	CompletionTokens    int     `gorm:"default:0" json:"completionTokens"`
	TotalTokens         int     `gorm:"default:0" json:"totalTokens"`
	FirstTokenLatencyMs *int64  `json:"firstTokenLatencyMs"`
	DurationMs          *int64  `json:"durationMs"`
	Stream              int     `gorm:"default:1" json:"stream"`
	ErrorMessage        *string `gorm:"type:text" json:"errorMessage"`
}
