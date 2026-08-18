package connection

import (
	"net/url"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func sqliteDialector(config DBConfigParams) gorm.Dialector {
	values := url.Values{}
	values.Add("_pragma", "journal_mode(WAL)")
	values.Add("_pragma", "busy_timeout(5000)")
	values.Add("_pragma", "foreign_keys(ON)")

	dsn := config.Database
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	return sqlite.Open(dsn + separator + values.Encode())
}
