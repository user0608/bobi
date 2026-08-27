package connection

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func sqliteDialector(config DatabaseConfig) gorm.Dialector {
	dsn := config.Database

	isDir := strings.HasSuffix(dsn, "/") || strings.HasSuffix(dsn, `\`)

	if isDir {
		if err := os.MkdirAll(dsn, 0755); err != nil {
			panic(err)
		}

		dsn = filepath.Join(dsn, "database.db")
	}

	values := url.Values{}
	values.Add("_pragma", "journal_mode(WAL)")
	values.Add("_pragma", "busy_timeout(5000)")
	values.Add("_pragma", "foreign_keys(ON)")

	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}

	return sqlite.Open(dsn + separator + values.Encode())
}
