package repository

import (
	"ic-wails/internal/models"
	"ic-wails/pkg/core/tx"
	pkgrepo "ic-wails/pkg/repository"
)

func NewLocalResourceRepo(ds *tx.DataSource) *LocalResourceRepo {
	instance := &LocalResourceRepo{
		BaseRepository: pkgrepo.NewBaseRepository[models.LocalResourceModel](ds),
	}
	instance.InitializeRepository()
	return instance
}

type LocalResourceRepo struct {
	*pkgrepo.BaseRepository[models.LocalResourceModel]
}
