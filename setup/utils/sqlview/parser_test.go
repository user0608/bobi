package sqlview

import (
	"slices"
	"strings"
	"testing"
)

func TestParseViewsAcceptsWhitespaceAndKeywordCase(t *testing.T) {
	t.Parallel()

	source := []byte("cReAtE\nOr\tRePlAcE\rTeMp\fReCuRsIvE vIeW\vpublic.example AS SELECT 1")
	got, err := parseViews(source)
	if err != nil {
		t.Fatalf("parseViews() error = %v", err)
	}

	want := []View{{Name: "public.example"}}
	if !slices.Equal(got, want) {
		t.Fatalf("parseViews() = %#v, want %#v", got, want)
	}
}

func TestParseViewsIgnoresOtherCreateStatements(t *testing.T) {
	t.Parallel()

	source := []byte(`
CREATE TABLE example (id INTEGER);
CREATE INDEX example_idx ON example (id);
CREATE OR TABLE invalid_but_unrelated;
CREATE FUNCTION example() RETURNS void AS 'SELECT 1';
`)
	got, err := parseViews(source)
	if err != nil {
		t.Fatalf("parseViews() error = %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("parseViews() = %#v, want empty slice", got)
	}
}

func TestParseViewsPostgreSQLIdentifiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
	}{
		{name: "unquoted dollar", identifier: "report$daily"},
		{name: "quoted punctuation", identifier: `"a.b;--/*view*/"`},
		{name: "quoted escaped quote", identifier: `"a""b"`},
		{name: "qualified quoted", identifier: `"Reporting"."Daily Sales"`},
		{name: "unicode default escape", identifier: `U&"d\0061t\+000061"`},
		{name: "unicode custom escape", identifier: `U&"d!0061ta" UESCAPE '!'`},
		{
			name:       "qualified unicode custom escapes",
			identifier: `U&"sch!0065ma" UESCAPE '!'.U&"v#0069ew" UESCAPE '#'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			source := []byte("CREATE VIEW " + tt.identifier + " AS SELECT 1;")
			got, err := parseViews(source)
			if err != nil {
				t.Fatalf("parseViews() error = %v", err)
			}
			want := []View{{Name: tt.identifier}}
			if !slices.Equal(got, want) {
				t.Fatalf("parseViews() = %#v, want %#v", got, want)
			}
		})
	}
}

func TestParseViewsSQLiteContextualIdentifiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		identifier string
	}{
		{name: "single quoted", identifier: `'legacy name'`},
		{name: "single quoted schema", identifier: `'attached db'.'legacy view'`},
		{name: "mixed delimiters", identifier: "[attached db].`legacy view`"},
		{name: "if not exists is a clause", identifier: "IF NOT EXISTS actual_view"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseViews([]byte("CREATE VIEW " + tt.identifier + " AS SELECT 1;"))
			if err != nil {
				t.Fatalf("parseViews() error = %v", err)
			}

			want := tt.identifier
			if after, ok := strings.CutPrefix(want, "IF NOT EXISTS "); ok {
				want = after
			}
			views := []View{{Name: want}}
			if !slices.Equal(got, views) {
				t.Fatalf("parseViews() = %#v, want %#v", got, views)
			}
		})
	}
}

func TestParseViewsSkipsUTF8BOM(t *testing.T) {
	t.Parallel()

	got, err := parseViews([]byte("\xef\xbb\xbfCREATE VIEW bom_view AS SELECT 1;"))
	if err != nil {
		t.Fatalf("parseViews() error = %v", err)
	}
	want := []View{{Name: "bom_view"}}
	if !slices.Equal(got, want) {
		t.Fatalf("parseViews() = %#v, want %#v", got, want)
	}
}

func TestParseViewsErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		source      []byte
		wantMessage string
	}{
		{
			name:        "unterminated block comment",
			source:      []byte("/* comment"),
			wantMessage: "unterminated block comment at byte 0",
		},
		{
			name:        "unterminated string",
			source:      []byte("SELECT 'value"),
			wantMessage: "unterminated string at byte 7",
		},
		{
			name:        "unterminated escape string",
			source:      []byte(`SELECT E'value\'`),
			wantMessage: "unterminated string at byte 8",
		},
		{
			name:        "unterminated dollar string",
			source:      []byte("SELECT $body$value"),
			wantMessage: "unterminated dollar-quoted string at byte 7",
		},
		{
			name:        "unterminated double-quoted identifier",
			source:      []byte(`CREATE VIEW "value AS SELECT 1`),
			wantMessage: "unterminated quoted identifier at byte 12",
		},
		{
			name:        "unterminated Unicode identifier",
			source:      []byte(`CREATE VIEW U&"value AS SELECT 1`),
			wantMessage: "unterminated quoted identifier at byte 14",
		},
		{
			name:        "unterminated backtick identifier",
			source:      []byte("CREATE VIEW `value AS SELECT 1"),
			wantMessage: "unterminated quoted identifier at byte 12",
		},
		{
			name:        "unterminated bracket identifier",
			source:      []byte("CREATE VIEW [value AS SELECT 1"),
			wantMessage: "unterminated quoted identifier at byte 12",
		},
		{
			name:        "invalid UTF-8",
			source:      []byte{0xff},
			wantMessage: "invalid UTF-8 at byte 0",
		},
		{
			name:        "missing view name",
			source:      []byte("CREATE VIEW"),
			wantMessage: "expected view name at end of input",
		},
		{
			name:        "invalid view name token",
			source:      []byte("CREATE VIEW ! AS SELECT 1"),
			wantMessage: "expected view name at byte 12",
		},
		{
			name:        "incomplete if not exists",
			source:      []byte("CREATE VIEW IF EXISTS example AS SELECT 1"),
			wantMessage: "expected IF NOT EXISTS at byte 12",
		},
		{
			name:        "missing name after schema",
			source:      []byte("CREATE VIEW public."),
			wantMessage: "expected view name after schema at end of input",
		},
		{
			name:        "missing Unicode escape character",
			source:      []byte(`CREATE VIEW U&"value" UESCAPE AS SELECT 1`),
			wantMessage: "expected UESCAPE character at byte 30",
		},
		{
			name:        "empty Unicode escape character",
			source:      []byte(`CREATE VIEW U&"value" UESCAPE '' AS SELECT 1`),
			wantMessage: "UESCAPE must be exactly one character at byte 30",
		},
		{
			name:        "multiple Unicode escape characters",
			source:      []byte(`CREATE VIEW U&"value" UESCAPE '!#' AS SELECT 1`),
			wantMessage: "UESCAPE must be exactly one character at byte 30",
		},
		{
			name:        "hexadecimal Unicode escape character",
			source:      []byte(`CREATE VIEW U&"value" UESCAPE 'a' AS SELECT 1`),
			wantMessage: "invalid UESCAPE character at byte 30",
		},
		{
			name:        "plus Unicode escape character",
			source:      []byte(`CREATE VIEW U&"value" UESCAPE '+' AS SELECT 1`),
			wantMessage: "invalid UESCAPE character at byte 30",
		},
		{
			name:        "quote Unicode escape character",
			source:      []byte(`CREATE VIEW U&"value" UESCAPE '''' AS SELECT 1`),
			wantMessage: "invalid UESCAPE character at byte 30",
		},
		{
			name:        "space Unicode escape character",
			source:      []byte(`CREATE VIEW U&"value" UESCAPE ' ' AS SELECT 1`),
			wantMessage: "invalid UESCAPE character at byte 30",
		},
		{
			name: "invalid Unicode escape in qualified view name",
			source: []byte(
				`CREATE VIEW U&"schema" UESCAPE '!'.U&"value" UESCAPE 'a' AS SELECT 1`,
			),
			wantMessage: "invalid UESCAPE character at byte 53",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseViews(tt.source)
			if err == nil {
				t.Fatalf("parseViews() = %#v, nil; want error", got)
			}
			if !strings.Contains(err.Error(), tt.wantMessage) {
				t.Fatalf("parseViews() error = %q, want it to contain %q", err, tt.wantMessage)
			}
		})
	}
}
