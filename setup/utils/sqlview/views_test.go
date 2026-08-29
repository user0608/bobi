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

func TestViewNames(t *testing.T) {
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

	got, err := ViewNames(viewFS)
	if err != nil {
		t.Fatalf("ViewNames() error = %v", err)
	}

	want := []string{
		"users",
		"main.active_users",
		"temp_users",
		`temp."Recent Users"`,
		"public.active_users",
		"reporting.numbers",
		`analytics."Monthly Sales"`,
		`"reporting"."escaped ""name"""`,
		`U&"d\0061ta"`,
		`U&"sch!0065ma" UESCAPE '!'.U&"v!0069ew" UESCAPE '!'`,
		"café",
		"[main].[order details]",
		"`audit`.`changes`",
		"'legacy sqlite name'",
		"abort",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("ViewNames() = %#v, want %#v", got, want)
	}
}

func TestViewNamesIgnoresNonExecutableText(t *testing.T) {
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

	got, err := ViewNames(viewFS)
	if err != nil {
		t.Fatalf("ViewNames() error = %v", err)
	}

	want := []string{"visible_view"}
	if !slices.Equal(got, want) {
		t.Fatalf("ViewNames() = %#v, want %#v", got, want)
	}
}

func TestViewNamesEmptyFilesystem(t *testing.T) {
	t.Parallel()

	got, err := ViewNames(fstest.MapFS{})
	if err != nil {
		t.Fatalf("ViewNames() error = %v", err)
	}
	if got == nil {
		t.Fatal("ViewNames() returned a nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("ViewNames() = %#v, want empty slice", got)
	}
}

func TestViewNamesErrors(t *testing.T) {
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

			got, err := ViewNames(tt.viewFS)
			if err == nil {
				t.Fatalf("ViewNames() = %#v, nil; want error", got)
			}
			if !strings.Contains(err.Error(), tt.wantMessage) {
				t.Fatalf("ViewNames() error = %q, want it to contain %q", err, tt.wantMessage)
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Fatalf("ViewNames() error = %v, want wrapped error %v", err, tt.wantCause)
			}
		})
	}
}

func TestViewNamesReturnsEverySQLiteIdentifierForm(t *testing.T) {
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
			names, err := ViewNames(viewFS)
			if err != nil {
				t.Fatalf("ViewNames() error = %v", err)
			}
			if !slices.Equal(names, []string{tt.identifier}) {
				t.Fatalf("ViewNames() = %#v, want %#v", names, []string{tt.identifier})
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
