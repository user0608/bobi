package binds_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user0608/bobi/binds"
)

func newContext(body string) *echo.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}

func mustUUID(s string) uuid.UUID {
	return uuid.MustParse(s)
}

func TestRequestUUIDs(t *testing.T) {
	u1 := "220a4e82-0dc0-46bf-98f9-e0e3c06b2bf7"
	u2 := "ced02bc8-d1c7-4bcc-8eaa-a6ad58df251a"

	tests := []struct {
		name    string
		body    string
		want    []uuid.UUID
		wantErr bool
	}{
		{
			name: "code single",
			body: `{"code":"` + u1 + `"}`,
			want: []uuid.UUID{mustUUID(u1)},
		},
		{
			name: "codes array",
			body: `{"codes":["` + u1 + `","` + u2 + `"]}`,
			want: []uuid.UUID{mustUUID(u1), mustUUID(u2)},
		},
		{
			name: "id field",
			body: `{"id":"` + u1 + `"}`,
			want: []uuid.UUID{mustUUID(u1)},
		},
		{
			name: "ids array",
			body: `{"ids":["` + u1 + `","` + u2 + `"]}`,
			want: []uuid.UUID{mustUUID(u1), mustUUID(u2)},
		},
		{
			name: "uuid fallback",
			body: `{"uuid":"` + u1 + `"}`,
			want: []uuid.UUID{mustUUID(u1)},
		},
		{
			name: "uuids fallback",
			body: `{"uuids":["` + u1 + `"]}`,
			want: []uuid.UUID{mustUUID(u1)},
		},
		{
			name: "values lowest priority",
			body: `{"values":["` + u1 + `","` + u2 + `"]}`,
			want: []uuid.UUID{mustUUID(u1), mustUUID(u2)},
		},

		// PRIORITY TESTS (IMPORTANT)
		{
			name: "code beats all",
			body: `{"uuid":"` + u1 + `","code":"` + u2 + `"}`,
			want: []uuid.UUID{mustUUID(u2)},
		},
		{
			name: "codes beats id",
			body: `{"codes":["` + u1 + `"],"id":"` + u2 + `"}`,
			want: []uuid.UUID{mustUUID(u1)},
		},
		{
			name: "id beats uuid",
			body: `{"uuid":"` + u1 + `","id":"` + u2 + `"}`,
			want: []uuid.UUID{mustUUID(u2)},
		},
		{
			name: "ids beats uuid",
			body: `{"ids":["` + u1 + `"],"uuid":"` + u2 + `"}`,
			want: []uuid.UUID{mustUUID(u1)},
		},
		{
			name: "uuid beats values",
			body: `{"values":["` + u1 + `"],"uuid":"` + u2 + `"}`,
			want: []uuid.UUID{mustUUID(u2)},
		},

		{
			name: "unique values",
			body: `{"codes":["` + u1 + `","` + u1 + `","` + u2 + `"]}`,
			want: []uuid.UUID{mustUUID(u1), mustUUID(u2)},
		},
		{
			name:    "invalid uuid",
			body:    `{"code":"invalid"}`,
			wantErr: true,
		},
		{
			name:    "missing field",
			body:    `{}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newContext(tt.body)

			res, err := binds.RequestUUIDs(c)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}

func TestRequestStrings(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    []string
		wantErr bool
	}{
		{
			name: "code priority",
			body: `{"code":["a"],"values":["b"]}`,
			want: []string{"a"},
		},
		{
			name: "values used",
			body: `{"values":["a","b",""]}`,
			want: []string{"a", "b"},
		},
		{
			name: "single string",
			body: `{"value":"a"}`,
			want: []string{"a"},
		},
		{
			name: "multiple fields uses first match",
			body: `{"strings":["x"],"values":["y"]}`,
			want: []string{"y"},
		},
		{
			name: "unique values",
			body: `{"values":["a","a","b"]}`,
			want: []string{"a", "b"},
		},
		{
			name:    "invalid type",
			body:    `{"values":[1,2]}`,
			wantErr: true,
		},
		{
			name:    "missing field",
			body:    `{}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newContext(tt.body)

			res, err := binds.RequestStrings(c)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}
