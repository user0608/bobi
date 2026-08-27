package appenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBinaryPath(t *testing.T) {
	path, err := resolveBinaryPath()
	if err != nil {
		t.Fatalf("resolveBinaryPath() error = %v", err)
	}

	if !filepath.IsAbs(string(path)) {
		t.Fatalf("resolveBinaryPath() = %q, want an absolute path", path)
	}

	info, err := os.Stat(string(path))
	if err != nil {
		t.Fatalf("resolveBinaryPath() = %q, want an existing file: %v", path, err)
	}

	if info.IsDir() {
		t.Fatalf("resolveBinaryPath() = %q, want a file", path)
	}

	resolved, err := filepath.EvalSymlinks(string(path))
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", path, err)
	}

	if string(path) != resolved {
		t.Fatalf(
			"resolveBinaryPath() = %q, want symlink-resolved path %q",
			path,
			resolved,
		)
	}
}

func TestBinaryDir(t *testing.T) {
	tests := []struct {
		name string
		path BinaryPath
		want BinaryDir
	}{
		{
			name: "relative path",
			path: BinaryPath(filepath.Join("tmp", "bobi")),
			want: BinaryDir("tmp"),
		},
		{
			name: "nested path",
			path: BinaryPath(filepath.Join("usr", "local", "bin", "bobi")),
			want: BinaryDir(filepath.Join("usr", "local", "bin")),
		},
		{
			name: "current directory",
			path: BinaryPath("bobi"),
			want: BinaryDir("."),
		},
		{
			name: "absolute path",
			path: BinaryPath(filepath.Join(
				string(filepath.Separator),
				"usr",
				"bin",
				"bobi",
			)),
			want: BinaryDir(filepath.Join(
				string(filepath.Separator),
				"usr",
				"bin",
			)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := binaryDir(tt.path)

			if got != tt.want {
				t.Fatalf("binaryDir(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestBinaryDirOfResolvedBinaryExists(t *testing.T) {
	path, err := resolveBinaryPath()
	if err != nil {
		t.Fatalf("resolveBinaryPath() error = %v", err)
	}

	dir := binaryDir(path)

	info, err := os.Stat(string(dir))
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v", dir, err)
	}

	if !info.IsDir() {
		t.Fatalf("binaryDir(%q) = %q, want an existing directory", path, dir)
	}
}

func TestBinaryDirMatchesResolvedBinaryPath(t *testing.T) {
	path, err := resolveBinaryPath()
	if err != nil {
		t.Fatalf("resolveBinaryPath() error = %v", err)
	}

	got := binaryDir(path)
	want := BinaryDir(filepath.Dir(string(path)))

	if got != want {
		t.Fatalf("binaryDir(%q) = %q, want %q", path, got, want)
	}
}
