package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewStorageValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		config Config
	}{
		{name: "missing driver"},
		{name: "unknown driver", config: Config{Driver: "ftp"}},
		{name: "negative timeout", config: Config{Driver: DriverLocal, Timeout: -1}},
		{name: "missing local root", config: Config{Driver: DriverLocal}},
		{name: "missing bucket", config: Config{Driver: DriverS3, S3: S3Config{Region: "us-east-1"}}},
		{name: "missing region", config: Config{Driver: DriverS3, S3: S3Config{Bucket: "files"}}},
		{name: "partial aws credentials", config: Config{Driver: DriverS3, S3: S3Config{Bucket: "files", Region: "us-east-1", AccessKey: "key"}}},
		{name: "token without credentials", config: Config{Driver: DriverS3, S3: S3Config{Bucket: "files", Region: "us-east-1", SessionToken: "token"}}},
		{name: "missing minio endpoint", config: Config{Driver: DriverMinIO, MinIO: MinIOConfig{Bucket: "files"}}},
		{name: "insecure endpoint syntax", config: Config{Driver: DriverMinIO, MinIO: MinIOConfig{Bucket: "files", Endpoint: "ftp://localhost"}}},
		{name: "credentials in endpoint", config: Config{Driver: DriverMinIO, MinIO: MinIOConfig{Bucket: "files", Endpoint: "http://user:pass@localhost"}}},
		{name: "missing minio credentials", config: Config{Driver: DriverMinIO, MinIO: MinIOConfig{Bucket: "files", Endpoint: "http://localhost:9000"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewStorage(tt.config)
			require.Error(t, err)
		})
	}
}

// The same observable contract is exercised against local storage, the SDK's
// HTTP wire implementation, and real services in integration-tagged tests.
func storageContract(t *testing.T, files Storage) {
	t.Helper()
	ctx := t.Context()
	opts := PutOptions{ContentType: "image/jpeg"}
	require.NoError(t, files.Put(ctx, "/febrero/kevin.jpg", strings.NewReader("photo"), opts))
	body, err := files.Open(ctx, "febrero/kevin.jpg")
	require.NoError(t, err)
	data, readErr := io.ReadAll(body)
	require.NoError(t, body.Close())
	require.NoError(t, readErr)
	require.Equal(t, "photo", string(data))

	info, err := files.Stat(ctx, "/febrero/kevin.jpg")
	require.NoError(t, err)
	require.Equal(t, "febrero/kevin.jpg", info.Key)
	require.EqualValues(t, 5, info.Size)
	require.False(t, info.LastModified.IsZero())

	require.NoError(t, files.Copy(ctx, "/febrero/kevin.jpg", "/febrero/copy.jpg"))
	require.NoError(t, files.Copy(ctx, "febrero/copy.jpg", "febrero/copy.jpg"))
	require.NoError(t, files.Put(ctx, "febrero/kevin.jpg", strings.NewReader("replacement"), opts))
	info, err = files.Stat(ctx, "febrero/kevin.jpg")
	require.NoError(t, err)
	require.EqualValues(t, 11, info.Size)
	info, err = files.Stat(ctx, "febrero/copy.jpg")
	require.NoError(t, err)
	require.EqualValues(t, 5, info.Size)

	page, err := files.List(ctx, ListOptions{Prefix: "/febrero/", Limit: 1})
	require.NoError(t, err)
	require.Len(t, page.Objects, 1)
	require.Equal(t, "febrero/copy.jpg", page.Objects[0].Key)
	require.NotEmpty(t, page.ContinuationToken)
	_, err = files.List(ctx, ListOptions{Prefix: "other/", ContinuationToken: page.ContinuationToken})
	require.ErrorIs(t, err, ErrInvalidToken)
	page, err = files.List(ctx, ListOptions{Prefix: "febrero/", Limit: 1, ContinuationToken: page.ContinuationToken})
	require.NoError(t, err)
	require.Len(t, page.Objects, 1)
	require.Equal(t, "febrero/kevin.jpg", page.Objects[0].Key)
	require.Empty(t, page.ContinuationToken)
	page, err = files.List(ctx, ListOptions{Prefix: "absent/"})
	require.NoError(t, err)
	require.Empty(t, page.Objects)

	require.NoError(t, files.Delete(ctx, "/febrero/kevin.jpg"))
	require.NoError(t, files.Delete(ctx, "febrero/kevin.jpg"))
	_, err = files.Open(ctx, "febrero/kevin.jpg")
	require.ErrorIs(t, err, ErrNotFound)
	_, err = files.Stat(ctx, "febrero/kevin.jpg")
	require.ErrorIs(t, err, ErrNotFound)
	require.ErrorIs(t, files.Copy(ctx, "missing", "destination"), ErrNotFound)
	require.Error(t, files.Put(ctx, "empty", nil, PutOptions{}))

	for _, key := range []string{"", "/", "../escape", "a/../../escape", "a/../b", "a/./b", `a\b`, "a\x00b", "a//b", "a/", ".storage-tmp/secret"} {
		t.Run("invalid key "+key, func(t *testing.T) {
			require.ErrorIs(t, files.Put(ctx, key, strings.NewReader("x"), opts), ErrInvalidKey)
			_, err := files.Open(ctx, key)
			require.ErrorIs(t, err, ErrInvalidKey)
			require.ErrorIs(t, files.Delete(ctx, key), ErrInvalidKey)
			_, err = files.Stat(ctx, key)
			require.ErrorIs(t, err, ErrInvalidKey)
			require.ErrorIs(t, files.Copy(ctx, "febrero/copy.jpg", key), ErrInvalidKey)
			require.ErrorIs(t, files.Copy(ctx, key, "valid"), ErrInvalidKey)
		})
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	require.ErrorIs(t, files.Put(canceled, "canceled", strings.NewReader("x"), opts), context.Canceled)
	_, err = files.Open(canceled, "febrero/copy.jpg")
	require.ErrorIs(t, err, context.Canceled)
	_, err = files.Stat(canceled, "febrero/copy.jpg")
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, files.Delete(canceled, "febrero/copy.jpg"), context.Canceled)
	_, err = files.List(canceled, ListOptions{})
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, files.Copy(canceled, "febrero/copy.jpg", "canceled"), context.Canceled)

	_, err = files.List(ctx, ListOptions{Prefix: "../"})
	require.ErrorIs(t, err, ErrInvalidKey)
	_, err = files.List(ctx, ListOptions{Limit: -1})
	require.Error(t, err)
	_, err = files.List(ctx, ListOptions{Limit: 1001})
	require.Error(t, err)
	_, err = files.List(ctx, ListOptions{ContinuationToken: "invalid"})
	require.ErrorIs(t, err, ErrInvalidToken)

	// A failed source stream must not replace the previous object.
	err = files.Put(ctx, "febrero/copy.jpg", failingReader{}, opts)
	require.ErrorIs(t, err, errSource)
	info, err = files.Stat(ctx, "febrero/copy.jpg")
	require.NoError(t, err)
	require.EqualValues(t, 5, info.Size)
}

var errSource = errors.New("source failed")

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errSource }
