package httpserver

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// Public
type PublicHandler struct {
	Method           string
	Path             string
	RequiredPerms    []string
	AnyRequiredPerms []string
	Handler          echo.HandlerFunc
}

var _ Route = (*PublicHandler)(nil)

// GetMethod implements [Route].
func (h *PublicHandler) GetMethod() string {
	if h.Method == "" {
		return http.MethodGet
	}
	return h.Method
}

// GetPath implements [Route].
func (h *PublicHandler) GetPath() string {
	return h.Path
}

// HandleRequest implements [Route].
func (h *PublicHandler) HandleRequest(c *echo.Context) error {
	if h.Handler == nil {
		return echo.NewHTTPError(
			http.StatusInternalServerError,
			"route handler is not configured",
		)
	}
	return h.Handler(c)
}
