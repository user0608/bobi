package migrations

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/pressly/goose/v3"
	"github.com/user0608/bobi/connection"
)

const baseDir = "migrations"

type MigrationFS fs.FS

type MigrationRunner struct {
	storageManager connection.StorageManager
	migrationFS    fs.FS
}

var gooseMu sync.Mutex

func NewMigrationRunner(
	storageManager connection.StorageManager,
	mfs MigrationFS,
) *MigrationRunner {

	return &MigrationRunner{
		storageManager: storageManager,
		migrationFS:    mfs,
	}
}

func NewMigrationScriptRunner(mfs MigrationFS) *MigrationRunner {
	return &MigrationRunner{migrationFS: mfs}
}

func (mr *MigrationRunner) SQLScript() (string, error) {
	if mr.migrationFS == nil {
		return "", nil
	}

	entries, err := fs.ReadDir(mr.migrationFS, baseDir)
	if err != nil {
		return "", fmt.Errorf("read migrations dir: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	rgx := regexp.MustCompile(`(?s)-- \+goose Up(.*?)(?:-- \+goose Down|$)`)

	var output strings.Builder

	for _, entry := range entries {
		if entry.IsDir() || path.Ext(entry.Name()) != ".sql" {
			continue
		}

		filePath := path.Join(baseDir, entry.Name())

		content, err := fs.ReadFile(mr.migrationFS, filePath)
		if err != nil {
			return "", fmt.Errorf("read migration file %q: %w", filePath, err)
		}

		match := rgx.FindSubmatch(content)
		if match == nil {
			return "", fmt.Errorf("migration file %q has no goose Up section", filePath)
		}

		fmt.Fprintf(&output, "-- %s\n", entry.Name())

		scanner := bufio.NewScanner(bytes.NewReader(match[1]))
		for scanner.Scan() {
			line := bytes.TrimSpace(scanner.Bytes())

			if len(line) == 0 || bytes.HasPrefix(line, []byte("--")) {
				continue
			}

			output.Write(line)
			output.WriteByte('\n')
		}

		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("scan migration file %q: %w", filePath, err)
		}
	}

	return output.String(), nil
}

func (mr *MigrationRunner) setupGoose(ctx context.Context) (*sql.DB, error) {
	if mr.storageManager == nil {
		return nil, errors.New("migration storage manager is required")
	}

	tx := mr.storageManager.Conn(ctx)
	db, err := tx.DB()
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (mr *MigrationRunner) runGoose(ctx context.Context, action string, fn func(*sql.DB) error) error {
	gooseMu.Lock()
	defer gooseMu.Unlock()

	if mr.migrationFS != nil {
		goose.SetBaseFS(mr.migrationFS)
	}

	db, err := mr.setupGoose(ctx)
	if err != nil {
		return err
	}
	if err := fn(db); err != nil {
		slog.Error("database migration failed", "action", action, "error", err)
		return err
	}
	return nil
}

func (mr *MigrationRunner) Up(ctx context.Context) error {
	if mr.migrationFS == nil {
		return nil
	}

	return mr.runGoose(ctx, "up", func(db *sql.DB) error {
		return goose.UpContext(ctx, db, baseDir)
	})
}

func (mr *MigrationRunner) Down(ctx context.Context) error {
	if mr.migrationFS == nil {
		return nil
	}
	return mr.runGoose(ctx, "down", func(db *sql.DB) error {
		return goose.DownContext(ctx, db, baseDir)
	})
}

func (mr *MigrationRunner) Status(ctx context.Context) error {
	if mr.migrationFS == nil {
		return nil
	}
	return mr.runGoose(ctx, "status", func(db *sql.DB) error {
		return goose.StatusContext(ctx, db, baseDir)
	})
}
