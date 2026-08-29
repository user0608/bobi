package web

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

func TestWebHandlerServesFiles(t *testing.T) {
	server := httpserver.NewServer(httpserver.ServerParams{
		Routes: []httpserver.Route{
			NewWebHandler(fstest.MapFS{
				"index.html":    &fstest.MapFile{Data: []byte("home")},
				"about.html":    &fstest.MapFile{Data: []byte("about")},
				"assets/app.js": &fstest.MapFile{Data: []byte("console.log('app')")},
			}, "/"),
		},
	})

	tests := []struct {
		name         string
		path         string
		status       int
		contentType  string
		responseBody string
	}{
		{name: "root serves index", path: "/", status: http.StatusOK, contentType: "text/html; charset=utf-8", responseBody: "home"},
		{name: "html file serves directly", path: "/about.html", status: http.StatusOK, contentType: "text/html; charset=utf-8", responseBody: "about"},
		{name: "asset serves directly", path: "/assets/app.js", status: http.StatusOK, contentType: "text/javascript; charset=utf-8", responseBody: "console.log('app')"},
		{name: "unknown path is not found", path: "/dashboard", status: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))

			require.Equal(t, tt.status, recorder.Code)
			if tt.contentType != "" {
				require.Equal(t, tt.contentType, recorder.Header().Get("Content-Type"))
			}
			if tt.responseBody != "" {
				require.Equal(t, tt.responseBody, recorder.Body.String())
			}
		})
	}
}

func TestWebHandlerSupportsPrefixes(t *testing.T) {
	tests := []struct {
		name        string
		prefix      string
		requestPath string
		routePath   string
	}{
		{name: "empty means root", prefix: "", requestPath: "/about.html", routePath: "/*"},
		{name: "exact prefix", prefix: "/app/", requestPath: "/app/", routePath: "/app*"},
		{name: "prefixed file", prefix: "/app/", requestPath: "/app/about.html", routePath: "/app*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewWebHandler(fstest.MapFS{
				"index.html": &fstest.MapFile{Data: []byte("home")},
				"about.html": &fstest.MapFile{Data: []byte("about")},
			}, tt.prefix)
			server := httpserver.NewServer(httpserver.ServerParams{Routes: []httpserver.Route{handler}})

			require.Equal(t, tt.routePath, handler.GetPath())
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.requestPath, nil))

			require.Equal(t, http.StatusOK, recorder.Code)
		})
	}
}

func TestWebHandlerRejectsInvalidPaths(t *testing.T) {
	server := httpserver.NewServer(httpserver.ServerParams{
		Routes: []httpserver.Route{
			NewWebHandler(fstest.MapFS{
				"index.html":    &fstest.MapFile{Data: []byte("home")},
				"assets/app.js": &fstest.MapFile{Data: []byte("asset")},
			}, "/app/"),
		},
	})

	for _, requestPath := range []string{"/", "/application/app.js", "/app/../secret", "/app/assets\\app.js", "/app/assets/missing.js"} {
		t.Run(requestPath, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, requestPath, nil))

			require.Equal(t, http.StatusNotFound, recorder.Code)
			require.NotContains(t, recorder.Body.String(), "home")
		})
	}
}

func TestWebHandlerReturnsServerErrorWithoutFilesystem(t *testing.T) {
	server := httpserver.NewServer(httpserver.ServerParams{
		Routes: []httpserver.Route{NewWebHandler(nil, "/")},
	})
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestWebHandlerPropagatesFilesystemErrors(t *testing.T) {
	wantErr := errors.New("filesystem unavailable")
	handler := NewWebHandler(errorFS{err: wantErr}, "/")
	server := echo.New()
	context := server.NewContext(
		httptest.NewRequest(http.MethodGet, "/asset.js", nil),
		httptest.NewRecorder(),
	)

	require.ErrorIs(t, handler.HandleRequest(context), wantErr)
}

type errorFS struct{ err error }

func (f errorFS) Open(string) (fs.File, error) { return nil, f.err }
