package models

import (
	"time"

	pkgmodels "ic-wails/pkg/models"
)

// AiModelCacheModel 按当前 AI 配置缓存可用模型列表。
type AiModelCacheModel struct {
	*pkgmodels.BaseModel
	ConfigFingerprint string     `gorm:"type:varchar(128);index:idx_ai_model_cache_fp_model,priority:1;not null" json:"configFingerprint"`
	Model             string     `gorm:"type:varchar(255);index:idx_ai_model_cache_fp_model,priority:2;not null" json:"model"`
	Label             string     `gorm:"type:varchar(255);not null" json:"label"`
	Source            string     `gorm:"type:varchar(32)" json:"source"`
	Available         bool       `gorm:"default:true" json:"available"`
	Status            string     `gorm:"type:varchar(32)" json:"status"`
	Hint              string     `gorm:"type:text" json:"hint"`
	RefreshedAt       *time.Time `gorm:"type:datetime;not null" json:"refreshedAt"`
}
