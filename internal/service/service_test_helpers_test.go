package service

import (
	"path/filepath"
	"testing"

	"ic-wails/pkg/common"
	"ic-wails/pkg/core/tx"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestDataSource(t *testing.T) *tx.DataSource {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}

	return tx.CreateDataSource("test", db)
}

func assertServicePanic(t *testing.T, fn func(), expectedCode int, expectedMsg string) {
	t.Helper()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic, got nil")
		}

		servicePanic, ok := r.(common.ServicePanic)
		if !ok {
			t.Fatalf("expected common.ServicePanic, got %T", r)
		}

		if servicePanic.Code != expectedCode {
			t.Fatalf("expected code %d, got %d", expectedCode, servicePanic.Code)
		}

		if servicePanic.Msg != expectedMsg {
			t.Fatalf("expected message %q, got %q", expectedMsg, servicePanic.Msg)
		}
	}()

	fn()
}
