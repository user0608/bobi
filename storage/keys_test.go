package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitBucket(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		value  string
		bucket string
		prefix string
	}{
		{name: "bucket root", value: "archivos", bucket: "archivos"},
		{name: "one prefix", value: "archivos/fotos", bucket: "archivos", prefix: "fotos/"},
		{name: "nested prefix", value: "archivos/fotos/usuarios", bucket: "archivos", prefix: "fotos/usuarios/"},
		{name: "trailing slash", value: "archivos/fotos/", bucket: "archivos", prefix: "fotos/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bucket, prefix, err := splitBucket(tt.value)
			require.NoError(t, err)
			require.Equal(t, tt.bucket, bucket)
			require.Equal(t, tt.prefix, prefix)
		})
	}
	for _, value := range []string{"", "/archivos/fotos", "archivos/../escape", "archivos//fotos", "archivos/a//b", "archivos/fotos//", "archivos//", "archivos/./b", "archivos\\fotos", "archivos/.storage-tmp", "a b/fotos"} {
		t.Run("invalid "+value, func(t *testing.T) {
			t.Parallel()
			_, _, err := splitBucket(value)
			require.Error(t, err)
		})
	}
	for _, value := range []string{"ab", "Files", "files?query", "files#fragment", "-files", "files-", "files..other", "127.0.0.1"} {
		t.Run("invalid bucket "+value, func(t *testing.T) {
			t.Parallel()
			_, _, err := splitBucket(value)
			require.Error(t, err)
		})
	}
}

func FuzzNormalizeKey(f *testing.F) {
	f.Add("/febrero/kevin.jpg")
	f.Add("../escape")
	f.Fuzz(func(t *testing.T, key string) {
		normalized, err := normalizeKey(key, false)
		if err != nil {
			return
		}
		again, err := normalizeKey(normalized, false)
		require.NoError(t, err)
		require.Equal(t, normalized, again)
	})
}
