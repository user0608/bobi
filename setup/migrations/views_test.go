package migrations

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	"github.com/user0608/bobi/connection"
	"github.com/user0608/bobi/setup/utils/sqlview"
)

func TestMigrationRunnerUpCreatesAndRefreshesViews(t *testing.T) {
	migrationFS := fstest.MapFS{
		"migrations/001_schema.sql": &fstest.MapFile{Data: []byte(`-- +goose Up
CREATE TABLE marker (id INTEGER PRIMARY KEY);
-- +goose Down
DROP TABLE marker;
`)},
		"migrations/_views/01_base.sql": &fstest.MapFile{
			Data: []byte(`CREATE VIEW "base view" AS SELECT 'first' AS value;`),
		},
		"migrations/_views/nested/02_dependent.SQL": &fstest.MapFile{
			Data: []byte(`CREATE VIEW dependent_view AS SELECT value FROM "base view";`),
		},
		"migrations/_views/notes.txt": &fstest.MapFile{
			Data: []byte(`CREATE VIEW ignored_view AS SELECT 1;`),
		},
	}
	runner, db := newSQLiteMigrationRunner(t, migrationFS)

	if err := runner.Up(context.Background()); err != nil {
		t.Fatalf("first Up() error = %v", err)
	}
	if got := queryString(t, db, "SELECT value FROM dependent_view"); got != "first" {
		t.Fatalf("dependent_view value = %q, want %q", got, "first")
	}

	migrationFS["migrations/_views/01_base.sql"] = &fstest.MapFile{
		Data: []byte(`CREATE VIEW "base view" AS SELECT 'second' AS value;`),
	}
	if err := runner.Up(context.Background()); err != nil {
		t.Fatalf("second Up() error = %v", err)
	}
	if got := queryString(t, db, "SELECT value FROM dependent_view"); got != "second" {
		t.Fatalf("dependent_view value = %q, want %q", got, "second")
	}
	if objectExists(t, db, "ignored_view", "view") {
		t.Fatal("non-SQL file was executed")
	}
}

func TestMigrationRunnerUpRestoresViewsAfterMigrationFailure(t *testing.T) {
	migrationFS := fstest.MapFS{
		"migrations/001_schema.sql": &fstest.MapFile{Data: []byte(`-- +goose Up
CREATE TABLE marker (id INTEGER PRIMARY KEY);
-- +goose Down
DROP TABLE marker;
`)},
		"migrations/_views/stable.sql": &fstest.MapFile{
			Data: []byte(`CREATE VIEW stable_view AS SELECT 42 AS value;`),
		},
	}
	runner, db := newSQLiteMigrationRunner(t, migrationFS)

	if err := runner.Up(context.Background()); err != nil {
		t.Fatalf("first Up() error = %v", err)
	}

	migrationFS["migrations/002_invalid.sql"] = &fstest.MapFile{Data: []byte(`-- +goose Up
THIS IS NOT SQL;
-- +goose Down
SELECT 1;
`)}
	if err := runner.Up(context.Background()); err == nil {
		t.Fatal("second Up() error = nil, want migration error")
	}

	if got := queryInt(t, db, "SELECT value FROM stable_view"); got != 42 {
		t.Fatalf("stable_view value = %d, want 42", got)
	}
}

func TestMigrationRunnerDownDoesNotProcessViews(t *testing.T) {
	migrationFS := fstest.MapFS{
		"migrations/001_schema.sql": &fstest.MapFile{Data: []byte(`-- +goose Up
CREATE TABLE marker (id INTEGER PRIMARY KEY);
-- +goose Down
DROP TABLE marker;
`)},
		"migrations/_views/stable.sql": &fstest.MapFile{
			Data: []byte(`CREATE VIEW stable_view AS SELECT 7 AS value;`),
		},
	}
	runner, db := newSQLiteMigrationRunner(t, migrationFS)

	if err := runner.Up(context.Background()); err != nil {
		t.Fatalf("Up() error = %v", err)
	}
	migrationFS["migrations/_views/stable.sql"] = &fstest.MapFile{
		Data: []byte(`CREATE VIEW stable_view AS INVALID SQL;`),
	}

	if err := runner.Down(context.Background()); err != nil {
		t.Fatalf("Down() error = %v", err)
	}
	if got := queryInt(t, db, "SELECT value FROM stable_view"); got != 7 {
		t.Fatalf("stable_view value = %d, want 7", got)
	}
}

func TestMigrationRunnerUpValidatesViewsBeforeMigrating(t *testing.T) {
	tests := []struct {
		name        string
		migrationFS fstest.MapFS
		wantError   string
	}{
		{
			name: "views path is a file",
			migrationFS: fstest.MapFS{
				"migrations/001_schema.sql": &fstest.MapFile{Data: migrationCreatingMarker()},
				"migrations/_views":         &fstest.MapFile{Data: []byte("not a directory")},
			},
			wantError: "is not a directory",
		},
		{
			name: "malformed view SQL",
			migrationFS: fstest.MapFS{
				"migrations/001_schema.sql": &fstest.MapFile{Data: migrationCreatingMarker()},
				"migrations/_views/broken.sql": &fstest.MapFile{
					Data: []byte(`CREATE VIEW "unterminated`),
				},
			},
			wantError: "read view definitions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner, db := newSQLiteMigrationRunner(t, tt.migrationFS)

			err := runner.Up(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Up() error = %v, want it to contain %q", err, tt.wantError)
			}
			if objectExists(t, db, "marker", "table") {
				t.Fatal("migration ran before views were validated")
			}
		})
	}
}

