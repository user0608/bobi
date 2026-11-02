# Storage

`storage` exposes the same interface for local files, AWS S3, and MinIO:

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

## Configuration

Only the selected driver's section is required and validated. An omitted or
unknown driver returns an error. Remote buckets must already exist; the
constructor does not probe them or change their permissions.

### Local

```yaml
storage:
  driver: local
  local:
    root: "./data/archivos/fotos"
```

`root` is required and created if necessary. New directories use permissions
`0700` and files `0600`. Existing permissions are not changed. Access through
symlinks cannot escape the root (`os.Root`); listings exclude symlinks and
special files. The configured root and its filesystem should be trusted.

### AWS S3

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

`bucket` and `region` are required. Explicit `access_key` and `secret_key` must be
provided together and take precedence over the AWS credential chain. Omit both
to use the SDK's standard credentials, including profiles, environment
variables, and IAM roles. `session_token` is optional for explicit temporary
credentials and requires both credential fields. Standard general-purpose S3
buckets are supported, not S3 Express directory buckets or access-point ARNs.

### MinIO

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

`endpoint`, `bucket`, `access_key`, and `secret_key` are required. `region`
defaults to `us-east-1`. The endpoint must be an HTTP(S) origin without a path,
query, fragment, or embedded credentials. Use HTTPS in production; TLS
certificate verification is not disabled.

Credentials may be stored in the configuration file. Add that specific file to
`.gitignore`, restrict its permissions, and exclude it from published images and
logs. An already-tracked file is not protected by adding it to `.gitignore`.

`timeout` is optional and defaults to five minutes for remote operations,
including the lifetime of streams returned by `Open`. Caller deadlines can
shorten it. Negative timeouts are invalid; zero selects the default. Local I/O
checks caller cancellation between reads; it
cannot interrupt a caller-provided reader blocked inside `Read`.

## Roots and keys

For S3/MinIO, the first segment of `bucket` is the bucket name; all remaining
segments form a prefix root. No separate `prefix` configuration is needed:

```text
bucket: archivos/fotos
key:    /febrero/kevin.jpg

actual bucket: archivos
actual key:    fotos/febrero/kevin.jpg
```

The same key with local `root: ./data/archivos/fotos` writes to
`./data/archivos/fotos/febrero/kevin.jpg`. Leading slashes are ignored, never
interpreted as absolute paths. All operations respect the configured root;
`Stat` and `List` return relative keys without the configured prefix.

Empty keys, `.`/`..` segments, internal empty segments, trailing slashes,
backslashes, colons, NUL, invalid UTF-8, and keys longer than 1024 bytes are
rejected. `.storage-tmp` is reserved for local upload staging. List prefixes
may be empty or end with `/`. Treat keys as literal strings, not URL-encoded
paths: percent-encoded text is not decoded by this package.

## Usage and Fx

```go
files, err := storage.NewStorage(storage.Config{
    Driver: storage.DriverLocal,
    Local:  storage.LocalConfig{Root: "./data/archivos/fotos"},
})
if err != nil {
    return err
}

if err := files.Put(ctx, "/febrero/kevin.jpg", body, storage.PutOptions{
    ContentType: "image/jpeg",
}); err != nil {
    return err
}

reader, err := files.Open(ctx, "febrero/kevin.jpg")
if err != nil {
    return err
}
defer reader.Close()
// Stream reader to its destination; do not load large files into memory.
```

With `setup.Service`, storage is opt-in:

```go
service := setup.NewService(setup.WithStorage())
service.Run(fx.Provide(NewDocumentService))

func NewDocumentService(files storage.Storage) *DocumentService {
    return &DocumentService{files: files}
}
```

`WithStorage` reads the Viper `storage` section and provides `storage.Storage`
using Fx. Construction is lazy, when a consumer requests the interface. Existing
applications do not need storage configuration. Without `setup.Service`, provide
your own `storage.Config` and `storage.NewStorage` directly to Fx.

## Operation semantics

- `Put` replaces an object after a successful upload; failed source streams do
  not truncate the old object. Local writes stage and rename; remote uploads use
  the SDK's bounded multipart transfer manager. Application-level upload size
  limits are the caller's responsibility.
- Always close the reader returned by `Open`, including after read errors.
- `Delete` is idempotent for missing objects. Local directories are not objects
  and are not removed. S3 bucket versioning, retention, and permissions still
  apply; deletion is not a promise of permanent erasure.
- `Stat` returns key, size, and last-modified time. Remote `Stat` also returns
  the stored content type. Local storage ignores `PutOptions.ContentType` and
  does not persist MIME metadata; `List` omits content type on every backend.
- `List` recursively selects by key prefix, not directory name: `feb` matches
  both `febrero/a.jpg` and `feb.txt`. Its default limit is 100, maximum 1000.
  Pass the returned `ContinuationToken` into the next request with the same
  prefix and backend. An empty token means there are no more pages. Tokens are
  opaque pagination cursors, not authorization tokens or consistent snapshots.
  Local listing scans the tree but retains at most limit+1 entries. Remote
  pages can be shorter when externally-created keys are not representable by
  this API; continue while a token is present.
- `Copy` operates within one storage and replaces the destination. It does not
  delete the source. Remote copies up to 5 GiB are server-side; larger copies
  stream through multipart upload. Large streaming copies preserve content
  type, not arbitrary provider-specific metadata.

Use `errors.Is(err, storage.ErrNotFound)`, `storage.ErrInvalidKey`, and
`storage.ErrInvalidToken` for portable errors. Provider errors otherwise remain
available, including access denied and missing-bucket errors. S3 HEAD responses
can return generic 404s, so not every provider can distinguish a missing object
from a missing bucket. No public URLs, ACL changes, or authentication middleware
are created by this package.

## Tests

```bash
go test -race ./storage ./setup
go test -tags=integration ./storage
```

Integration tests start MinIO using Docker. `BOBI_TEST_MINIO_IMAGE` can override
the pinned image. AWS tests are opt-in with `BOBI_TEST_S3_BUCKET` and
`BOBI_TEST_S3_REGION`, using the standard credential chain. They create and clean
up objects beneath a unique test prefix in that existing bucket.
