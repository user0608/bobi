package migrations

import (
	"context"
	"testing"
	"testing/fstest"

	_ "github.com/user0608/bobi/errs"
)

// This test intentionally imports errs together with the GORM SQLite dialector.
// Before the driver fix, their SQLite drivers both register as "sqlite" and the
// test binary panics during package initialization, before this function runs.
func TestSQLiteMigrationDownDriverRegistrationRepro(t *testing.T) {
	runner, db := newSQLiteMigrationRunner(t, fstest.MapFS{
		"migrations/001_users.sql": &fstest.MapFile{Data: []byte(`-- +goose Up
CREATE TABLE users (id INTEGER PRIMARY KEY);
-- +goose Down
DROP TABLE users;
`)},
	})
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	if err := runner.Up(ctx); err != nil {
		t.Fatalf("Up() error = %v", err)
	}
	if err := runner.Down(ctx); err != nil {
		t.Fatalf("Down() error = %v", err)
	}
}
