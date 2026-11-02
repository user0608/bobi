package setup

import (
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/user0608/bobi/storage"
	"go.uber.org/fx"
)

func TestStorageConfig(t *testing.T) {
	t.Parallel()
	v := viper.New()
	v.SetConfigType("yaml")
	require.NoError(t, v.ReadConfig(strings.NewReader(`
storage:
  driver: minio
  timeout: 2m
  minio:
    endpoint: http://localhost:9000
    bucket: archivos/fotos
    access_key: configured-key
    secret_key: configured-secret
`)))
	s := NewService(WithStorage())
	require.True(t, s.storageEnabled)
	require.False(t, NewService().storageEnabled)
	config, err := s.storageConfig(v)
	require.NoError(t, err)
	require.Equal(t, storage.DriverMinIO, config.Driver)
	require.Equal(t, "archivos/fotos", config.MinIO.Bucket)
	require.Equal(t, "configured-key", config.MinIO.AccessKey)
	require.Equal(t, "configured-secret", config.MinIO.SecretKey)
	require.Equal(t, 2*time.Minute, config.Timeout)
}

func TestStorageFxInjection(t *testing.T) {
	t.Parallel()
	v := viper.New()
	v.Set("storage.driver", "local")
	v.Set("storage.local.root", t.TempDir())
	s := NewService(WithStorage())
	var files storage.Storage
	app := fx.New(
		fx.Supply(v),
		fx.Provide(s.storageConfig, storage.NewStorage),
		fx.Populate(&files),
		fx.NopLogger,
	)
	require.NoError(t, app.Err())
	require.NotNil(t, files)
}
