package storage

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"unicode/utf8"
)

const internalDirectory = ".storage-tmp"

func normalizeKey(key string, prefix bool) (string, error) {
	key = strings.TrimLeft(key, "/")
	if len(key) > 1024 || strings.ContainsAny(key, "\\\x00") || !utf8.ValidString(key) {
		return "", ErrInvalidKey
	}
	if key == "" {
		if prefix {
			return "", nil
		}
		return "", ErrInvalidKey
	}
	parts := strings.Split(key, "/")
	for i, part := range parts {
		if part == "" && prefix && i == len(parts)-1 {
			continue
		}
		if part == "" || part == "." || part == ".." || part == internalDirectory {
			return "", ErrInvalidKey
		}
		// Reject Windows volume/device syntax for consistent portable keys.
		if strings.Contains(part, ":") {
			return "", ErrInvalidKey
		}
	}
	return key, nil
}

func splitBucket(value string) (string, string, error) {
	if value == "" || strings.HasPrefix(value, "/") {
		return "", "", fmt.Errorf("storage bucket is required and must not start with /")
	}
	bucket, prefix, _ := strings.Cut(value, "/")
	if !validBucket(bucket) {
		return "", "", fmt.Errorf("invalid storage bucket")
	}
	if strings.HasPrefix(prefix, "/") {
		return "", "", ErrInvalidKey
	}
	if _, err := normalizeKey(prefix, true); err != nil {
		return "", "", fmt.Errorf("invalid bucket prefix: %w", err)
	}
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix != "" {
		prefix += "/"
	}
	return bucket, prefix, nil
}

func validBucket(bucket string) bool {
	if len(bucket) < 3 || len(bucket) > 63 || net.ParseIP(bucket) != nil {
		return false
	}
	for _, label := range strings.Split(bucket, ".") {
		if label == "" || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, ch := range label {
			if (ch < 'a' || ch > 'z') && (ch < '0' || ch > '9') && ch != '-' {
				return false
			}
		}
	}
	return true
}

type listToken struct {
	Scope string `json:"scope"`
	After string `json:"after"`
}

func listScope(identity, prefix string) string {
	sum := sha256.Sum256([]byte(identity + "\x00" + prefix))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func prepareList(opts ListOptions, identity string) (prefix, after, scope string, limit int, err error) {
	prefix, err = normalizeKey(opts.Prefix, true)
	if err != nil {
		return
	}
	limit = opts.Limit
	if limit == 0 {
		limit = 100
	}
	if limit < 1 || limit > 1000 {
		err = fmt.Errorf("storage list limit must be between 1 and 1000")
		return
	}
	scope = listScope(identity, prefix)
	if opts.ContinuationToken == "" {
		return
	}
	if len(opts.ContinuationToken) > 8192 {
		err = ErrInvalidToken
		return
	}
	var token listToken
	data, decodeErr := base64.RawURLEncoding.DecodeString(opts.ContinuationToken)
	if decodeErr != nil || json.Unmarshal(data, &token) != nil || token.Scope != scope || token.After == "" {
		err = ErrInvalidToken
		return
	}
	after = token.After
	return
}

func nextToken(scope, after string) string {
	data, _ := json.Marshal(listToken{Scope: scope, After: after})
	return base64.RawURLEncoding.EncodeToString(data)
}

// contextReader checks cancellation between reads; it cannot interrupt a
// caller-provided reader blocked inside Read.
type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

type contextReadCloser struct {
	contextReader
	closer io.Closer
	cancel context.CancelFunc
}

func (r *contextReadCloser) Close() error {
	if r.cancel != nil {
		r.cancel()
	}
	return r.closer.Close()
}
