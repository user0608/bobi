package errs

import (
	"errors"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestDbfNil(t *testing.T) {
	if got := Dbf(nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestDbfRecordNotFound(t *testing.T) {
	err := Dbf(errors.New("record not found"))

	if err == nil {
		t.Fatal("expected error")
	}

	// Ajusta estos asserts según la API real de tu custom error.
	if err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
}

func TestDbfKnownPgErrorNotLoggable(t *testing.T) {
	err := &pgconn.PgError{
		Code:    string(PgDuplicateRecordError),
		Message: "duplicate key value violates unique constraint",
	}

	got := Dbf(err)

	if got == nil {
		t.Fatal("expected error")
	}

	if got.Error() == "" {
		t.Fatal("expected non-empty error")
	}
}

func TestDbfForeignKeyInsertOrUpdate(t *testing.T) {
	err := &pgconn.PgError{
		Code:    string(PgDependentRecordsError),
		Message: "insert or update on table violates foreign key constraint",
	}

	got := Dbf(err)

	if got == nil {
		t.Fatal("expected error")
	}

	if got.Error() == "" {
		t.Fatal("expected non-empty error")
	}
}

func TestDbfUnknownPgError(t *testing.T) {
	err := &pgconn.PgError{
		Code:    "99999",
		Message: "unknown pg error",
	}

	got := Dbf(err)

	if got == nil {
		t.Fatal("expected error")
	}

	if got.Error() == "" {
		t.Fatal("expected non-empty error")
	}
}

func TestAddPgErrs(t *testing.T) {
	const customCode PGCode = "ZZ999"
	const customMsg = "custom postgres error"

	AddPgErrs(customCode, customMsg, http.StatusTeapot, false)

	err := &pgconn.PgError{
		Code:    string(customCode),
		Message: "custom db error",
	}

	got := Dbf(err)

	if got == nil {
		t.Fatal("expected error")
	}

	if got.Error() == "" {
		t.Fatal("expected non-empty error")
	}
}

func TestDbfDevmodeKeepsOriginalErrorLoggable(t *testing.T) {

	err := &pgconn.PgError{
		Code:    string(PgDuplicateRecordError),
		Message: "duplicate key value violates unique constraint",
	}

	got := Dbf(err)

	if got == nil {
		t.Fatal("expected error")
	}

	if got.Error() == "" {
		t.Fatal("expected non-empty error")
	}
}

func TestIsPgErrCodeTrue(t *testing.T) {
	err := &pgconn.PgError{
		Code: string(PgDuplicateRecordError),
	}

	if !IsPgErrCode(err, PgDuplicateRecordError) {
		t.Fatal("expected true")
	}
}

func TestIsPgErrCodeFalseForDifferentCode(t *testing.T) {
	err := &pgconn.PgError{
		Code: string(PgDuplicateRecordError),
	}

	if IsPgErrCode(err, PgDependentRecordsError) {
		t.Fatal("expected false")
	}
}

func TestIsPgErrCodeFalseForNonPgError(t *testing.T) {
	err := errors.New("regular error")

	if IsPgErrCode(err, PgDuplicateRecordError) {
		t.Fatal("expected false")
	}
}
