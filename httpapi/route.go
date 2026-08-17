package httpapi

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

// region Public

type publicRouteMarker interface{ isPublicRoute() }

var _ publicRouteMarker = (*PublicRoute)(nil)

type PublicRoute struct{}

func (*PublicRoute) isPublicRoute() {}

// region System

type systemRouteMarker interface{ isSystemRoute() }

var _ systemRouteMarker = (*SystemRoute)(nil)

type SystemRoute struct{}

func (*SystemRoute) isSystemRoute() {}
