package errs

import (
	"errors"
	"net/http"

	gosqlite "github.com/glebarez/go-sqlite"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// Dbf translates errors from either supported database backend.
func Dbf(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return newError(err, ErrRecordNotFound, http.StatusBadRequest)
	}
	if pgerr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return postgresError(err, pgerr)
	}
	if sqliteErr, ok := errors.AsType[*gosqlite.Error](err); ok {
		return sqliteError(err, sqliteErr)
	}
	return newError(err, ErrDatabase, http.StatusInternalServerError)
}
