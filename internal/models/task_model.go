package models

import pkgmodels "ic-wails/pkg/models"

type TaskCategoryModel struct {
	*pkgmodels.BaseModel
	Name   string  `gorm:"type:varchar(255);not null" json:"name"`
	Key    string  `gorm:"type:varchar(255);not null" json:"key"`
	Remark *string `gorm:"type:varchar(2000);" json:"remark"`
	Color  *string `gorm:"type:varchar(255);" json:"color"`
}
