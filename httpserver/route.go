package httpserver

import (
	"github.com/labstack/echo/v5"
	"go.uber.org/fx"
)

const RouteTag = `group:"http-api-routes"`

type Route interface {
	GetMethod() string
	GetPath() string
	HandleRequest(c *echo.Context) error
}

func AsRoute(fn any) any {
	return fx.Annotate(
		fn,
		fx.As(new(Route)),
		fx.ResultTags(RouteTag),
	)
}
