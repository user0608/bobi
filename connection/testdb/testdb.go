package testdb

import (
	"context"
	"fmt"
	"io/fs"

	"log/slog"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/user0608/bobi/connection"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	postgresImage    = "postgres:18-alpine"
	testDatabaseName = "testdb"
	testUsername     = "testuser"
	testPassword     = "testpass"
	testLogLevel     = "error"
)

var (
	postgresOnce sync.Once

	sharedStorage   connection.StorageManager
	sharedContainer *tcpostgres.PostgresContainer
	sharedErr       error
)

func NewPostgresStorage(t *testing.T, migrationsDir fs.FS) connection.StorageManager {
	t.Helper()

	postgresOnce.Do(func() {
		sharedStorage, sharedContainer, sharedErr = startPostgres()
	})

	require.NoError(t, sharedErr)
	require.NotNil(t, sharedStorage)

	require.NoError(t, runMigrations(sharedStorage, migrationsDir))

	t.Cleanup(func() {
		resetSchemas(t, sharedStorage)
	})

	return sharedStorage
}

func NewSQLiteStorage(t *testing.T, migrationsDir fs.FS) connection.StorageManager {
	t.Helper()

	storage, err := connection.NewConnection(connection.DatabaseConfig{
		Driver:   connection.DatabaseDriverSQLite,
		Database: filepath.Join(t.TempDir(), "test.db"),
		LogLevel: testLogLevel,
	})
	require.NoError(t, err)
	require.NotNil(t, storage)

	goose.SetDialect("sqlite3")
	require.NoError(t, runMigrations(storage, migrationsDir))

	return storage
}

func startPostgres() (connection.StorageManager, *tcpostgres.PostgresContainer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	container, err := tcpostgres.Run(
		ctx,
		postgresImage,
		tcpostgres.WithDatabase(testDatabaseName),
		tcpostgres.WithUsername(testUsername),
		tcpostgres.WithPassword(testPassword),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, nil, err
	}

	storage, err := newStorageFromContainer(ctx, container)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, nil, err
	}

	return storage, container, nil
}

func newStorageFromContainer(ctx context.Context, container *tcpostgres.PostgresContainer) (connection.StorageManager, error) {
	host, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	mappedPort, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(mappedPort.Port())
	if err != nil {
		return nil, err
	}

	return connection.NewConnection(connection.DatabaseConfig{
		Driver:   connection.DatabaseDriverPostgres,
		Host:     host,
		Port:     uint(port),
		User:     testUsername,
		Database: testDatabaseName,
		Password: testPassword,
		LogLevel: testLogLevel,
	})
}

func runMigrations(storage connection.StorageManager, migrationsDir fs.FS) error {
	if migrationsDir == nil {
		return nil
	}

	goose.SetBaseFS(migrationsDir)

	var ctx = context.TODO()
	tx := storage.Conn(ctx)
	db, err := tx.DB()
	if err != nil {
		return err
	}
	if err := goose.UpContext(ctx, db, "migrations"); err != nil {
		slog.Error(fmt.Sprintf("Database %s failed", "up"), "error", err)
		return err
	}
	return nil
}

func resetSchemas(t *testing.T, storage connection.StorageManager) {
	t.Helper()

	err := storage.Conn(context.Background()).Exec(`
		DO $$
		DECLARE
			schema_record RECORD;
		BEGIN
			FOR schema_record IN (
				SELECT schema_name
				FROM information_schema.schemata
				WHERE schema_name NOT IN ('public', 'information_schema')
				  AND schema_name NOT LIKE 'pg_%'
			) LOOP
				EXECUTE format('DROP SCHEMA IF EXISTS %I CASCADE', schema_record.schema_name);
			END LOOP;

			DROP SCHEMA IF EXISTS public CASCADE;
			CREATE SCHEMA public;
			GRANT ALL ON SCHEMA public TO public;
		END $$;
	`).Error
	require.NoError(t, err)
}
