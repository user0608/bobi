package httpserver

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func getMethod(method string) string {
	if method == "" {
		return http.MethodGet
	}
	return method
}

func handleRequest(handler echo.HandlerFunc, c *echo.Context) error {
	if handler == nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"route handler is not configured",
		)
	}
	return handler(c)
}

// Public
type PublicHandler struct {
	Method  string
	Path    string
	Handler echo.HandlerFunc
}

var _ Route = (*PublicHandler)(nil)

// GetMethod implements [Route].
func (h *PublicHandler) GetMethod() string {
	return getMethod(h.Method)
}

// GetPath implements [Route].
func (h *PublicHandler) GetPath() string {
	return h.Path
}

// HandleRequest implements [Route].
func (h *PublicHandler) HandleRequest(c *echo.Context) error {
	return handleRequest(h.Handler, c)
}

// Private
type PrivateHandler struct {
	Method  string
	Path    string
	Handler echo.HandlerFunc
}

var _ Route = (*PrivateHandler)(nil)

// GetMethod implements [Route].
func (h *PrivateHandler) GetMethod() string {
	return getMethod(h.Method)
}

// GetPath implements [Route].
func (h *PrivateHandler) GetPath() string {
	return h.Path
}

// HandleRequest implements [Route].
func (h *PrivateHandler) HandleRequest(c *echo.Context) error {
	return handleRequest(h.Handler, c)
}
