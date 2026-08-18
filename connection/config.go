package connection

import (
	"context"

	"gorm.io/gorm"
)

type DatabaseDriver string

const (
	DatabaseDriverSQLite   DatabaseDriver = "sqlite"
	DatabaseDriverPostgres DatabaseDriver = "postgres"
)

type DatabaseConfig struct {
	Driver   DatabaseDriver `mapstructure:"driver"`
	Host     string         `mapstructure:"host"`
	Port     uint           `mapstructure:"port"`
	Database string         `mapstructure:"database"`
	Username string         `mapstructure:"username"`
	Password string         `mapstructure:"password"`
	LogLevel string         `mapstructure:"log_level"`
}

type StorageManager interface {
	Conn(ctx context.Context) *gorm.DB
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
