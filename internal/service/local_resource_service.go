package service

import (
	"context"
	"ic-wails/internal/models"
	"ic-wails/internal/repository"
	pkgservice "ic-wails/pkg/service"
	"ic-wails/pkg/until"
	"os"
)

func NewLocalResourceService(repo *repository.LocalResourceRepo) *LocalResourceService {
	return &LocalResourceService{
		BaseService: pkgservice.NewBaseService[models.LocalResourceModel](repo),
	}
}

type LocalResourceService struct {
	*pkgservice.BaseService[models.LocalResourceModel]
}

// GetResourceById 按 ID 查询资源，若 Content 为空则从 Path 读取文件内容填充。
func (s *LocalResourceService) GetResourceById(ctx context.Context, id uint64) models.LocalResourceModel {
	resource, err := s.Exec(ctx).One(nil, "`id` = ? and `valid` = 1", id)
	if err != nil {
		until.Log.Printf("查询资源失败: %v", err)
		panic(err)
	}
	if resource.Content == nil && resource.Path != nil {
		file, fileErr := os.ReadFile(*resource.Path)
		if fileErr != nil {
			until.Log.Printf("读取文件失败: %v", fileErr)
			panic(fileErr)
		}
		resource.Content = file
	}
	return resource
}
