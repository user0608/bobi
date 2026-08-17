package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"go.uber.org/fx"
)

func buildMiddlewares(route Route) []echo.MiddlewareFunc {
	var before []echo.MiddlewareFunc
	var after []echo.MiddlewareFunc

	if r, ok := route.(BeforeSecurityMiddlewareProvider); ok {
		before = r.BeforeSecurityMiddlewares()
	}

	if r, ok := route.(AfterSecurityMiddlewareProvider); ok {
		after = r.AfterSecurityMiddlewares()
	}

	middlewares := make([]echo.MiddlewareFunc, 0, len(before)+len(after))
	middlewares = append(middlewares, before...)
	middlewares = append(middlewares, after...)

	return middlewares
}

func NewServer(routes []Route) *echo.Echo {
	server := echo.New()
	server.Use(middleware.RequestLogger())
	server.Use(middleware.Recover())

	for _, route := range routes {
		routeMiddlewares := buildMiddlewares(route)

		switch route.GetMethod() {
		case http.MethodGet:
			server.GET(route.GetPath(), route.HandleRequest, routeMiddlewares...)
		case http.MethodPost:
			server.POST(route.GetPath(), route.HandleRequest, routeMiddlewares...)
		case http.MethodPut:
			server.PUT(route.GetPath(), route.HandleRequest, routeMiddlewares...)
		case http.MethodDelete:
			server.DELETE(route.GetPath(), route.HandleRequest, routeMiddlewares...)
		case http.MethodPatch:
			server.PATCH(route.GetPath(), route.HandleRequest, routeMiddlewares...)
		default:
			slog.Warn("unsupported route method", "method", route.GetMethod(), "path", route.GetPath())
		}
	}

	return server
}

var Module = fx.Module("http-server",
	fx.Provide(
		fx.Annotate(
			NewServer,
			fx.ParamTags(RouteTag),
		),
	),
	fx.Invoke(StartWebServer),
)

type HttpApiConfig struct {
	Address string
}

func StartWebServer(
	lc fx.Lifecycle,
	e *echo.Echo,
	c *HttpApiConfig,
) {
	server := http.Server{
		Addr:    c.Address,
		Handler: e,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			e.Use(middleware.CORS())

			go func() {
				slog.Info("Starting HTTP server", "address", server.Addr)

				if err := server.ListenAndServe(); err != nil &&
					!errors.Is(err, http.ErrServerClosed) {
					slog.Error("HTTP server error", "error", err)
				}
			}()

			return nil
		},

		OnStop: func(ctx context.Context) error {

			shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			if err := server.Shutdown(shutdownCtx); err != nil {
				slog.Error("Error shutting down HTTP server", "error", err)
				return err
			}

			slog.Info("HTTP server stopped successfully")
			return nil
		},
	})
}
