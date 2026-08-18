package connection

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func postgresDialector(config DatabaseConfig) gorm.Dialector {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=disable",
		config.Host,
		config.Username,
		config.Password,
		config.Database,
		config.Port,
	)

	return postgres.Open(dsn)
}
