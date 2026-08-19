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

var _ httpserver.Route = (*SPAHandler)(nil)

func NewSPAHandler(content fs.FS, prefix string) *SPAHandler {
	prefix = normalizePrefix(prefix)
	return &SPAHandler{content: content, prefix: prefix}
}

// GetPath implements [httpserver.Route].
func (s *SPAHandler) GetPath() string {
	if s.prefix == "/" {
		return "/*"
	}
	return strings.TrimSuffix(s.prefix, "/") + "*"
}

// HandleRequest implements [httpserver.Route].
func (s *SPAHandler) HandleRequest(c *echo.Context) error {
	if s.content == nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "filesystem is not configured")
	}

	requestedPath, valid := spaPath(c.Request().URL.Path, s.prefix)
	if !valid {
		return echo.ErrNotFound
	}

	info, err := fs.Stat(s.content, requestedPath)
	if err == nil {
		if !info.IsDir() {
			return c.FileFS(requestedPath, s.content)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	indexInfo, err := fs.Stat(s.content, "index.html")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return echo.NewHTTPError(http.StatusInternalServerError, "index.html is not available")
		}
		return err
	}
	if indexInfo.IsDir() {
		return echo.NewHTTPError(http.StatusInternalServerError, "index.html is not available")
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
