package httpserver_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/user0608/bobi/httpserver"
)

func TestPublicHandlerDefaultsToGet(t *testing.T) {
	handler := &httpserver.PublicHandler{Path: "/health"}

	require.Equal(t, http.MethodGet, handler.GetMethod())
	require.Equal(t, "/health", handler.GetPath())
}

func TestPublicHandlerDelegatesToHandler(t *testing.T) {
	called := false
	handler := &httpserver.PublicHandler{
		Path: "/health",
		Handler: func(c *echo.Context) error {
			called = true
			return c.String(http.StatusNoContent, "")
		},
	}
	server := httpserver.NewServer([]httpserver.Route{handler})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)

	require.True(t, called)
	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestPublicHandlerWithoutHandlerReturnsServerError(t *testing.T) {
	server := httpserver.NewServer([]httpserver.Route{
		&httpserver.PublicHandler{Path: "/health"},
	})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusInternalServerError, recorder.Code)
}

func TestNewServerRegistersHTTPMethods(t *testing.T) {
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
		http.MethodHead,
		http.MethodOptions,
		http.MethodConnect,
	}

	routes := make([]httpserver.Route, 0, len(methods))
	for _, method := range methods {
		method := method
		routes = append(routes, &httpserver.PublicHandler{
			Method: method,
			Path:   "/" + method,
			Handler: func(c *echo.Context) error {
				return c.NoContent(http.StatusNoContent)
			},
		})
	}
	server := httpserver.NewServer(routes)

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			request := httptest.NewRequest(method, "/"+method, nil)
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, request)

			require.Equal(t, http.StatusNoContent, recorder.Code)
		})
	}
}

func TestNewServerRunsRouteMiddlewaresInOrder(t *testing.T) {
	var calls []string
	appendCall := func(name string) echo.MiddlewareFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
				calls = append(calls, name)
				return next(c)
			}
		}
	}

	server := httpserver.NewServer([]httpserver.Route{
		&httpserver.PublicHandler{
			Path:              "/ordered",
			BeforeMiddlewares: []echo.MiddlewareFunc{appendCall("before")},
			Middlewares:       []echo.MiddlewareFunc{appendCall("after")},
			Handler: func(c *echo.Context) error {
				calls = append(calls, "handler")
				return c.NoContent(http.StatusNoContent)
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/ordered", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Equal(t, []string{"before", "after", "handler"}, calls)
}

func TestNewServerConfiguresCORS(t *testing.T) {
	server := httpserver.NewServer([]httpserver.Route{
		&httpserver.PublicHandler{
			Path: "/health",
			Handler: func(c *echo.Context) error {
				return c.NoContent(http.StatusNoContent)
			},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.Header.Set(echo.HeaderOrigin, "https://example.com")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)

	require.NotEmpty(t, recorder.Header().Get(echo.HeaderAccessControlAllowOrigin))
}

func TestStartWebServerReturnsListenError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	app := fx.New(
		fx.Supply(echo.New()),
		fx.Supply(&httpserver.HttpApiConfig{Address: listener.Addr().String()}),
		fx.Invoke(httpserver.StartWebServer),
	)

	err = app.Start(context.Background())
	require.Error(t, err)
	require.False(t, errors.Is(err, context.Canceled))

	stopErr := app.Stop(context.Background())
	require.NoError(t, stopErr)
}

func TestMethodsReturnExpectedHTTPMethods(t *testing.T) {
	tests := []struct {
		name   string
		method func() string
		want   string
	}{
		{name: "GET", method: func() string { return (&httpserver.MethodGet{}).GetMethod() }, want: http.MethodGet},
		{name: "POST", method: func() string { return (&httpserver.MethodPost{}).GetMethod() }, want: http.MethodPost},
		{name: "PUT", method: func() string { return (&httpserver.MethodPut{}).GetMethod() }, want: http.MethodPut},
		{name: "DELETE", method: func() string { return (&httpserver.MethodDelete{}).GetMethod() }, want: http.MethodDelete},
		{name: "PATCH", method: func() string { return (&httpserver.MethodPatch{}).GetMethod() }, want: http.MethodPatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.method())
		})
	}
}
