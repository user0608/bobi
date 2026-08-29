// Package sqlview extracts view identifiers from SQLite and PostgreSQL files.
package sqlview

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// ViewNames returns the SQL identifiers of every view declared in SQL files.
// It walks viewFS recursively and preserves identifier quoting and schema
// qualification so each returned value can be used directly in a query.
// Results follow filesystem lexical order, and exact duplicates appear once.
func ViewNames(viewFS fs.FS) ([]string, error) {
	if viewFS == nil {
		return nil, errors.New("view filesystem is required")
	}

	names := []string{}
	seen := map[string]struct{}{}

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

		fileNames, err := parseViewNames(content)
		if err != nil {
			return fmt.Errorf("parse SQL file %q: %w", filePath, err)
		}

		for _, name := range fileNames {
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			names = append(names, name)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return names, nil
}
