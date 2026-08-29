package httpserver

import "github.com/labstack/echo/v5"

type MiddlewareResolver interface {
	Resolve(route Route) []echo.MiddlewareFunc
}
