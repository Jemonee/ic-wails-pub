//go:build wireinject
// +build wireinject

package cmd

import (
	"ic-wails/internal/api"
	"ic-wails/internal/config"
	"ic-wails/internal/core"
	"ic-wails/internal/repository"
	"ic-wails/internal/service"

	"github.com/google/wire"
)

func InitializeApp() *core.ApplicationHolder {
	wire.Build(
		config.NewApplicationConfigManager,

		repository.NewDataSource,
		repository.NewTaskRepository,
		repository.NewLocalResourceRepo,
		repository.NewAiChatRecordRepository,
		repository.NewAiModelCacheRepository,

		service.NewTaskCategoryService,
		service.NewLocalResourceService,

		service.NewAiChatService,

		api.NewWindowManagerApi,
		api.NewTaskCategoryApi,
		api.NewLocalResourceApi,
		api.NewAiChatApi,
		api.NewFrontendNetApi,
		api.NewFrontendApi,

		core.NewApplicationHolder,
	)
	return nil
}
