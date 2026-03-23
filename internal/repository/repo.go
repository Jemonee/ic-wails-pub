package repository

import (
	"ic-wails/internal/untils"
	"ic-wails/pkg/core/tx"
	"ic-wails/pkg/until"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func NewDataSource() *tx.DataSource {
	dir := untils.RunningDir()
	fullPath := filepath.Join(dir, "data.db")
	db, err := gorm.Open(sqlite.Open(fullPath), &gorm.Config{})
	if err != nil {
		until.Log.Panicf("数据库连接失败: %s", err.Error())
	}
	return tx.CreateDataSource("main", db)
}
