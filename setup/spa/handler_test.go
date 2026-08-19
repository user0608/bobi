package spa

import (
	"errors"
	"io/fs"
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
			"assets/docs":   &fstest.MapFile{Mode: fs.ModeDir},
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
		{
			name:         "directory falls back to index",
			path:         "/_/assets/docs",
			status:       http.StatusOK,
			contentType:  "text/html; charset=utf-8",
			responseBody: "<div id=app></div>",
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
	tests := []struct {
		name    string
		content fs.FS
	}{
		{
			name: "missing index",
			content: fstest.MapFS{
				"assets/app.js": &fstest.MapFile{Data: []byte("console.log('app')")},
			},
		},
		{
			name: "index is a directory",
			content: fstest.MapFS{
				"index.html": &fstest.MapFile{Mode: fs.ModeDir},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httpserver.NewServer([]httpserver.Route{
				NewSPAHandler(tt.content, "/_/"),
			})
			request := httptest.NewRequest(http.MethodGet, "/_/dashboard", nil)
			recorder := httptest.NewRecorder()

			server.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusInternalServerError, recorder.Code)
			require.Contains(t, recorder.Body.String(), "index.html is not available")
		})
	}
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
		{name: "empty means root", prefix: "", requestPath: "/dashboard", routePath: "/*"},
		{name: "root", prefix: "/", requestPath: "/dashboard", routePath: "/*"},
		{name: "root with repeated slashes", prefix: "///", requestPath: "/dashboard", routePath: "/*"},
		{name: "pocketbase", prefix: "/_/", requestPath: "/_/dashboard", routePath: "/_*"},
		{name: "without slashes", prefix: "app", requestPath: "/app/dashboard", routePath: "/app*"},
		{name: "without trailing slash", prefix: "/app", requestPath: "/app/dashboard", routePath: "/app*"},
		{name: "custom", prefix: "/app/", requestPath: "/app/dashboard", routePath: "/app*"},
		{name: "repeated trailing slashes", prefix: "/app///", requestPath: "/app/dashboard", routePath: "/app*"},
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

func TestSPAHandlerPrefixBoundary(t *testing.T) {
	server := httpserver.NewServer([]httpserver.Route{
		NewSPAHandler(fstest.MapFS{
			"index.html": &fstest.MapFile{Data: []byte("<div id=app></div>")},
		}, "/app/"),
	})

	tests := []struct {
		name   string
		path   string
		status int
	}{
		{name: "exact prefix", path: "/app", status: http.StatusOK},
		{name: "prefix root", path: "/app/", status: http.StatusOK},
		{name: "prefix route", path: "/app/dashboard", status: http.StatusOK},
		{name: "similar prefix", path: "/application/dashboard", status: http.StatusNotFound},
		{name: "encoded traversal", path: "/app/%2e%2e/secret", status: http.StatusNotFound},
		{name: "parent path", path: "/", status: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			recorder := httptest.NewRecorder()

			server.ServeHTTP(recorder, request)

			require.Equal(t, tt.status, recorder.Code)
			if tt.status == http.StatusNotFound {
				require.NotContains(t, recorder.Body.String(), "<div id=app></div>")
			}
		})
	}
}

func TestSPAHandlerIgnoresQueryString(t *testing.T) {
	server := httpserver.NewServer([]httpserver.Route{
		NewSPAHandler(fstest.MapFS{
			"index.html":    &fstest.MapFile{Data: []byte("index")},
			"assets/app.js": &fstest.MapFile{Data: []byte("asset")},
		}, "/app/"),
	})
	request := httptest.NewRequest(http.MethodGet, "/app/assets/app.js?v=123", nil)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "asset", recorder.Body.String())
}

func TestSPAHandlerPropagatesFilesystemErrors(t *testing.T) {
	wantErr := errors.New("filesystem unavailable")
	tests := []struct {
		name    string
		content fs.FS
	}{
		{name: "requested file stat fails", content: errorFS{err: wantErr}},
		{name: "fallback index stat fails", content: fallbackErrorFS{err: wantErr}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewSPAHandler(tt.content, "/app/")
			server := echo.New()
			request := httptest.NewRequest(http.MethodGet, "/app/asset.js", nil)
			recorder := httptest.NewRecorder()
			context := server.NewContext(request, recorder)

			err := handler.HandleRequest(context)

			require.ErrorIs(t, err, wantErr)
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

func TestNormalizePrefix(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		want   string
	}{
		{name: "empty", prefix: "", want: "/"},
		{name: "root", prefix: "/", want: "/"},
		{name: "repeated root slashes", prefix: "///", want: "/"},
		{name: "without slashes", prefix: "app", want: "/app/"},
		{name: "without leading slash", prefix: "app/", want: "/app/"},
		{name: "without trailing slash", prefix: "/app", want: "/app/"},
		{name: "with slashes", prefix: "/app/", want: "/app/"},
		{name: "repeated surrounding slashes", prefix: "//app///", want: "/app/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, normalizePrefix(tt.prefix))
		})
	}
}

func TestSPAPath(t *testing.T) {
	tests := []struct {
		name        string
		requestPath string
		prefix      string
		wantPath    string
		wantValid   bool
	}{
		{name: "root index", requestPath: "/", prefix: "/", wantPath: "index.html", wantValid: true},
		{name: "root asset", requestPath: "/assets/app.js", prefix: "/", wantPath: "assets/app.js", wantValid: true},
		{name: "exact prefix", requestPath: "/app", prefix: "/app/", wantPath: "index.html", wantValid: true},
		{name: "prefix index", requestPath: "/app/", prefix: "/app/", wantPath: "index.html", wantValid: true},
		{name: "prefix asset", requestPath: "/app/assets/app.js", prefix: "/app/", wantPath: "assets/app.js", wantValid: true},
		{name: "clean dot segment", requestPath: "/app/assets/./app.js", prefix: "/app/", wantPath: "assets/app.js", wantValid: true},
		{name: "clean repeated separator", requestPath: "/app/assets//app.js", prefix: "/app/", wantPath: "assets/app.js", wantValid: true},
		{name: "outside prefix", requestPath: "/api/users", prefix: "/app/", wantValid: false},
		{name: "similar prefix", requestPath: "/application/app.js", prefix: "/app/", wantValid: false},
		{name: "parent traversal", requestPath: "/app/../secret", prefix: "/app/", wantValid: false},
		{name: "nested parent traversal", requestPath: "/app/assets/../../secret", prefix: "/app/", wantValid: false},
		{name: "backslash", requestPath: `/app/assets\app.js`, prefix: "/app/", wantValid: false},
		{name: "absolute path after prefix", requestPath: "/app//secret", prefix: "/app/", wantValid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, valid := spaPath(tt.requestPath, tt.prefix)

			require.Equal(t, tt.wantValid, valid)
			require.Equal(t, tt.wantPath, path)
		})
	}
}

type errorFS struct {
	err error
}

func (f errorFS) Open(string) (fs.File, error) {
	return nil, f.err
}

type fallbackErrorFS struct {
	err error
}

func (f fallbackErrorFS) Open(name string) (fs.File, error) {
	if name != "index.html" {
		return nil, fs.ErrNotExist
	}
	return nil, f.err
}
