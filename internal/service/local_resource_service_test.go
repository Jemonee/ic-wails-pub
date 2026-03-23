package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"ic-wails/internal/models"
	"ic-wails/internal/repository"
	pkgmodels "ic-wails/pkg/models"
)

func TestLocalResourceServiceReturnsStoredContent(t *testing.T) {
	ds := newTestDataSource(t)
	repo := repository.NewLocalResourceRepo(ds)
	svc := NewLocalResourceService(repo)
	ctx := context.Background()

	resource := models.LocalResourceModel{
		BaseModel: &pkgmodels.BaseModel{},
		Name:      "内存资源",
		Type:      "text/plain",
		Content:   []byte("stored-content"),
	}
	if err := repo.Create(&resource); err != nil {
		t.Fatalf("create resource failed: %v", err)
	}

	got := svc.GetResourceById(ctx, *resource.Id)
	if string(got.Content) != "stored-content" {
		t.Fatalf("expected stored content, got %q", string(got.Content))
	}
}

func TestLocalResourceServiceLoadsFileContentWhenContentEmpty(t *testing.T) {
	ds := newTestDataSource(t)
	repo := repository.NewLocalResourceRepo(ds)
	svc := NewLocalResourceService(repo)
	ctx := context.Background()

	filePath := filepath.Join(t.TempDir(), "resource.txt")
	if err := os.WriteFile(filePath, []byte("file-content"), 0o600); err != nil {
		t.Fatalf("write temp file failed: %v", err)
	}

	resource := models.LocalResourceModel{
		BaseModel: &pkgmodels.BaseModel{},
		Name:      "文件资源",
		Path:      &filePath,
		Type:      "text/plain",
	}
	if err := repo.Create(&resource); err != nil {
		t.Fatalf("create resource failed: %v", err)
	}

	got := svc.GetResourceById(ctx, *resource.Id)
	if string(got.Content) != "file-content" {
		t.Fatalf("expected file content to be loaded, got %q", string(got.Content))
	}
}

func TestLocalResourceServicePanicsWhenFileMissing(t *testing.T) {
	ds := newTestDataSource(t)
	repo := repository.NewLocalResourceRepo(ds)
	svc := NewLocalResourceService(repo)
	ctx := context.Background()

	missingPath := filepath.Join(t.TempDir(), "missing.txt")
	resource := models.LocalResourceModel{
		BaseModel: &pkgmodels.BaseModel{},
		Name:      "缺失文件资源",
		Path:      &missingPath,
		Type:      "text/plain",
	}
	if err := repo.Create(&resource); err != nil {
		t.Fatalf("create resource failed: %v", err)
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when file is missing")
		}

		err, ok := r.(error)
		if !ok {
			t.Fatalf("expected error panic, got %T", r)
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected os.ErrNotExist, got %v", err)
		}
	}()

	svc.GetResourceById(ctx, *resource.Id)
}