func TestMigrationRunnerRefreshViewsJoinsMigrationAndRecreationErrors(t *testing.T) {
	migrationFS := fstest.MapFS{
		"migrations/_views/broken.sql": &fstest.MapFile{
			Data: []byte(`CREATE VIEW restored_view AS INVALID SQL;`),
		},
	}
	runner, db := newSQLiteMigrationRunner(t, migrationFS)
	if _, err := db.Exec(`CREATE VIEW restored_view AS SELECT 1 AS value;`); err != nil {
		t.Fatalf("create initial view: %v", err)
	}

	migrationErr := errors.New("migration failed")
	err := runner.refreshViews(context.Background(), db, func() error {
		return migrationErr
	})
	if !errors.Is(err, migrationErr) {
		t.Fatalf("refreshViews() error = %v, want wrapped migration error", err)
	}
	if !strings.Contains(err.Error(), `execute view SQL file "broken.sql"`) {
		t.Fatalf("refreshViews() error = %v, want recreation error", err)
	}
}

func TestMigrationRunnerRefreshViewsRollsBackPartialDrop(t *testing.T) {
	migrationFS := fstest.MapFS{
		"migrations/_views/01_materialized.sql": &fstest.MapFile{
			Data: []byte(`CREATE MATERIALIZED VIEW materialized_view AS SELECT 1;`),
		},
		"migrations/_views/02_regular.sql": &fstest.MapFile{
			Data: []byte(`CREATE VIEW regular_view AS SELECT 9 AS value;`),
		},
	}
	runner, db := newSQLiteMigrationRunner(t, migrationFS)
	if _, err := db.Exec(`CREATE VIEW regular_view AS SELECT 9 AS value;`); err != nil {
		t.Fatalf("create initial view: %v", err)
	}

	migrationCalled := false
	err := runner.refreshViews(context.Background(), db, func() error {
		migrationCalled = true
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "drop view materialized_view") {
		t.Fatalf("refreshViews() error = %v, want materialized drop error", err)
	}
	if migrationCalled {
		t.Fatal("migration ran after drop failure")
	}
	if got := queryInt(t, db, "SELECT value FROM regular_view"); got != 9 {
		t.Fatalf("regular_view value = %d, want 9 after rollback", got)
	}
}

func TestMigrationRunnerRefreshViewsRollsBackPartialRecreation(t *testing.T) {
	migrationFS := fstest.MapFS{
		"migrations/_views/01_valid.sql": &fstest.MapFile{
			Data: []byte(`CREATE VIEW valid_view AS SELECT 1 AS value;`),
		},
		"migrations/_views/02_invalid.sql": &fstest.MapFile{
			Data: []byte(`CREATE VIEW invalid_view AS INVALID SQL;`),
		},
	}
	runner, db := newSQLiteMigrationRunner(t, migrationFS)

	err := runner.refreshViews(context.Background(), db, func() error { return nil })
	if err == nil || !strings.Contains(err.Error(), `execute view SQL file "02_invalid.sql"`) {
		t.Fatalf("refreshViews() error = %v, want recreation error", err)
	}
	if objectExists(t, db, "valid_view", "view") {
		t.Fatal("successful script was not rolled back after later recreation failure")
	}
}

func TestDropViewSQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		view sqlview.View
		want string
	}{
		{
			name: "regular",
			view: sqlview.View{Name: `public."daily report"`},
			want: `DROP VIEW IF EXISTS public."daily report"`,
		},
		{
			name: "materialized",
			view: sqlview.View{Name: "public.daily_report", Materialized: true},
			want: "DROP MATERIALIZED VIEW IF EXISTS public.daily_report",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := dropViewSQL(tt.view); got != tt.want {
				t.Fatalf("dropViewSQL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func newSQLiteMigrationRunner(t *testing.T, migrationFS fstest.MapFS) (*MigrationRunner, *sql.DB) {
	t.Helper()

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	storage, err := connection.NewConnection(connection.DatabaseConfig{
		Driver:   connection.DatabaseDriverSQLite,
		Database: t.TempDir() + "/test.db",
	})
	if err != nil {
		t.Fatalf("create SQLite connection: %v", err)
	}
	db, err := storage.Conn(context.Background()).DB()
	if err != nil {
		t.Fatalf("get SQL database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return NewMigrationRunner(storage, migrationFS), db
}

func migrationCreatingMarker() []byte {
	return []byte(`-- +goose Up
CREATE TABLE marker (id INTEGER PRIMARY KEY);
-- +goose Down
DROP TABLE marker;
`)
}

func queryString(t *testing.T, db *sql.DB, query string) string {
	t.Helper()

	var value string
	if err := db.QueryRow(query).Scan(&value); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return value
}

func queryInt(t *testing.T, db *sql.DB, query string) int {
	t.Helper()

	var value int
	if err := db.QueryRow(query).Scan(&value); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	return value
}

func objectExists(t *testing.T, db *sql.DB, name, objectType string) bool {
	t.Helper()

	var count int
	err := db.QueryRow(
		"SELECT count(*) FROM sqlite_master WHERE name = ? AND type = ?",
		name,
		objectType,
	).Scan(&count)
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	return count > 0
}
