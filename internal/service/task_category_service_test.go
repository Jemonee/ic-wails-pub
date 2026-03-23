package service

import (
	"context"
	"testing"

	"ic-wails/internal/models"
	"ic-wails/internal/repository"
	pkgmodels "ic-wails/pkg/models"
)

func TestTaskCategoryServiceAddAndList(t *testing.T) {
	ds := newTestDataSource(t)
	repo := repository.NewTaskRepository(ds)
	svc := NewTaskCategoryService(repo)
	ctx := context.Background()

	result := svc.AddTaskCateGory(ctx, models.TaskCategoryModel{
		BaseModel: &pkgmodels.BaseModel{},
		Name:      "工作",
		Key:       "work",
	})
	if result != "操作成功" {
		t.Fatalf("expected success message, got %q", result)
	}

	list := svc.GetTaskCategoryList(ctx)
	if len(list) != 1 {
		t.Fatalf("expected 1 category, got %d", len(list))
	}
	if list[0].Key != "work" || list[0].Name != "工作" {
		t.Fatalf("unexpected category: %+v", list[0])
	}
}

func TestTaskCategoryServiceRejectsDuplicateKey(t *testing.T) {
	ds := newTestDataSource(t)
	repo := repository.NewTaskRepository(ds)
	svc := NewTaskCategoryService(repo)
	ctx := context.Background()

	_ = svc.AddTaskCateGory(ctx, models.TaskCategoryModel{
		BaseModel: &pkgmodels.BaseModel{},
		Name:      "工作",
		Key:       "work",
	})

	assertServicePanic(t, func() {
		svc.AddTaskCateGory(ctx, models.TaskCategoryModel{
			BaseModel: &pkgmodels.BaseModel{},
			Name:      "重复工作",
			Key:       "work",
		})
	}, 400, "当前key已经存在")
}

func TestTaskCategoryServiceSaveRejectsNilID(t *testing.T) {
	ds := newTestDataSource(t)
	repo := repository.NewTaskRepository(ds)
	svc := NewTaskCategoryService(repo)
	ctx := context.Background()

	assertServicePanic(t, func() {
		svc.SaveTaskCateGory(ctx, []models.TaskCategoryModel{{
			BaseModel: &pkgmodels.BaseModel{},
			Name:      "未持久化分类",
			Key:       "draft",
		}})
	}, 400, "当前key已经存在")
}

func TestTaskCategoryServiceDeleteMarksCategoryInvalid(t *testing.T) {
	ds := newTestDataSource(t)
	repo := repository.NewTaskRepository(ds)
	svc := NewTaskCategoryService(repo)
	ctx := context.Background()

	_ = svc.AddTaskCateGory(ctx, models.TaskCategoryModel{
		BaseModel: &pkgmodels.BaseModel{},
		Name:      "临时分类",
		Key:       "temp",
	})

	result := svc.DeleteTaskCateGory(ctx, "temp")
	if result != "操作成功" {
		t.Fatalf("expected success delete message, got %q", result)
	}

	list := svc.GetTaskCategoryList(ctx)
	if len(list) != 0 {
		t.Fatalf("expected empty valid category list after delete, got %d", len(list))
	}

	stored, err := repo.SelectList(nil, "`Key` = ?", "temp")
	if err != nil {
		t.Fatalf("query deleted category failed: %v", err)
	}
	if len(stored) != 1 {
		t.Fatalf("expected stored deleted category, got %d rows", len(stored))
	}
	if stored[0].Valid != 0 {
		t.Fatalf("expected deleted category valid=0, got %d", stored[0].Valid)
	}
}
