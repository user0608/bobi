package sqlview

import (
	"database/sql"
	"errors"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	_ "modernc.org/sqlite"
)

func TestViews(t *testing.T) {
	t.Parallel()

	viewFS := fstest.MapFS{
		"definitions/01_sqlite.sql": &fstest.MapFile{Data: []byte(`
CREATE VIEW users AS SELECT 1;
CREATE VIEW IF NOT EXISTS main.active_users AS SELECT 1;
CREATE TEMP VIEW temp_users AS SELECT 1;
CREATE TEMPORARY VIEW temp."Recent Users" AS SELECT 1;
`)},
		"definitions/nested/02_postgres.SQL": &fstest.MapFile{Data: []byte(`
CREATE OR REPLACE VIEW public.active_users AS SELECT 1;
CREATE OR REPLACE TEMPORARY RECURSIVE VIEW reporting.numbers (n) AS SELECT 1;
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics."Monthly Sales" AS SELECT 1;
CREATE VIEW "reporting"."escaped ""name""" AS SELECT 1;
CREATE VIEW U&"d\0061ta" AS SELECT 1;
CREATE VIEW U&"sch!0065ma" UESCAPE '!'.U&"v!0069ew" UESCAPE '!' AS SELECT 1;
CREATE VIEW café AS SELECT 1;
`)},
		"definitions/nested/03_quoted.sql": &fstest.MapFile{Data: []byte(`
CREATE VIEW [main].[order details] AS SELECT 1;
CREATE VIEW ` + "`audit`.`changes`" + ` AS SELECT 1;
CREATE VIEW 'legacy sqlite name' AS SELECT 1;
CREATE VIEW abort AS SELECT 1;
CREATE VIEW users AS SELECT 2;
`)},
		"definitions/ignored.txt": &fstest.MapFile{Data: []byte("CREATE VIEW ignored AS SELECT 1;")},
	}

	got, err := Views(viewFS)
	if err != nil {
		t.Fatalf("Views() error = %v", err)
	}

	want := []View{
		{Name: "users"},
		{Name: "main.active_users"},
		{Name: "temp_users"},
		{Name: `temp."Recent Users"`},
		{Name: "public.active_users"},
		{Name: "reporting.numbers"},
		{Name: `analytics."Monthly Sales"`, Materialized: true},
		{Name: `"reporting"."escaped ""name"""`},
		{Name: `U&"d\0061ta"`},
		{Name: `U&"sch!0065ma" UESCAPE '!'.U&"v!0069ew" UESCAPE '!'`},
		{Name: "café"},
		{Name: "[main].[order details]"},
		{Name: "`audit`.`changes`"},
		{Name: "'legacy sqlite name'"},
		{Name: "abort"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("Views() = %#v, want %#v", got, want)
	}
}

func TestViewsIgnoresNonExecutableText(t *testing.T) {
	t.Parallel()

	viewFS := fstest.MapFS{
		"views.sql": &fstest.MapFile{Data: []byte(`
-- CREATE VIEW line_comment AS SELECT 1;
/* CREATE VIEW block_comment AS SELECT 1;
   /* CREATE VIEW nested_comment AS SELECT 1; */
*/
SELECT 'CREATE VIEW string_literal AS SELECT 1;';
SELECT 'CREATE VIEW doubled_quote AS SELECT ''value'';';
SELECT E'CREATE VIEW escape_string AS SELECT \'value\'';
SELECT N'CREATE VIEW national_string AS SELECT 1;';
SELECT X'4352454154452056494557';
SELECT U&'CREATE VIEW unicode_string AS SELECT 1;';
SELECT $1;
SELECT $tag;
DO $$ BEGIN
    EXECUTE 'CREATE VIEW dollar_quote AS SELECT 1';
END $$;
DO $body$ BEGIN
    EXECUTE 'CREATE VIEW tagged_dollar_quote AS SELECT 1';
END $body$;
SELECT "CREATE VIEW quoted_identifier AS SELECT 1";
CREATE TABLE ordinary_table (id INTEGER);
CREATE VIEW visible_view AS SELECT 1;
`)},
	}

	got, err := Views(viewFS)
	if err != nil {
		t.Fatalf("Views() error = %v", err)
	}

	want := []View{{Name: "visible_view"}}
	if !slices.Equal(got, want) {
		t.Fatalf("Views() = %#v, want %#v", got, want)
	}
}

func TestViewsEmptyFilesystem(t *testing.T) {
	t.Parallel()

	got, err := Views(fstest.MapFS{})
	if err != nil {
		t.Fatalf("Views() error = %v", err)
	}
	if got == nil {
		t.Fatal("Views() returned a nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("Views() = %#v, want empty slice", got)
	}
}

func TestViewsRejectsConflictingKinds(t *testing.T) {
	t.Parallel()

	viewFS := fstest.MapFS{
		"01_regular.sql": &fstest.MapFile{Data: []byte("CREATE VIEW reports AS SELECT 1;")},
		"02_materialized.sql": &fstest.MapFile{
			Data: []byte("CREATE MATERIALIZED VIEW reports AS SELECT 1;"),
		},
	}

	got, err := Views(viewFS)
	if err == nil {
		t.Fatalf("Views() = %#v, nil; want conflicting kind error", got)
	}
	if !strings.Contains(err.Error(), "both regular and materialized") {
		t.Fatalf("Views() error = %q, want conflicting kind error", err)
	}
}

func TestViewsErrors(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("filesystem failure")
	tests := []struct {
		name        string
		viewFS      fs.FS
		wantMessage string
		wantCause   error
	}{
		{
			name:        "nil filesystem",
			wantMessage: "view filesystem is required",
		},
		{
			name:        "walk failure",
			viewFS:      errorFS{err: sentinel},
			wantMessage: `walk "."`,
			wantCause:   sentinel,
		},
		{
			name: "read failure",
			viewFS: readErrorFS{
				FS: fstest.MapFS{
					"broken.sql": &fstest.MapFile{Data: []byte("CREATE VIEW example AS SELECT 1;")},
				},
				name: "broken.sql",
				err:  sentinel,
			},
			wantMessage: `read SQL file "broken.sql"`,
			wantCause:   sentinel,
		},
		{
			name: "SQL parse failure",
			viewFS: fstest.MapFS{
				"broken.sql": &fstest.MapFile{Data: []byte("CREATE VIEW \"unterminated")},
			},
			wantMessage: `parse SQL file "broken.sql": unterminated quoted identifier`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Views(tt.viewFS)
			if err == nil {
				t.Fatalf("Views() = %#v, nil; want error", got)
			}
			if !strings.Contains(err.Error(), tt.wantMessage) {
				t.Fatalf("Views() error = %q, want it to contain %q", err, tt.wantMessage)
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Fatalf("Views() error = %v, want wrapped error %v", err, tt.wantCause)
			}
		})
	}
}

func TestViewsReturnsEverySQLiteIdentifierForm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
	}{
		{name: "unquoted", identifier: "plain_view"},
		{name: "schema qualified", identifier: "main.schema_view"},
		{name: "double quoted", identifier: `"double quoted"`},
		{name: "escaped double quote", identifier: `"quote""inside"`},
		{name: "backtick quoted", identifier: "`backtick name`"},
		{name: "escaped backtick", identifier: "`back``tick`"},
		{name: "bracket quoted", identifier: "[bracket name]"},
		{name: "single quoted compatibility", identifier: "'single quoted'"},
		{name: "escaped single quote", identifier: "'single '' quote'"},
		{name: "mixed qualification", identifier: "\"main\".`mixed delimiters`"},
		{name: "fallback keyword", identifier: "abort"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			statement := "CREATE VIEW " + tt.identifier + " AS SELECT 1 AS value;"
			viewFS := fstest.MapFS{
				"schema.sql": &fstest.MapFile{Data: []byte(statement)},
			}
			views, err := Views(viewFS)
			if err != nil {
				t.Fatalf("Views() error = %v", err)
			}
			want := []View{{Name: tt.identifier}}
			if !slices.Equal(views, want) {
				t.Fatalf("Views() = %#v, want %#v", views, want)
			}

			db, err := sql.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatalf("sql.Open() error = %v", err)
			}
			t.Cleanup(func() { _ = db.Close() })

			if _, err := db.Exec(statement); err != nil {
				t.Fatalf("create SQLite view with identifier %s: %v", tt.identifier, err)
			}

			var value int
			if err := db.QueryRow("SELECT value FROM " + tt.identifier).Scan(&value); err != nil {
				t.Fatalf("query SQLite view %s: %v", tt.identifier, err)
			}
			if value != 1 {
				t.Fatalf("query SQLite view %s returned %d, want 1", tt.identifier, value)
			}
		})
	}
}

type errorFS struct {
	err error
}

func (f errorFS) Open(string) (fs.File, error) {
	return nil, f.err
}

type readErrorFS struct {
	fs.FS
	name string
	err  error
}

func (f readErrorFS) Open(name string) (fs.File, error) {
	if name == f.name {
		return nil, f.err
	}
	return f.FS.Open(name)
}
