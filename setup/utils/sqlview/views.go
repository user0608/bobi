// Package sqlview extracts view identifiers from SQLite and PostgreSQL files.
package sqlview

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// View describes a view declared in a SQL file.
type View struct {
	Name         string
	Materialized bool
}

// Views returns every view declared in SQL files. It walks viewFS recursively
// and preserves identifier quoting and schema qualification so each name can
// be used directly in a query. Results follow filesystem lexical order, and
// exact duplicates appear once.
func Views(viewFS fs.FS) ([]View, error) {
	if viewFS == nil {
		return nil, errors.New("view filesystem is required")
	}

	views := []View{}
	seen := map[string]bool{}

	err := fs.WalkDir(viewFS, ".", func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk %q: %w", filePath, walkErr)
		}
		if entry.IsDir() || !strings.EqualFold(path.Ext(filePath), ".sql") {
			return nil
		}

		content, err := fs.ReadFile(viewFS, filePath)
		if err != nil {
			return fmt.Errorf("read SQL file %q: %w", filePath, err)
		}

		fileViews, err := parseViews(content)
		if err != nil {
			return fmt.Errorf("parse SQL file %q: %w", filePath, err)
		}

		for _, view := range fileViews {
			materialized, ok := seen[view.Name]
			if ok && materialized != view.Materialized {
				return fmt.Errorf("view %s is declared as both regular and materialized", view.Name)
			}
			if ok {
				continue
			}
			seen[view.Name] = view.Materialized
			views = append(views, view)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return views, nil
}
