---
name: bobi
description: Use the Bobi Go library for HTTP services, validation, errors, persistence, configuration, migrations, JWT keys, and common types. Trigger when a project imports github.com/user0608/bobi or mentions Bobi packages.
---

# Bobi

Use `github.com/user0608/bobi` as a set of focused packages. Inspect `go.mod` first and preserve the version already selected by the project.

## Package routing

- HTTP routes, Echo handlers, middleware, or server lifecycle: read `httpserver/SKILL.md`.
- Request binding, multipart files, or struct validation: read `binds-kcheck/SKILL.md`.
- HTTP response envelopes or application errors: read `errors-answer/SKILL.md`.
- GORM, SQLite, PostgreSQL, or transactions: read `connection/SKILL.md`.
- Application bootstrap, embedded assets, or migrations: read `setup-migrations/SKILL.md`.
- RSA JWT key storage: read `jwtkeys/SKILL.md`.
- Date/time and JSON array types: read `types/SKILL.md`.
- Small generic helpers: read `utilities/SKILL.md`.

## General rules

- Prefer Bobi's public APIs over duplicating equivalent helpers in the application.
- Keep application-specific business logic outside Bobi packages.
- Propagate `context.Context` to database operations and transaction callbacks.
- Keep repositories, services, and handlers separate: repositories access data,
  services apply business rules, and handlers write HTTP responses.
- Translate handler errors at the HTTP boundary with `answer.Err` or `answer.Auto`;
  do not return a typed Bobi error directly as the handler response.
- Do not expose private fields of `connection.StorageManager` or `jwtkeys.JwtKeyStore`; use their public methods.
- Prefer ordered, readable code with one meaningful operation per step.
- Keep lines short and avoid deeply nested conditionals or chained method calls.
- Do not introduce factory functions for simple construction; use them only when a framework such as Fx requires a provider or when construction has real policy.
- Run `go test ./...` after changes and use `go test -race ./...` when changing shared state or concurrency.
