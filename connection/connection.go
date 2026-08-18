package connection

import (
	"context"
	"fmt"
	"log"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type storageManager struct {
	db *gorm.DB
}

var _ StorageManager = (*storageManager)(nil)

func NewConnection(config DatabaseConfig) (StorageManager, error) {
	var dialector gorm.Dialector
	switch config.Driver {
	case DatabaseDriverPostgres:
		dialector = postgresDialector(config)
	case DatabaseDriverSQLite:
		dialector = sqliteDialector(config)
	case DatabaseDriver(""):
		return nil, fmt.Errorf(
			"database driver is required; supported drivers are %q and %q",
			DatabaseDriverPostgres,
			DatabaseDriverSQLite,
		)

	default:
		return nil, fmt.Errorf(
			"unsupported database driver %q; supported drivers are %q and %q",
			config.Driver,
			DatabaseDriverPostgres,
			DatabaseDriverSQLite,
		)
	}

	db, err := openConnection(dialector, config.LogLevel)
	if err != nil {
		return nil, err
	}

	return &storageManager{db: db}, nil
}

func openConnection(dialector gorm.Dialector, logLevel string) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, &gorm.Config{
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(parseLogLevel(logLevel)),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		return nil, err
	}

	if dialector.Name() == string(DatabaseDriverSQLite) {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		// SQLite permits one writer at a time. A single pooled connection avoids
		// per-connection PRAGMA differences and lets busy_timeout do its job.
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
	}

	log.Println("Database connection established successfully.")

	return db, nil
}

func parseLogLevel(value string) logger.LogLevel {
	switch value {
	case "info":
		return logger.Info
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	default:
		return logger.Silent
	}
}

type txContextKey struct{}

var transactionKey txContextKey

func (s *storageManager) Conn(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(transactionKey).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}

	return s.db.WithContext(ctx)
}

func (s *storageManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if fn == nil {
		return nil
	}

	if _, ok := ctx.Value(transactionKey).(*gorm.DB); ok {
		return fn(ctx)
	}

	return s.Conn(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, transactionKey, tx)
		return fn(txCtx)
	})
}
