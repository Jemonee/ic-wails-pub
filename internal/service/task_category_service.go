package service

import (
	"context"
	"ic-wails/internal/models"
	"ic-wails/internal/repository"
	"ic-wails/pkg/common"
	pkgservice "ic-wails/pkg/service"
	"ic-wails/pkg/until"
	"time"
)

func NewTaskCategoryService(taskRepository *repository.TaskRepository) *TaskCategoryService {
	return &TaskCategoryService{
		BaseService: pkgservice.NewBaseService[models.TaskCategoryModel](taskRepository),
	}
}

type TaskCategoryService struct {
	*pkgservice.BaseService[models.TaskCategoryModel]
}

func (ts *TaskCategoryService) GetTaskCategoryList(ctx context.Context) []models.TaskCategoryModel {
	queryPage := common.Page[models.TaskCategoryModel]{PageNo: 1, PageSize: 10}
	page, queryError := ts.Exec(ctx).SelectPage(nil, queryPage, "valid = ?", 1)
	if queryError != nil {
		until.Log.Printf("数据查询失败: %v", queryError)
		panic(queryError)
	}
	return page.Result
}

func (ts *TaskCategoryService) AddTaskCateGory(ctx context.Context, taskCategory models.TaskCategoryModel) string {
	key := taskCategory.Key
	data, queryError := ts.Exec(ctx).SelectList(nil, "`Key` = ? and `Valid` = ?", key, 1)
	if queryError != nil {
		until.Log.Printf("查询数据出现异常: %v", queryError)
		panic(queryError)
	}
	if len(data) > 0 {
		until.Log.Print("当前key已经存在了", data)
		panic(common.ServicePanic{
			Code: 400,
			Msg:  "当前key已经存在",
		})
	}
	ts.Exec(ctx).Create(&taskCategory)
	return "操作成功"
}

func (ts *TaskCategoryService) SaveTaskCateGory(ctx context.Context, taskCategory []models.TaskCategoryModel) string {
	for _, category := range taskCategory {
		if category.Id == nil {
			panic(common.ServicePanic{
				Code: 400,
				Msg:  "当前key已经存在",
			})
		}
	}
	ts.Exec(ctx).Db().Save(taskCategory)
	return "操作成功"
}

func (ts *TaskCategoryService) SaveOrUpdateTaskCateGory(ctx context.Context, taskCategory models.TaskCategoryModel) string {
	now := time.Now()
	if taskCategory.Id == nil {
		return ts.AddTaskCateGory(ctx, taskCategory)
	}
	taskCategory.ModifyTime = &now
	ts.Exec(ctx).Db().Save(taskCategory)
	return ""
}

func (ts *TaskCategoryService) DeleteTaskCateGory(ctx context.Context, key string) string {
	result := ts.Exec(ctx).Delete("`Key` = ?", key)
	if result != nil {
		panic(result)
	}
	return "操作成功"
}
