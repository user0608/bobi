package binds_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/user0608/bobi/binds"
)

type testPayload struct {
	Name string `json:"name" query:"name"`
	Age  int    `json:"age" query:"age"`
}

func newJSONContext(body string) *echo.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}

func newQueryContext(query string) *echo.Context {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?"+query, nil)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec)
}

func TestJSON(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    testPayload
		wantErr bool
	}{
		{
			name: "valid json",
			body: `{"name":"john","age":30}`,
			want: testPayload{Name: "john", Age: 30},
		},
		{
			name:    "invalid json",
			body:    `{"name":}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newJSONContext(tt.body)

			var payload testPayload
			err := binds.JSON(c, &payload)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, payload)
		})
	}
}

func TestQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    testPayload
		wantErr bool
	}{
		{
			name:  "valid query",
			query: "name=john&age=30",
			want:  testPayload{Name: "john", Age: 30},
		},
		{
			name:    "invalid query type",
			query:   "name=john&age=abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newQueryContext(tt.query)

			var payload testPayload
			err := binds.Query(c, &payload)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, payload)
		})
	}
}

func TestFrom(t *testing.T) {
	body := `{"name":"john","age":25}`
	c := newJSONContext(body)

	var payload testPayload
	err := binds.From(c, &payload)

	require.NoError(t, err)
	assert.Equal(t, testPayload{Name: "john", Age: 25}, payload)
}
