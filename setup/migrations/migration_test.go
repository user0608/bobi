package migrations

import (
	"context"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	"github.com/user0608/bobi/connection"
)

func TestMigrationRunnerSQLScript(t *testing.T) {
	t.Parallel()

	mfs := fstest.MapFS{
		"migrations/002_users.sql": &fstest.MapFile{Data: []byte(`-- +goose Up

-- create users
CREATE TABLE users (
    id INTEGER PRIMARY KEY
);
-- +goose Down
DROP TABLE users;
`)},
		"migrations/001_schema.sql": &fstest.MapFile{Data: []byte(`-- +goose Up
CREATE TABLE schema_version (version INTEGER);
-- +goose Down
DROP TABLE schema_version;
`)},
		"migrations/notes.txt":          &fstest.MapFile{Data: []byte("not SQL")},
		"migrations/subdir/ignored.sql": &fstest.MapFile{Data: []byte("-- +goose Up\nCREATE TABLE ignored_dir (id INTEGER);\n-- +goose Down\n")},
	}
	runner := NewMigrationRunner(nil, mfs)

	got, err := runner.SQLScript()
	if err != nil {
		t.Fatalf("SQLScript() error = %v", err)
	}

	want := "-- 001_schema.sql\nCREATE TABLE schema_version (version INTEGER);\n-- 002_users.sql\nCREATE TABLE users (\nid INTEGER PRIMARY KEY\n);\n"
	if got != want {
		t.Fatalf("SQLScript() = %q, want %q", got, want)
	}
}

func TestMigrationRunnerSQLScriptWithoutDownSection(t *testing.T) {
	runner := NewMigrationScriptRunner(fstest.MapFS{
		"migrations/001_schema.sql": &fstest.MapFile{Data: []byte("-- +goose Up\nCREATE TABLE schema_version (version INTEGER);\n")},
	})

	got, err := runner.SQLScript()
	if err != nil {
		t.Fatalf("SQLScript() error = %v", err)
	}
	want := "-- 001_schema.sql\nCREATE TABLE schema_version (version INTEGER);\n"
	if got != want {
		t.Fatalf("SQLScript() = %q, want %q", got, want)
	}
}

func TestMigrationRunnerSQLScriptRejectsMissingUpSection(t *testing.T) {
	runner := NewMigrationScriptRunner(fstest.MapFS{
		"migrations/001_invalid.sql": &fstest.MapFile{Data: []byte("CREATE TABLE invalid (id INTEGER);\n")},
	})

	_, err := runner.SQLScript()
	if err == nil || !strings.Contains(err.Error(), "001_invalid.sql") {
		t.Fatalf("SQLScript() error = %v, want malformed migration error", err)
	}
}

func TestMigrationRunnerSQLScriptNilFS(t *testing.T) {
	runner := NewMigrationRunner(nil, nil)

	got, err := runner.SQLScript()
	if err != nil {
		t.Fatalf("SQLScript() error = %v", err)
	}
	if got != "" {
		t.Fatalf("SQLScript() = %q, want empty string", got)
	}
}

func TestMigrationRunnerSQLScriptErrors(t *testing.T) {
	tests := []struct {
		name string
		mfs  fs.FS
		want string
	}{
		{
			name: "missing migrations directory",
			mfs:  fstest.MapFS{},
			want: "read migrations dir",
		},
		{
			name: "scanner error",
			mfs: fstest.MapFS{
				"migrations/001_large.sql": &fstest.MapFile{Data: []byte("-- +goose Up\n" + strings.Repeat("x", 70*1024) + "\n-- +goose Down\n")},
			},
			want: "scan migration file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := NewMigrationRunner(nil, tt.mfs)

			_, err := runner.SQLScript()
			if err == nil {
				t.Fatal("SQLScript() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("SQLScript() error = %q, want it to contain %q", err, tt.want)
			}
		})
	}
}

func TestMigrationRunnerNilFSOperations(t *testing.T) {
	runner := NewMigrationRunner(nil, nil)
	ctx := context.Background()

	if err := runner.Up(ctx); err != nil {
		t.Fatalf("Up() error = %v", err)
	}
	if err := runner.Down(ctx); err != nil {
		t.Fatalf("Down() error = %v", err)
	}
	if err := runner.Status(ctx); err != nil {
		t.Fatalf("Status() error = %v", err)
	}
}

func TestMigrationRunnerRequiresStorage(t *testing.T) {
	runner := NewMigrationRunner(nil, fstest.MapFS{
		"migrations/001_schema.sql": &fstest.MapFile{Data: []byte("-- +goose Up\nCREATE TABLE schema_version (version INTEGER);\n-- +goose Down\nDROP TABLE schema_version;\n")},
	})

	if err := runner.Up(context.Background()); err == nil || !strings.Contains(err.Error(), "storage manager is required") {
		t.Fatalf("Up() error = %v, want missing storage manager error", err)
	}
}

func TestMigrationRunnerGooseOperations(t *testing.T) {
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}

	mfs := fstest.MapFS{
		"migrations/001_users.sql": &fstest.MapFile{Data: []byte(`-- +goose Up
CREATE TABLE users (id INTEGER PRIMARY KEY);
-- +goose Down
DROP TABLE users;
`)},
	}
	storage, err := connection.NewConnection(connection.DatabaseConfig{
		Driver:   connection.DatabaseDriverSQLite,
		Database: t.TempDir() + "/test.db",
	})
	if err != nil {
		t.Fatal(err)
	}
	db, err := storage.Conn(context.Background()).DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	runner := NewMigrationRunner(storage, mfs)
	ctx := context.Background()

	if err := runner.Up(ctx); err != nil {
		t.Fatalf("Up() error = %v", err)
	}
	if !storage.Conn(ctx).Migrator().HasTable("users") {
		t.Fatal("Up() did not create users table")
	}
	if err := runner.Status(ctx); err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if err := runner.Down(ctx); err != nil {
		t.Fatalf("Down() error = %v", err)
	}
	if storage.Conn(ctx).Migrator().HasTable("users") {
		t.Fatal("Down() did not drop users table")
	}
}

func TestMigrationRunnerGooseErrors(t *testing.T) {
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatal(err)
	}

	storage, err := connection.NewConnection(connection.DatabaseConfig{
		Driver:   connection.DatabaseDriverSQLite,
		Database: t.TempDir() + "/test.db",
	})
	if err != nil {
		t.Fatal(err)
	}
	db, err := storage.Conn(context.Background()).DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	runner := NewMigrationRunner(storage, fstest.MapFS{})
	err = runner.Up(context.Background())
	if err == nil {
		t.Fatal("Up() error = nil, want error")
	}
}
