package web

import (
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/user0608/bobi/httpserver"
)

type WebHandler struct {
	httpserver.MethodGet
	content fs.FS
	prefix  string
}

var _ httpserver.Route = (*WebHandler)(nil)

func NewWebHandler(content fs.FS, prefix string) *WebHandler {
	return &WebHandler{content: content, prefix: normalizePrefix(prefix)}
}

// GetPath implements [httpserver.Route].
func (w *WebHandler) GetPath() string {
	if w.prefix == "/" {
		return "/*"
	}
	return strings.TrimSuffix(w.prefix, "/") + "*"
}

// HandleRequest implements [httpserver.Route].
func (w *WebHandler) HandleRequest(c *echo.Context) error {
	if w.content == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "filesystem is not configured")
	}

	requestedPath, valid := webPath(c.Request().URL.Path, w.prefix)
	if !valid {
		return echo.ErrNotFound
	}

	info, err := fs.Stat(w.content, requestedPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return echo.ErrNotFound
		}
		return err
	}
	if info.IsDir() {
		return echo.ErrNotFound
	}

	return c.FileFS(requestedPath, w.content)
}

func normalizePrefix(prefix string) string {
	prefix = "/" + strings.Trim(prefix, "/")
	if prefix == "//" {
		return "/"
	}
	if prefix != "/" {
		prefix += "/"
	}
	return prefix
}

func webPath(requestPath, prefix string) (string, bool) {
	if prefix != "/" {
		if requestPath == strings.TrimSuffix(prefix, "/") {
			requestPath = ""
		} else if strings.HasPrefix(requestPath, prefix) {
			requestPath = strings.TrimPrefix(requestPath, prefix)
		} else {
			return "", false
		}
	} else {
		requestPath = strings.TrimPrefix(requestPath, "/")
	}

	if requestPath == "" {
		return "index.html", true
	}

	cleanPath := path.Clean(requestPath)
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, "../") || strings.Contains(cleanPath, "\\") {
		return "", false
	}
	if !fs.ValidPath(cleanPath) {
		return "", false
	}

	return cleanPath, true
}
