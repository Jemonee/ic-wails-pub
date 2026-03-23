package api

import (
	"context"
	"ic-wails/internal/models"
	"ic-wails/internal/service"
	pkgapi "ic-wails/pkg/api"
	"ic-wails/pkg/common"
)

func NewLocalResourceApi(localResourceService *service.LocalResourceService) *LocalResourceApi {
	return &LocalResourceApi{
		LocalResourceService: localResourceService,
	}
}

type LocalResourceApi struct {
	LocalResourceService *service.LocalResourceService
}

func (lra *LocalResourceApi) GetResourceById(ctx context.Context, id uint64) (result common.R[models.LocalResourceModel]) {
	defer pkgapi.DeferWailsPanicHandler(&result)
	res := lra.LocalResourceService.GetResourceById(ctx, id)
	return common.S(&res)
}
