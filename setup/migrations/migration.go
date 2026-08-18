package migrations

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/pressly/goose/v3"
	"github.com/user0608/bobi/connection"
)

const baseDir = "migrations"

type MigrationFS fs.FS

type MigrationRunner struct {
	storageManager connection.StorageManager
	migrationFS    fs.FS
}

func NewMigrationRunner(
	storageManager connection.StorageManager,
	mfs MigrationFS,
) *MigrationRunner {

	return &MigrationRunner{
		storageManager: storageManager,
		migrationFS:    mfs,
	}
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

	rgx := regexp.MustCompile(`(?s)-- \+goose Up(.*?)-- \+goose Down`)

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
			continue
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

	tx := mr.storageManager.Conn(ctx)
	db, err := tx.DB()
	if err != nil {
		return nil, err
	}
	if mr.migrationFS != nil {
		goose.SetBaseFS(mr.migrationFS)
	}

	return db, nil
}

func (mr *MigrationRunner) Up(ctx context.Context) error {
	if mr.migrationFS == nil {
		return nil
	}

	db, err := mr.setupGoose(ctx)
	if err != nil {
		return err
	}
	if err := goose.UpContext(ctx, db, baseDir); err != nil {
		slog.Error(fmt.Sprintf("Database %s failed", "up"), "error", err)
		return err
	}
	return nil
}

func (mr *MigrationRunner) Down(ctx context.Context) error {
	if mr.migrationFS == nil {
		return nil
	}
	db, err := mr.setupGoose(ctx)
	if err != nil {
		return err
	}
	if err := goose.DownContext(ctx, db, baseDir); err != nil {
		slog.Error(fmt.Sprintf("Database %s failed", "up"), "error", err)
		return err
	}
	return nil
}

func (mr *MigrationRunner) Status(ctx context.Context) error {
	if mr.migrationFS == nil {
		return nil
	}
	db, err := mr.setupGoose(ctx)
	if err != nil {
		return err
	}
	if err := goose.StatusContext(ctx, db, baseDir); err != nil {
		slog.Error(fmt.Sprintf("Database %s failed", "up"), "error", err)
		return err
	}
	return nil
}
