package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/user0608/bobi/setup/utils/sqlview"
)

const viewsDir = baseDir + "/_views"

type viewScript struct {
	path    string
	content []byte
}

type viewSet struct {
	views   []sqlview.View
	scripts []viewScript
}

func (mr *MigrationRunner) refreshViews(ctx context.Context, db *sql.DB, migrate func() error) error {
	views, err := mr.loadViews()
	if err != nil {
		return err
	}
	if views == nil {
		return migrate()
	}

	if err := dropViews(ctx, db, views.views); err != nil {
		return err
	}

	migrationErr := migrate()
	recreateErr := createViews(ctx, db, views.scripts)

	switch {
	case migrationErr != nil && recreateErr != nil:
		return errors.Join(migrationErr, recreateErr)
	case migrationErr != nil:
		return migrationErr
	default:
		return recreateErr
	}
}

func (mr *MigrationRunner) loadViews() (*viewSet, error) {
	info, err := fs.Stat(mr.migrationFS, viewsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat views directory %q: %w", viewsDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("views path %q is not a directory", viewsDir)
	}

	viewFS, err := fs.Sub(mr.migrationFS, viewsDir)
	if err != nil {
		return nil, fmt.Errorf("open views directory %q: %w", viewsDir, err)
	}

	views, err := sqlview.Views(viewFS)
	if err != nil {
		return nil, fmt.Errorf("read view definitions: %w", err)
	}
	scripts, err := readViewScripts(viewFS)
	if err != nil {
		return nil, err
	}

	return &viewSet{views: views, scripts: scripts}, nil
}

func readViewScripts(viewFS fs.FS) ([]viewScript, error) {
	scripts := []viewScript{}

	err := fs.WalkDir(viewFS, ".", func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk view SQL files at %q: %w", filePath, walkErr)
		}
		if entry.IsDir() || !strings.EqualFold(path.Ext(filePath), ".sql") {
			return nil
		}

		content, err := fs.ReadFile(viewFS, filePath)
		if err != nil {
			return fmt.Errorf("read view SQL file %q: %w", filePath, err)
		}
		scripts = append(scripts, viewScript{path: filePath, content: content})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return scripts, nil
}

func dropViews(ctx context.Context, db *sql.DB, views []sqlview.View) error {
	if len(views) == 0 {
		return nil
	}

	return runViewTransaction(ctx, db, func(tx *sql.Tx) error {
		for index := len(views) - 1; index >= 0; index-- {
			view := views[index]
			if _, err := tx.ExecContext(ctx, dropViewSQL(view)); err != nil {
				return fmt.Errorf("drop view %s: %w", view.Name, err)
			}
		}
		return nil
	})
}

func createViews(ctx context.Context, db *sql.DB, scripts []viewScript) error {
	if len(scripts) == 0 {
		return nil
	}

	return runViewTransaction(ctx, db, func(tx *sql.Tx) error {
		for _, script := range scripts {
			if _, err := tx.ExecContext(ctx, string(script.content)); err != nil {
				return fmt.Errorf("execute view SQL file %q: %w", script.path, err)
			}
		}
		return nil
	})
}

func runViewTransaction(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin view transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return errors.Join(err, fmt.Errorf("rollback view transaction: %w", rollbackErr))
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit view transaction: %w", err)
	}
	return nil
}

func dropViewSQL(view sqlview.View) string {
	if view.Materialized {
		return "DROP MATERIALIZED VIEW IF EXISTS " + view.Name
	}
	return "DROP VIEW IF EXISTS " + view.Name
}
