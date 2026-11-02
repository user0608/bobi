package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalStorage(t *testing.T) {
	t.Parallel()
	root := filepath.Join(t.TempDir(), "archivos", "fotos")
	files, err := NewStorage(Config{Driver: DriverLocal, Local: LocalConfig{Root: root}, S3: S3Config{Bucket: "invalid/.."}})
	require.NoError(t, err)
	storageContract(t, files)
	data, err := os.ReadFile(filepath.Join(root, "febrero", "copy.jpg"))
	require.NoError(t, err)
	require.Equal(t, "photo", string(data))
}

func TestLocalStorageRejectsSymlinkEscapes(t *testing.T) {
	t.Parallel()
	root, outside := t.TempDir(), t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret"), []byte("untouched"), 0o600))
	require.NoError(t, os.Symlink(outside, filepath.Join(root, "escape")))
	require.NoError(t, os.Symlink(filepath.Join(outside, "secret"), filepath.Join(root, "link")))
	files, err := NewStorage(Config{Driver: DriverLocal, Local: LocalConfig{Root: root}})
	require.NoError(t, err)
	_, err = files.Open(t.Context(), "escape/secret")
	require.Error(t, err)
	_, err = files.Stat(t.Context(), "escape/secret")
	require.Error(t, err)
	require.Error(t, files.Put(t.Context(), "escape/secret", strings.NewReader("changed"), PutOptions{}))
	require.Error(t, files.Delete(t.Context(), "escape/secret"))
	_, err = files.Open(t.Context(), "link")
	require.Error(t, err)
	page, err := files.List(t.Context(), ListOptions{})
	require.NoError(t, err)
	require.Empty(t, page.Objects)
	// Replacing the final symlink is safe and does not overwrite its target.
	require.NoError(t, files.Put(t.Context(), "link", strings.NewReader("replacement"), PutOptions{}))
	data, err := os.ReadFile(filepath.Join(outside, "secret"))
	require.NoError(t, err)
	require.Equal(t, "untouched", string(data))
}

func TestLocalListSortAndPagination(t *testing.T) {
	t.Parallel()
	files, err := NewStorage(Config{Driver: DriverLocal, Local: LocalConfig{Root: t.TempDir()}})
	require.NoError(t, err)
	keys := []string{"a.txt", "a/b.txt", "a/z.txt", "z.txt"}
	for _, key := range keys {
		require.NoError(t, files.Put(t.Context(), key, strings.NewReader(key), PutOptions{}))
	}
	var got []string
	var token string
	for {
		page, err := files.List(t.Context(), ListOptions{Limit: 1, ContinuationToken: token})
		require.NoError(t, err)
		for _, info := range page.Objects {
			got = append(got, info.Key)
		}
		token = page.ContinuationToken
		if token == "" {
			break
		}
	}
	require.Equal(t, keys, got)
	_, err = files.Stat(t.Context(), "a")
	require.ErrorIs(t, err, ErrNotFound)
	_, err = files.Open(t.Context(), "a")
	require.ErrorIs(t, err, ErrNotFound)
	require.NoError(t, files.Delete(t.Context(), "a"))
}

func TestLocalOpenCancellation(t *testing.T) {
	t.Parallel()
	files, err := NewStorage(Config{Driver: DriverLocal, Local: LocalConfig{Root: t.TempDir()}})
	require.NoError(t, err)
	require.NoError(t, files.Put(t.Context(), "file", strings.NewReader("data"), PutOptions{}))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	body, err := files.Open(ctx, "file")
	require.NoError(t, err)
	cancel()
	_, err = io.ReadAll(body)
	require.ErrorIs(t, err, context.Canceled)
	require.NoError(t, body.Close())
}
