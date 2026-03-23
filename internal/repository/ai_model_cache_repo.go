package repository

import (
	"ic-wails/internal/models"
	"ic-wails/pkg/core/tx"
	pkgrepo "ic-wails/pkg/repository"
	"strings"

	"gorm.io/gorm"
)

func NewAiModelCacheRepository(ds *tx.DataSource) *AiModelCacheRepository {
	instance := &AiModelCacheRepository{
		BaseRepository: pkgrepo.NewBaseRepository[models.AiModelCacheModel](ds),
	}
	instance.InitializeRepository()
	return instance
}

type AiModelCacheRepository struct {
	*pkgrepo.BaseRepository[models.AiModelCacheModel]
}

func (repo *AiModelCacheRepository) ListByConfigFingerprint(configFingerprint string) ([]models.AiModelCacheModel, error) {
	list := make([]models.AiModelCacheModel, 0)
	err := repo.Db().
		Model(repo.GetEntity()).
		Where("valid = ? AND config_fingerprint = ?", 1, strings.TrimSpace(configFingerprint)).
		Order("sort ASC, create_time ASC, id ASC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (repo *AiModelCacheRepository) ReplaceByConfigFingerprint(configFingerprint string, list []models.AiModelCacheModel) error {
	fingerprint := strings.TrimSpace(configFingerprint)
	return repo.Db().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("config_fingerprint = ?", fingerprint).Delete(repo.GetEntity()).Error; err != nil {
			return err
		}
		if len(list) == 0 {
			return nil
		}
		return tx.Create(&list).Error
	})
}
