package connection

import (
	"context"

	"gorm.io/gorm"
)

type StorageBackend string

const (
	BackendPostgres StorageBackend = "postgres"
	BackendSQLite   StorageBackend = "sqlite"
	BackendNone     StorageBackend = "none"
)

type DBConfigParams struct {
	Backend  StorageBackend
	Host     string
	Port     uint
	Database string
	Username string
	Password string
	LogLevel string
}

type StorageManager interface {
	Conn(ctx context.Context) *gorm.DB
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}
