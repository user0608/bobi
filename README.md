# Bobi

Reusable utility library for Go applications. It includes helpers for HTTP,
persistence, configuration, errors, common types, and JWT key management.

## Paquetes

- `answer`: application response structures.
- `binds`: form and request data binding and validation.
- `configs`: configuration path loading and resolution.
- `connection`: SQLite, PostgreSQL, and GORM connections and abstractions.
- `errs`: database and application error helpers.
- `httpserver`: HTTP server creation and configuration with Echo.
- `jwtkeys`: JWT RSA key pair generation and loading.
- `setup`: application setup and service configuration.
- `setup/migrations`: database migration execution and management.
- `types`: helper types for dates, times, UUIDs, and arrays.

## Desarrollo

Run all tests:

```bash
go test ./...
```

Run tests with the race detector:

```bash
go test -race ./...
```
