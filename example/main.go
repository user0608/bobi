package main

import (
	"embed"
	"io/fs"
	"log/slog"

	"github.com/labstack/echo/v5"
	"github.com/user0608/bobi/httpserver"
	"github.com/user0608/bobi/setup"
	"go.uber.org/fx"
)

//go:embed all:migrations
var MigrationsDir embed.FS

//go:embed all:dist
var UIDir embed.FS

func main() {
	ui, _ := fs.Sub(UIDir, "dist")

	service := setup.NewService(
		setup.WithMigration(MigrationsDir),
		setup.WithSPA_UI(ui),
	)
	service.Run(fx.Provide(NewMidlResolver))
}

type MidlResolver struct {
}

func NewMidlResolver() httpserver.MiddlewareResolver {
	return &MidlResolver{}
}

// Resolve implements [httpserver.MiddlewareResolver].
func (m *MidlResolver) Resolve(route httpserver.Route) []echo.MiddlewareFunc {
	slog.Info("MiddlewareResolver executed")
	return []echo.MiddlewareFunc{}
}
