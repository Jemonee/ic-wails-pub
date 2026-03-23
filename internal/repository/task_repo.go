package repository

import (
	"ic-wails/internal/models"
	"ic-wails/pkg/core/tx"
	pkgrepo "ic-wails/pkg/repository"
)

func NewTaskRepository(ds *tx.DataSource) *TaskRepository {
	instance := &TaskRepository{
		BaseRepository: pkgrepo.NewBaseRepository[models.TaskCategoryModel](ds),
	}
	instance.InitializeRepository()
	return instance
}

type TaskRepository struct {
	*pkgrepo.BaseRepository[models.TaskCategoryModel]
}
