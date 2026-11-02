---
name: storage
description: Use Bobi storage for local files, AWS S3, or MinIO behind one interface. Trigger when using storage.Storage, NewStorage, WithStorage, file uploads, bucket prefixes, or storage configuration.
---

# Bobi Storage

Import `github.com/user0608/bobi/storage`. Inject `storage.Storage` into
application services; do not depend on a concrete backend or confuse this
interface with the database-oriented `connection.StorageManager`.

## Configuration

Select one driver through the Viper `storage` section. Only the selected
backend's configuration is required and validated. Missing or unknown drivers
return an error; there is no implicit local fallback.

Local storage:

```yaml
storage:
  driver: local
  local:
    root: "./data/archivos/fotos"
```

`local.root` is required and created if necessary.

AWS S3:

```yaml
storage:
  driver: s3
  timeout: 5m
  s3:
    bucket: "archivos/fotos"
    region: "us-east-1"
    access_key: "YOUR_ACCESS_KEY"
    secret_key: "YOUR_SECRET_KEY"
    # session_token: "YOUR_TEMPORARY_SESSION_TOKEN"
```

`s3.bucket` and `s3.region` are required. Explicit `access_key` and `secret_key`
must be configured together. `session_token` is optional and requires both.
Omit the credential fields to use the standard AWS credential chain, including
profiles, environment variables, and IAM roles. Explicit credentials take
precedence. Use general-purpose S3 buckets, not directory buckets or access-point
ARNs.

MinIO:

```yaml
storage:
  driver: minio
  minio:
    endpoint: "http://localhost:9000"
    bucket: "archivos/fotos"
    access_key: "YOUR_ACCESS_KEY"
    secret_key: "YOUR_SECRET_KEY"
    # region: "us-east-1"
```

`minio.endpoint`, `bucket`, `access_key`, and `secret_key` are required. Region
defaults to `us-east-1`. The endpoint is an HTTP(S) origin without embedded
credentials, path, query, or fragment. Use HTTPS in production; do not disable
TLS verification.

Credentials may live in the configuration file. Keep that file out of Git,
published images, and logs, and restrict its permissions. Do not require
environment variables when explicit configuration credentials are supplied.

`timeout` is optional, defaults to five minutes, and bounds remote operations,
including the lifetime of an `Open` stream. Zero selects the default; negative
values are invalid. Propagate caller contexts and deadlines.

Remote buckets must already exist. Construction does not probe or create them,
change ACLs, or make objects public.

## Bucket roots and relative keys

The `bucket` field accepts `bucket/prefix`, including nested prefixes. Do not
introduce a separate `prefix` configuration field or pass the whole value as
the actual S3 bucket name:

```text
configuration: bucket: "archivos/fotos"
key:           /febrero/kevin.jpg
actual bucket: archivos
actual key:    fotos/febrero/kevin.jpg
```

With local `root: "./data/archivos/fotos"`, the same key writes to
`./data/archivos/fotos/febrero/kevin.jpg`. Leading slashes are ignored, never
treated as absolute paths. Services must not prepend the configured root to
their keys. `Stat` and `List` return relative keys without the backend prefix.

Empty keys, dot segments (`.` and `..`), internal empty segments, trailing
slashes, backslashes, colons, NUL, invalid UTF-8, and keys longer than 1024 bytes
are rejected. Keys are literal strings, not URL-encoded paths.

`.storage-tmp` is reserved and hidden from listings. Local uploads stage there
before replacing the destination; do not expose or use it as application
storage. Local filesystem access uses `os.Root` to prevent symlink escapes;
listings exclude symlinks and special files.

## Construction and Fx

With `setup.Service`, explicitly enable storage:

```go
service := setup.NewService(setup.WithStorage())
service.Run(fx.Provide(NewDocumentService))

type DocumentService struct {
	files storage.Storage
}

func NewDocumentService(files storage.Storage) *DocumentService {
	return &DocumentService{files: files}
}
```

`WithStorage` provides the configuration and `storage.NewStorage` through Fx.
Construction is lazy: a consumer must request `storage.Storage`. Do not register
`NewStorage` a second time when using `WithStorage`. Existing applications
without this option do not require storage configuration.

For standalone use:

```go
files, err := storage.NewStorage(storage.Config{
	Driver: storage.DriverLocal,
	Local:  storage.LocalConfig{Root: "./data/archivos/fotos"},
})
if err != nil {
	return err
}
```

The other driver constants are `storage.DriverS3` and `storage.DriverMinIO`;
their configuration types are `storage.S3Config` and `storage.MinIOConfig`.
Without `setup.Service`, an application can provide its own `storage.Config`
and `storage.NewStorage` directly to Fx.

## Operations

```go
type Storage interface {
	Put(ctx context.Context, key string, body io.Reader, opts PutOptions) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Stat(ctx context.Context, key string) (ObjectInfo, error)
	List(ctx context.Context, opts ListOptions) (ListResult, error)
	Copy(ctx context.Context, sourceKey, destinationKey string) error
}
```

- `Put` saves or replaces an object after a successful upload. Pass an
  `io.Reader` and `storage.PutOptions`; do not buffer entire uploads in memory.
  Failed source streams do not truncate the old object. Enforce application
  upload-size limits at the boundary.
- `Open` returns a stream the caller must close, even after read errors:

  ```go
  reader, err := files.Open(ctx, "/febrero/kevin.jpg")
  if err != nil {
  	return err
  }
  defer reader.Close()
  _, err = io.Copy(destination, reader)
  return err
  ```

- `Delete` succeeds when the object is absent. It does not remove local
  directories or override remote versioning, retention, or permissions.
- `Stat` returns `ObjectInfo` with `Key`, `Size`, and `LastModified`. Remote
  `Stat` also returns `ContentType`. `PutOptions.ContentType` is persisted only
  by S3/MinIO; local storage ignores it. `List` omits content type everywhere.
- `Copy` replaces a destination within the same storage without removing the
  source. Do not present copy-then-delete as an atomic move. Remote copies up to
  5 GiB are server-side; larger copies stream through multipart upload and
  preserve content type, not arbitrary provider-specific metadata.

### Listing and pagination

`ListOptions.Prefix` is a key prefix, not a directory: `feb` matches both
`feb.txt` and `febrero/kevin.jpg`. Empty lists all keys; a trailing slash is
allowed. Results are recursive and lexicographically ordered.

```go
var token string
for {
	page, err := files.List(ctx, storage.ListOptions{
		Prefix:            "febrero/",
		Limit:             100,
		ContinuationToken: token,
	})
	if err != nil {
		return err
	}
	for _, object := range page.Objects {
		// Process object.Key relative to the configured root.
		_ = object
	}
	token = page.ContinuationToken
	if token == "" {
		break
	}
}
```

Limit defaults to 100 and must be 1–1000 when specified. Reuse the returned
token only with the same storage and prefix. Continue until the token is empty,
not until the object slice is empty: remote pages may omit externally-created
keys that this API cannot represent. Tokens are opaque cursors, not authorization
or consistent snapshots. Local listing scans the tree with bounded result memory.

## Errors and boundaries

Use `errors.Is` with `storage.ErrNotFound`, `storage.ErrInvalidKey`, and
`storage.ErrInvalidToken`. Preserve other provider errors, including access
denied. S3 HEAD may return a generic 404 that cannot distinguish a missing object
from a missing bucket.

Keep storage operations out of HTTP response formatting. Return errors to the
application service and translate them at the HTTP boundary. The storage package
does not provide public URLs, signed URLs, HTTP serving, or authentication.
