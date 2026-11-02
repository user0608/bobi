// Package storage provides rooted file and object storage for local filesystems,
// AWS S3, and MinIO. Keys are always relative to the configured root.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

// Driver selects a storage backend. Its zero value is invalid.
type Driver string

const (
	DriverLocal Driver = "local"
	DriverS3    Driver = "s3"
	DriverMinIO Driver = "minio"
)

var (
	ErrInvalidKey   = errors.New("invalid storage key")
	ErrNotFound     = errors.New("storage object not found")
	ErrInvalidToken = errors.New("invalid storage continuation token")
)

// Config selects one backend; only the selected backend's section is validated.
type Config struct {
	Driver  Driver        `mapstructure:"driver"`
	Local   LocalConfig   `mapstructure:"local"`
	S3      S3Config      `mapstructure:"s3"`
	MinIO   MinIOConfig   `mapstructure:"minio"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type LocalConfig struct {
	Root string `mapstructure:"root"`
}

// S3Config accepts bucket or bucket/prefix in Bucket. Omit both credential
// fields to use the AWS SDK's standard credential chain, including IAM roles.
type S3Config struct {
	Bucket       string `mapstructure:"bucket"`
	Region       string `mapstructure:"region"`
	AccessKey    string `mapstructure:"access_key"`
	SecretKey    string `mapstructure:"secret_key"`
	SessionToken string `mapstructure:"session_token"`
}

// MinIOConfig requires static credentials and an HTTP(S) endpoint. Region
// defaults to us-east-1. Bucket accepts bucket or bucket/prefix.
type MinIOConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	Bucket    string `mapstructure:"bucket"`
	Region    string `mapstructure:"region"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
}

// Storage operates on slash-separated keys beneath one configured root.
// Leading slashes are ignored; empty keys, dot segments, backslashes, and the
// reserved .storage-tmp segment are rejected. Methods are concurrency-safe.
type Storage interface {
	// Put replaces an existing object only after a successful upload.
	Put(ctx context.Context, key string, body io.Reader, opts PutOptions) error
	// Open returns a stream that the caller must close, even after read errors.
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete succeeds when the object does not exist.
	Delete(ctx context.Context, key string) error
	// Stat returns ErrNotFound for missing objects or local directories.
	Stat(ctx context.Context, key string) (ObjectInfo, error)
	// List recursively lists objects by key prefix, in lexicographical order.
	List(ctx context.Context, opts ListOptions) (ListResult, error)
	// Copy replaces the destination within this storage; it is not a move.
	Copy(ctx context.Context, sourceKey, destinationKey string) error
}

type PutOptions struct {
	// ContentType is persisted by S3/MinIO. Local storage has no persistent
	// content-type metadata and ignores this field.
	ContentType string
}

// ObjectInfo contains a relative key and available object metadata.
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	// ContentType is populated by remote Stat; List and local Stat omit it.
	ContentType string
}

type ListOptions struct {
	// Prefix is a key prefix, not necessarily a directory. Empty lists all keys.
	Prefix string
	// Limit defaults to 100 and must be between 1 and 1000 when specified.
	Limit int
	// ContinuationToken is opaque and scoped to this storage and Prefix.
	ContinuationToken string
}

type ListResult struct {
	Objects           []ObjectInfo
	ContinuationToken string
}

// NewStorage validates configuration and constructs the selected backend. It
// creates the local root, but does not create or probe remote buckets.
// Timeout bounds remote operations (including an Open stream's lifetime) and
// defaults to five minutes. Caller deadlines may shorten it.
func NewStorage(config Config) (Storage, error) {
	if config.Timeout < 0 {
		return nil, fmt.Errorf("storage timeout must not be negative")
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}
	switch config.Driver {
	case DriverLocal:
		return newLocalStorage(config.Local)
	case DriverS3:
		return newS3Storage(config.S3, config.Timeout)
	case DriverMinIO:
		return newMinIOStorage(config.MinIO, config.Timeout)
	default:
		return nil, fmt.Errorf("unsupported storage driver %q; supported drivers are local, s3, minio", config.Driver)
	}
}
