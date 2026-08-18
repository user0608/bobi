package errs

import (
	"net/http"

	moderncsqlite "modernc.org/sqlite"
)

const (
	sqliteConstraint        = 19
	sqliteConstraintPrimary = 1555
	sqliteConstraintUnique  = 2067
	sqliteConstraintNotNull = 1299
	sqliteConstraintForeign = 787
	sqliteConstraintCheck   = 275
	sqliteBusy              = 5
	sqliteLocked            = 6
)

func sqliteError(err error, sqliteErr *moderncsqlite.Error) error {
	switch sqliteErr.Code() {
	case sqliteConstraintUnique, sqliteConstraintPrimary:
		return newError(nil, pgMessage(PgDuplicateRecordError), http.StatusBadRequest)
	case sqliteConstraintForeign:
		return newError(nil, message23503, http.StatusBadRequest)
	case sqliteConstraintNotNull:
		return newError(nil, pgMessage(PgNonNullableFieldsError), http.StatusBadRequest)
	case sqliteConstraintCheck:
		return newError(nil, pgMessage(PgInvalidFormatError), http.StatusBadRequest)
	case sqliteConstraint:
		return newError(nil, pgMessage(PgDataIntegrityError), http.StatusBadRequest)
	case sqliteBusy, sqliteLocked:
		return newError(err, ErrDatabase, http.StatusInternalServerError)
	default:
		mutex.RLock()
		developmentMode := devmode
		mutex.RUnlock()
		if developmentMode {
			return newError(err, ErrDatabase, http.StatusInternalServerError)
		}
		return newError(nil, ErrDatabase, http.StatusInternalServerError)
	}
}
