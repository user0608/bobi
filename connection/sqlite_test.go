package connection

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSQLiteProductionSettings(t *testing.T) {
	storage, err := NewConnection(DBConfigParams{
		Backend:  BackendSQLite,
		Database: filepath.Join(t.TempDir(), "production.db"),
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	var journalMode string
	if err := storage.Conn(ctx).Raw("PRAGMA journal_mode").Scan(&journalMode).Error; err != nil {
		t.Fatal(err)
	}
	if journalMode != "wal" {
		t.Fatalf("expected WAL journal mode, got %q", journalMode)
	}

	var foreignKeys int
	if err := storage.Conn(ctx).Raw("PRAGMA foreign_keys").Scan(&foreignKeys).Error; err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatalf("expected foreign keys enabled, got %d", foreignKeys)
	}

	sqlDB, err := storage.Conn(ctx).DB()
	if err != nil {
		t.Fatal(err)
	}
	if sqlDB.Stats().MaxOpenConnections != 1 {
		t.Fatalf("expected one open connection, got %d", sqlDB.Stats().MaxOpenConnections)
	}
}
