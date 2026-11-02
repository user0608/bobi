package storage

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
)

type wireObject struct {
	data        []byte
	contentType string
}

// This test server exercises actual SDK serialization and signing, not a mocked
// Go client. Real MinIO and AWS coverage lives in s3_integration_test.go.
type s3WireServer struct {
	mu      sync.Mutex
	objects map[string]wireObject
	auth    string
	copied  string
	parts   map[int][]byte
	aborted bool
}

func (s *s3WireServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.auth = r.Header.Get("Authorization")
	key := strings.TrimPrefix(r.URL.Path, "/archivos/")
	w.Header().Set("Content-Type", "application/xml")
	if r.URL.Query().Has("uploads") {
		s.parts = make(map[int][]byte)
		fmt.Fprint(w, `<InitiateMultipartUploadResult><UploadId>test-upload</UploadId></InitiateMultipartUploadResult>`)
		return
	}
	if r.URL.Query().Get("uploadId") != "" {
		switch r.Method {
		case http.MethodPut:
			part, _ := strconv.Atoi(r.URL.Query().Get("partNumber"))
			data, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			s.parts[part] = data
			w.Header().Set("ETag", fmt.Sprintf(`"part-%d"`, part))
		case http.MethodPost:
			var data []byte
			for part := 1; part <= len(s.parts); part++ {
				data = append(data, s.parts[part]...)
			}
			s.objects[key] = wireObject{data: data}
			fmt.Fprint(w, `<CompleteMultipartUploadResult><ETag>"complete"</ETag></CompleteMultipartUploadResult>`)
			s.parts = nil
		case http.MethodDelete:
			s.aborted = true
			s.parts = nil
			w.WriteHeader(http.StatusNoContent)
		}
		return
	}
	if r.URL.Query().Get("list-type") == "2" {
		prefix, after := r.URL.Query().Get("prefix"), r.URL.Query().Get("start-after")
		limit, _ := strconv.Atoi(r.URL.Query().Get("max-keys"))
		var keys []string
		for key := range s.objects {
			if strings.HasPrefix(key, prefix) && key > after {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		truncated := len(keys) > limit
		if truncated {
			keys = keys[:limit]
		}
		output := struct {
			XMLName     xml.Name `xml:"ListBucketResult"`
			IsTruncated bool
			Contents    []struct {
				Key          string
				Size         int
				LastModified string
			}
		}{IsTruncated: truncated}
		for _, key := range keys {
			output.Contents = append(output.Contents, struct {
				Key          string
				Size         int
				LastModified string
			}{key, len(s.objects[key].data), "2026-01-01T00:00:00Z"})
		}
		xml.NewEncoder(w).Encode(output)
		return
	}
	switch r.Method {
	case http.MethodPut:
		if source := r.Header.Get("X-Amz-Copy-Source"); source != "" {
			s.copied, _ = url.PathUnescape(source)
			object, ok := s.objects[strings.TrimPrefix(s.copied, "archivos/")]
			if !ok {
				wireNotFound(w)
				return
			}
			s.objects[key] = object
			fmt.Fprint(w, `<CopyObjectResult><ETag>"copied"</ETag><LastModified>2026-01-01T00:00:00Z</LastModified></CopyObjectResult>`)
			return
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		s.objects[key] = wireObject{data: data, contentType: r.Header.Get("Content-Type")}
		w.Header().Set("ETag", `"uploaded"`)
	case http.MethodGet, http.MethodHead:
		object, ok := s.objects[key]
		if !ok {
			wireNotFound(w)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(object.data)))
		w.Header().Set("Content-Type", object.contentType)
		w.Header().Set("Last-Modified", "Thu, 01 Jan 2026 00:00:00 GMT")
		if r.Method == http.MethodGet {
			w.Write(object.data)
		}
	case http.MethodDelete:
		delete(s.objects, key)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func wireNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, `<Error><Code>NoSuchKey</Code><Message>missing object</Message></Error>`)
}

func TestRemoteStorageContract(t *testing.T) {
	for _, driver := range []Driver{DriverS3, DriverMinIO} {
		t.Run(string(driver), func(t *testing.T) {
			wire := &s3WireServer{objects: map[string]wireObject{
				"fotos-other/hidden": {data: []byte("outside prefix")},
			}}
			server := httptest.NewServer(wire)
			t.Cleanup(server.Close)
			var files Storage
			var err error
			if driver == DriverMinIO {
				files, err = NewStorage(Config{Driver: driver, MinIO: MinIOConfig{
					Endpoint: server.URL, Bucket: "archivos/fotos", AccessKey: "configured-key", SecretKey: "configured-secret",
				}})
			} else {
				files, err = NewStorage(Config{Driver: driver, S3: S3Config{
					Region: "us-east-1", Bucket: "archivos/fotos", AccessKey: "configured-key", SecretKey: "configured-secret",
				}})
				if err == nil {
					// Keep production AWS endpoint policy intact; substitute only
					// the transport destination for this wire-level test.
					backend := files.(*s3Storage)
					client := s3.NewFromConfig(aws.Config{
						Region: "us-east-1", Credentials: backend.client.Options().Credentials,
					}, func(opts *s3.Options) {
						opts.BaseEndpoint = aws.String(server.URL)
						opts.UsePathStyle = true
					})
					files = makeS3Storage(client, backend.bucket, backend.prefix, backend.identity, time.Minute)
				}
			}
			require.NoError(t, err)
			storageContract(t, files)
			info, err := files.Stat(t.Context(), "febrero/copy.jpg")
			require.NoError(t, err)
			require.Equal(t, "image/jpeg", info.ContentType)
			wire.mu.Lock()
			defer wire.mu.Unlock()
			require.Contains(t, wire.auth, "Credential=configured-key/")
			require.Equal(t, "archivos/fotos/febrero/kevin.jpg", wire.copied)
			require.Contains(t, wire.objects, "fotos/febrero/copy.jpg")
			require.NotContains(t, wire.objects, "/fotos/febrero/copy.jpg")
		})
	}
}

func TestRemoteCopyEscapesSource(t *testing.T) {
	t.Parallel()
	wire := &s3WireServer{objects: map[string]wireObject{}}
	server := httptest.NewServer(wire)
	t.Cleanup(server.Close)
	files, err := NewStorage(Config{Driver: DriverMinIO, MinIO: MinIOConfig{
		Endpoint: server.URL, Bucket: "archivos/fotos", AccessKey: "key", SecretKey: "secret",
	}})
	require.NoError(t, err)
	key := "/febrero/kevin + #%.jpg"
	require.NoError(t, files.Put(t.Context(), key, strings.NewReader("photo"), PutOptions{}))
	require.NoError(t, files.Copy(t.Context(), key, "copy"))
	wire.mu.Lock()
	defer wire.mu.Unlock()
	require.Equal(t, "archivos/fotos/febrero/kevin + #%.jpg", wire.copied)
}

func TestRemoteMultipartUpload(t *testing.T) {
	wire := &s3WireServer{objects: map[string]wireObject{}}
	server := httptest.NewServer(wire)
	t.Cleanup(server.Close)
	files, err := NewStorage(Config{Driver: DriverMinIO, MinIO: MinIOConfig{
		Endpoint: server.URL, Bucket: "archivos/fotos", AccessKey: "key", SecretKey: "secret",
	}})
	require.NoError(t, err)
	data := bytes.Repeat([]byte("photo"), 4*1024*1024)
	require.NoError(t, files.Put(t.Context(), "/large/photo.bin", bytes.NewBuffer(data), PutOptions{}))
	body, err := files.Open(t.Context(), "large/photo.bin")
	require.NoError(t, err)
	got, readErr := io.ReadAll(body)
	require.NoError(t, body.Close())
	require.NoError(t, readErr)
	require.Equal(t, data, got)
	err = files.Put(t.Context(), "large/photo.bin", io.MultiReader(
		bytes.NewReader(data[:10*1024*1024]), failingReader{},
	), PutOptions{})
	require.ErrorIs(t, err, errSource)
	wire.mu.Lock()
	defer wire.mu.Unlock()
	require.True(t, wire.aborted)
	require.Nil(t, wire.parts)
	require.Equal(t, data, wire.objects["fotos/large/photo.bin"].data)
}
