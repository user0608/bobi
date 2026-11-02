package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
)

type localStorage struct {
	root string
}

var _ Storage = (*localStorage)(nil)

func newLocalStorage(config LocalConfig) (Storage, error) {
	if strings.TrimSpace(config.Root) == "" {
		return nil, fmt.Errorf("local storage root is required")
	}
	root, err := filepath.Abs(config.Root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return nil, fmt.Errorf("open storage root: %w", err)
	}
	if err := r.Close(); err != nil {
		return nil, err
	}
	return &localStorage{root: root}, nil
}

func (s *localStorage) Put(ctx context.Context, key string, body io.Reader, opts PutOptions) error {
	key, err := normalizeKey(key, false)
	if err != nil {
		return err
	}
	if body == nil {
		return fmt.Errorf("storage body is required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	r, err := os.OpenRoot(s.root)
	if err != nil {
		return err
	}
	defer r.Close()
	if err := r.MkdirAll(path.Dir(key), 0o700); err != nil {
		return err
	}
	if err := r.MkdirAll(internalDirectory, 0o700); err != nil {
		return err
	}
	temporary := internalDirectory + "/" + uuid.NewString()
	f, err := r.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer r.Remove(temporary)
	_, writeErr := io.Copy(f, contextReader{ctx: ctx, r: body})
	if writeErr == nil {
		writeErr = ctx.Err()
	}
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return fmt.Errorf("write storage object: %w", err)
	}
	// Root.Rename atomically replaces the destination without following its
	// final symlink; parent symlinks cannot escape the configured root.
	return r.Rename(temporary, key)
}

func (s *localStorage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	key, err := normalizeKey(key, false)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r, err := os.OpenRoot(s.root)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	f, err := r.Open(key)
	if err != nil {
		return nil, localError(err)
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		f.Close()
		if err != nil {
			return nil, localError(err)
		}
		return nil, ErrNotFound
	}
	return &contextReadCloser{contextReader: contextReader{ctx: ctx, r: f}, closer: f}, nil
}

func (s *localStorage) Delete(ctx context.Context, key string) error {
	key, err := normalizeKey(key, false)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	r, err := os.OpenRoot(s.root)
	if err != nil {
		return err
	}
	defer r.Close()
	info, err := r.Lstat(key)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil // Directories are not storage objects.
	}
	err = r.Remove(key)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func (s *localStorage) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	key, err := normalizeKey(key, false)
	if err != nil {
		return ObjectInfo{}, err
	}
	if err := ctx.Err(); err != nil {
		return ObjectInfo{}, err
	}
	r, err := os.OpenRoot(s.root)
	if err != nil {
		return ObjectInfo{}, err
	}
	defer r.Close()
	info, err := r.Stat(key)
	if err != nil {
		return ObjectInfo{}, localError(err)
	}
	if !info.Mode().IsRegular() {
		return ObjectInfo{}, ErrNotFound
	}
	return localInfo(key, info), nil
}

func (s *localStorage) List(ctx context.Context, opts ListOptions) (ListResult, error) {
	prefix, after, scope, limit, err := prepareList(opts, "local:"+s.root)
	if err != nil {
		return ListResult{}, err
	}
	if err := ctx.Err(); err != nil {
		return ListResult{}, err
	}
	r, err := os.OpenRoot(s.root)
	if err != nil {
		return ListResult{}, err
	}
	defer r.Close()
	result := ListResult{Objects: []ObjectInfo{}}
	err = fs.WalkDir(r.FS(), ".", func(key string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == internalDirectory {
				return fs.SkipDir
			}
			return nil
		}
		// Do not enumerate symlinks or special files. Root still protects Open
		// and Stat if a symlink is explicitly addressed.
		if !entry.Type().IsRegular() || !strings.HasPrefix(key, prefix) || key <= after {
			return nil
		}
		if _, err := normalizeKey(key, false); err != nil {
			return nil
		}
		index := sort.Search(len(result.Objects), func(i int) bool { return result.Objects[i].Key >= key })
		if index > limit {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		result.Objects = append(result.Objects, ObjectInfo{})
		copy(result.Objects[index+1:], result.Objects[index:])
		result.Objects[index] = localInfo(key, info)
		if len(result.Objects) > limit+1 {
			result.Objects = result.Objects[:limit+1]
		}
		return nil
	})
	if err != nil {
		return ListResult{}, err
	}
	if len(result.Objects) > limit {
		result.Objects = result.Objects[:limit]
		result.ContinuationToken = nextToken(scope, result.Objects[limit-1].Key)
	}
	return result, nil
}

func (s *localStorage) Copy(ctx context.Context, sourceKey, destinationKey string) error {
	if _, err := normalizeKey(destinationKey, false); err != nil {
		return err
	}
	body, err := s.Open(ctx, sourceKey)
	if err != nil {
		return err
	}
	defer body.Close()
	return s.Put(ctx, destinationKey, body, PutOptions{})
}

func localInfo(key string, info fs.FileInfo) ObjectInfo {
	return ObjectInfo{Key: key, Size: info.Size(), LastModified: info.ModTime()}
}

func localError(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w: %w", ErrNotFound, err)
	}
	return err
}
