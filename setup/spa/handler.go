package spa

import (
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/labstack/echo/v5"
	"github.com/user0608/bobi/httpserver"
)

type SPAHandler struct {
	httpserver.MethodGet
	content fs.FS
	prefix  string
}

func NewSPAHandler(content fs.FS, prefix string) *SPAHandler {
	prefix = normalizePrefix(prefix)
	return &SPAHandler{content: content, prefix: prefix}
}

// GetPath implements [httpserver.Route].
func (s *SPAHandler) GetPath() string {
	if s.prefix == "/" {
		return "/*"
	}
	return s.prefix + "*"
}

// HandleRequest implements [httpserver.Route].
func (s *SPAHandler) HandleRequest(c *echo.Context) error {
	if s.content == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "SPA filesystem is not configured")
	}

	requestedPath, valid := spaPath(c.Request().URL.Path, s.prefix)
	if valid {
		info, err := fs.Stat(s.content, requestedPath)
		if err == nil {
			if !info.IsDir() {
				return c.FileFS(requestedPath, s.content)
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}

	if _, err := fs.Stat(s.content, "index.html"); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return echo.NewHTTPError(http.StatusInternalServerError, "SPA index.html is not available")
		}
		return err
	}

	return c.FileFS("index.html", s.content)
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

func spaPath(requestPath, prefix string) (string, bool) {
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
