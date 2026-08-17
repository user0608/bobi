package connection

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func sqliteDialector(config DBConfigParams) gorm.Dialector {
	return sqlite.Open(config.Database)
}
