package main

import (
	"embed"
	"io/fs"

	"github.com/user0608/bobi/setup"
)

//go:embed all:migrations
var MigrationsDir embed.FS

//go:embed all:dist
var UIDir embed.FS

func main() {
	ui, _ := fs.Sub(UIDir, "dist")

	service := setup.NewService(
		// setup.WithMigration(MigrationsDir),
		setup.WithSPA_UI(ui),
	)
	service.Run()
}
