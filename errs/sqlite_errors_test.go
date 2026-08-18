package errs

import (
	"database/sql"
	"errors"
	"testing"

	moderncsqlite "modernc.org/sqlite"
)

func TestDbfSQLiteUniqueConstraint(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err = db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT UNIQUE)`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO users (email) VALUES (?)`, "a@example.com"); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO users (email) VALUES (?)`, "a@example.com")
	if err == nil {
		t.Fatal("expected unique constraint error")
	}

	if sqliteErr, ok := errors.AsType[*moderncsqlite.Error](err); !ok {
		t.Fatalf("expected SQLite error, got %T", err)
	} else if sqliteErr.Code() != sqliteConstraintUnique {
		t.Fatalf("expected SQLite unique code %d, got %d", sqliteConstraintUnique, sqliteErr.Code())
	}

	translated := Dbf(err)
	if translated == nil || translated.(*Err).Code() != 400 {
		t.Fatalf("expected translated bad request, got %v", translated)
	}
	if translated.(*Err).Wrapped() != nil {
		t.Fatal("expected non-loggable SQLite constraint to hide the driver error")
	}
}

func TestDbfSQLiteForeignKeyConstraint(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err = db.Exec(`PRAGMA foreign_keys = ON; CREATE TABLE parent (id INTEGER PRIMARY KEY); CREATE TABLE child (parent_id INTEGER REFERENCES parent(id))`); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO child (parent_id) VALUES (1)`)
	if err == nil {
		t.Fatal("expected foreign key constraint error")
	}

	translated := Dbf(err)
	if translated == nil || translated.(*Err).Message() != message23503 {
		t.Fatalf("expected foreign key message, got %v", translated)
	}
}
