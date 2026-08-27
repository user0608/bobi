package appenv

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/fx"
)

// BinaryPath is the absolute, symlink-resolved path to the running binary.
type BinaryPath string

// BinaryDir is the directory containing the running binary.
type BinaryDir string

var Module = fx.Module(
	"appenv",
	fx.Provide(
		resolveBinaryPath,
		binaryDir,
	),
)

func resolveBinaryPath() (BinaryPath, error) {
	path, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("get executable path: %w", err)
	}

	path, err = filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("make executable path absolute: %w", err)
	}

	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve executable symlinks: %w", err)
	}

	return BinaryPath(path), nil
}

func binaryDir(path BinaryPath) BinaryDir {
	return BinaryDir(filepath.Dir(string(path)))
}
