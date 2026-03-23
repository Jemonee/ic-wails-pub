package models

import pkgmodels "ic-wails/pkg/models"

type LocalResourceModel struct {
	*pkgmodels.BaseModel
	Name    string  `gorm:"type:varchar(255)" json:"name"`
	Path    *string `gorm:"type:varchar(2000)" json:"path"`
	Type    string  `gorm:"type:varchar(255)" json:"type"`
	Content []byte  `gorm:"type:blob" json:"content"`
}
