package api

import (
	"context"
	"ic-wails/internal/models"
	"ic-wails/internal/service"
	pkgapi "ic-wails/pkg/api"
	"ic-wails/pkg/common"
)

func NewTaskCategoryApi(taskCategoryService *service.TaskCategoryService) *TaskCategoryApi {
	return &TaskCategoryApi{
		TaskCategoryService: taskCategoryService,
	}
}

type TaskCategoryApi struct {
	TaskCategoryService *service.TaskCategoryService
}

func (tca *TaskCategoryApi) GetTaskCategoryList(ctx context.Context, name string) (result common.R[[]models.TaskCategoryModel]) {
	defer pkgapi.DeferWailsPanicHandler(&result)
	list := tca.TaskCategoryService.GetTaskCategoryList(ctx)
	return common.S[[]models.TaskCategoryModel](&list)
}

func (tca *TaskCategoryApi) AddTaskCateGory(ctx context.Context, taskCategory models.TaskCategoryModel) (result common.R[string]) {
	defer pkgapi.DeferWailsPanicHandler(&result)
	res := tca.TaskCategoryService.AddTaskCateGory(ctx, taskCategory)
	return common.S[string](&res)
}

func (tca *TaskCategoryApi) SaveTaskCateGory(ctx context.Context, taskCategory []models.TaskCategoryModel) (result common.R[string]) {
	defer pkgapi.DeferWailsPanicHandler(&result)
	res := tca.TaskCategoryService.SaveTaskCateGory(ctx, taskCategory)
	return common.S[string](&res)
}

func (tca *TaskCategoryApi) SaveOrUpdateTaskCateGory(ctx context.Context, taskCategory models.TaskCategoryModel) (result common.R[string]) {
	defer pkgapi.DeferWailsPanicHandler(&result)
	res := tca.TaskCategoryService.SaveOrUpdateTaskCateGory(ctx, taskCategory)
	return common.S[string](&res)
}

func (tca *TaskCategoryApi) DeleteTaskCateGory(ctx context.Context, key string) (result common.R[string]) {
	defer pkgapi.DeferWailsPanicHandler(&result)
	res := tca.TaskCategoryService.DeleteTaskCateGory(ctx, key)
	return common.S[string](&res)
}
