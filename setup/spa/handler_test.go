package spa

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
	"github.com/user0608/bobi/httpserver"
)

func TestSPAHandlerServesAssetsAndReactRoutes(t *testing.T) {
	server := httpserver.NewServer([]httpserver.Route{
		NewSPAHandler(fstest.MapFS{
			"index.html":    &fstest.MapFile{Data: []byte("<div id=app></div>")},
			"assets/app.js": &fstest.MapFile{Data: []byte("console.log('app')")},
		}, "/_/"),
	})

	tests := []struct {
		name         string
		path         string
		status       int
		contentType  string
		responseBody string
	}{
		{
			name:         "root serves index",
			path:         "/_/",
			status:       http.StatusOK,
			contentType:  "text/html; charset=utf-8",
			responseBody: "<div id=app></div>",
		},
		{
			name:         "react route falls back to index",
			path:         "/_/dashboard",
			status:       http.StatusOK,
			contentType:  "text/html; charset=utf-8",
			responseBody: "<div id=app></div>",
		},
		{
			name:         "asset serves directly",
			path:         "/_/assets/app.js",
			status:       http.StatusOK,
			contentType:  "text/javascript; charset=utf-8",
			responseBody: "console.log('app')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			recorder := httptest.NewRecorder()

			server.ServeHTTP(recorder, request)

			require.Equal(t, tt.status, recorder.Code)
			require.Equal(t, tt.contentType, recorder.Header().Get("Content-Type"))
			require.Equal(t, tt.responseBody, recorder.Body.String())
		})
	}
}

func TestSPAHandlerReturnsServerErrorWithoutIndex(t *testing.T) {
	server := httpserver.NewServer([]httpserver.Route{
		NewSPAHandler(fstest.MapFS{
			"assets/app.js": &fstest.MapFile{Data: []byte("console.log('app')")},
		}, "/_/"),
	})

	request := httptest.NewRequest(http.MethodGet, "/_/dashboard", nil)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestSPAHandlerReturnsServerErrorWithoutFilesystem(t *testing.T) {
	server := httpserver.NewServer([]httpserver.Route{
		NewSPAHandler(nil, "/_/"),
	})

	request := httptest.NewRequest(http.MethodGet, "/_/", nil)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestSPAHandlerDoesNotCaptureOutsidePrefix(t *testing.T) {
	server := httpserver.NewServer([]httpserver.Route{
		NewSPAHandler(fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("<div id=app></div>")},
		}, "/_/"),
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestSPAHandlerSupportsDifferentPrefixes(t *testing.T) {
	tests := []struct {
		name        string
		prefix      string
		requestPath string
		routePath   string
	}{
		{name: "root", prefix: "/", requestPath: "/dashboard", routePath: "/*"},
		{name: "pocketbase", prefix: "/_/", requestPath: "/_/dashboard", routePath: "/_/*"},
		{name: "custom", prefix: "/app/", requestPath: "/app/dashboard", routePath: "/app/*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewSPAHandler(fstest.MapFS{
				"index.html": &fstest.MapFile{Data: []byte("<div id=app></div>")},
			}, tt.prefix)
			server := httpserver.NewServer([]httpserver.Route{handler})

			require.Equal(t, tt.routePath, handler.GetPath())

			request := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusOK, recorder.Code)
			require.Equal(t, "<div id=app></div>", recorder.Body.String())
		})
	}
}

func TestSPAHandlerDoesNotCaptureAPIRoutes(t *testing.T) {
	server := httpserver.NewServer([]httpserver.Route{
		NewSPAHandler(fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("<div id=app></div>")},
		}, "/_/"),
		&httpserver.PublicHandler{
			Method: http.MethodGet,
			Path:   "/api/health",
			Handler: func(c *echo.Context) error {
				return c.String(http.StatusOK, "api is healthy")
			},
		},
		&httpserver.PublicHandler{
			Method: http.MethodPost,
			Path:   "/api/users",
			Handler: func(c *echo.Context) error {
				return c.String(http.StatusCreated, "user created")
			},
		},
	})

	tests := []struct {
		name         string
		method       string
		path         string
		status       int
		responseBody string
	}{
		{
			name:         "existing GET route",
			method:       http.MethodGet,
			path:         "/api/health",
			status:       http.StatusOK,
			responseBody: "api is healthy",
		},
		{
			name:         "existing POST route",
			method:       http.MethodPost,
			path:         "/api/users",
			status:       http.StatusCreated,
			responseBody: "user created",
		},
		{
			name:   "unknown API route remains not found",
			method: http.MethodGet,
			path:   "/api/unknown",
			status: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, nil)
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, request)

			require.Equal(t, tt.status, recorder.Code)
			if tt.responseBody != "" {
				require.Equal(t, tt.responseBody, recorder.Body.String())
			}
		})
	}
}
