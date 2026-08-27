---
name: setup-migrations
description: Use Bobi setup, Fx modules, embedded static assets, and Goose migrations to bootstrap Go applications. Trigger when configuring setup.NewService, WithMigration, WithSPA_UI, WithWeb_UI, or migrate commands.
---

# Bobi Setup And Migrations

`setup.NewService` is the application bootstrap entry point. It can load Viper configuration, create the database connection, configure JWT keys, register the HTTP server, and expose embedded UI assets.

```go
//go:embed all:migrations
var migrationFS embed.FS

func main() {
	setup.NewService(
		setup.WithMigration(migrationFS),
		setup.WithVersion(version),
	).Run(
		fx.Provide(httpserver.AsRoute(ListUsersRoute)),
	)
}
```

Define `ListUsersRoute` as a named constructor in the application. Create one
constructor per route that returns `*httpserver.PublicHandler` and pass that
constructor to `AsRoute`. Do not create a custom structure for each route, use
an anonymous function, or build route values inline in `Run`.

Use `WithSPA_UI(fs)` for an SPA with fallback to `index.html`, or `WithWeb_UI(fs)` for static files. Do not configure both at the same root unless the route conflict is intentional. `WithSkipConfig` and `WithSkipDBConnection` disable the corresponding default providers.

## Configuration keys

The default service unmarshals HTTP settings from the root (`address`, `log_fmt`) and database settings from `database`. In the current implementation, the JWT key provider also reads the `database` configuration subtree; verify the installed Bobi version before choosing configuration keys for JWT paths.

## Migrations

Embed a directory named `migrations` and use Goose sections:

```sql
-- +goose Up
CREATE TABLE users (id TEXT PRIMARY KEY);

-- +goose Down
DROP TABLE users;
```

With migrations configured, run `go run . migrate up`, `down`, `status`, or `script`. Migration files are read from the embedded filesystem in lexical filename order. `script` prints only the Goose Up SQL.
