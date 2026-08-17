package connection

import (
	"context"

	"gorm.io/gorm"
)

// skipStorage disables persistence using the null object pattern.
type skipStorage struct{}

var _ StorageManager = (*skipStorage)(nil)

func SkipStorage() (StorageManager, error) {
	return &skipStorage{}, nil
}

func (*skipStorage) Conn(ctx context.Context) *gorm.DB {
	return nil
}

func (*skipStorage) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if fn == nil {
		return nil
	}

	return fn(ctx)
}
