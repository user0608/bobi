package main

import (
	"embed"

	"github.com/user0608/bobi/setup"
)

//go:embed all:migrations
var MigrationsDir embed.FS

func main() {
	service := setup.NewService(
	// setup.WithMigration(MigrationsDir),
	)
	service.Run()
}
