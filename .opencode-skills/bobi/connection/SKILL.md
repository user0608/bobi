---
name: connection
description: Use Bobi connection for GORM SQLite or PostgreSQL setup and context-aware transactions. Trigger when configuring DatabaseConfig, NewConnection, StorageManager, Conn, or WithTx.
---

# Bobi Database Connection

Configure `connection.DatabaseConfig` through Viper or directly:

```go
manager, err := connection.NewConnection(connection.DatabaseConfig{
	Driver:   connection.DatabaseDriverSQLite,
	Database: "./data/",
	LogLevel: "warn",
})
```

Supported drivers are `connection.DatabaseDriverSQLite` and `connection.DatabaseDriverPostgres`. SQLite enables WAL, foreign keys, and a busy timeout, and uses one pooled connection. PostgreSQL builds a DSN from host, port, database, user, and password.

Use the interface rather than depending on the concrete manager:

```go
func (repo *Repo) Find(ctx context.Context, id string) (*Model, error) {
	var model Model
	tx := repo.storage.Conn(ctx)
	result := tx.First(&model, "id = ?", id)
	if result.Error != nil {
		return nil, errs.Dbf(result.Error)
	}

	return &model, nil
}
```

The repository should not write an HTTP response or decide how a missing entity
is presented to the client. Return the translated database error to the service;
the service can then turn a missing value into `errs.NotFoundf`, and the handler
can render it with `answer.Err`.

Transactions must use the callback context so nested repository calls share the transaction:

```go
err := storage.WithTx(ctx, func(txCtx context.Context) error {
	if err := repo.Create(txCtx, value); err != nil {
		return err
	}
	return audit.Log(txCtx, value.ID)
})
```

`WithTx` is a no-op for a nil callback and avoids nesting a new transaction when the context already carries one. Never discard the callback error.

Keep repository methods linear: acquire the connection, execute the query, handle
the error, and return the result. Avoid factories, long chained expressions, and
deeply nested transaction or query logic when a local variable makes the flow
clearer.
