//go:build integration

package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestMinIOIntegration(t *testing.T) {
	testcontainers.SkipIfProviderIsNotHealthy(t)
	ctx := t.Context()
	image := os.Getenv("BOBI_TEST_MINIO_IMAGE")
	if image == "" {
		image = "minio/minio:RELEASE.2025-04-22T22-12-26Z"
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        image,
			ExposedPorts: []string{"9000/tcp"},
			Env: map[string]string{
				"MINIO_ROOT_USER": "bobi-test", "MINIO_ROOT_PASSWORD": "bobi-test-secret",
			},
			Cmd:        []string{"server", "/data"},
			WaitingFor: wait.ForHTTP("/minio/health/ready").WithPort("9000/tcp").WithStartupTimeout(time.Minute),
		},
		Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		require.NoError(t, container.Terminate(cleanup))
	})
	host, err := container.Host(ctx)
	require.NoError(t, err)
	port, err := container.MappedPort(ctx, "9000/tcp")
	require.NoError(t, err)
	files, err := NewStorage(Config{Driver: DriverMinIO, MinIO: MinIOConfig{
		Endpoint: fmt.Sprintf("http://%s:%s", host, port.Port()),
		Bucket:   "archivos/fotos", AccessKey: "bobi-test", SecretKey: "bobi-test-secret",
	}})
	require.NoError(t, err)
	backend := files.(*s3Storage)
	_, err = backend.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String("archivos")})
	require.NoError(t, err)
	storageContract(t, files)

	// Force multipart upload using an unseekable source larger than two parts.
	data := bytes.Repeat([]byte("photo"), 4*1024*1024)
	require.NoError(t, files.Put(ctx, "/large/photo.bin", bytes.NewBuffer(data), PutOptions{}))
	body, err := files.Open(ctx, "large/photo.bin")
	require.NoError(t, err)
	got, readErr := io.ReadAll(body)
	require.NoError(t, body.Close())
	require.NoError(t, readErr)
	require.Equal(t, data, got)
	_, err = files.Stat(ctx, "large/photo.bin")
	require.NoError(t, err)

	// A source failure after at least one part must leave the old object intact
	// and abort the incomplete multipart upload.
	err = files.Put(ctx, "large/photo.bin", io.MultiReader(
		strings.NewReader(strings.Repeat("x", 10*1024*1024)),
		failingReader{},
	), PutOptions{})
	require.ErrorIs(t, err, errSource)
	info, err := files.Stat(ctx, "large/photo.bin")
	require.NoError(t, err)
	require.EqualValues(t, len(data), info.Size)
	uploads, err := backend.client.ListMultipartUploads(ctx, &s3.ListMultipartUploadsInput{
		Bucket: aws.String("archivos"), Prefix: aws.String("fotos/"),
	})
	require.NoError(t, err)
	require.Empty(t, uploads.Uploads)
}

// Set BOBI_TEST_S3_BUCKET to an existing bucket (optionally with a base prefix)
// and BOBI_TEST_S3_REGION. Credentials come from the standard AWS SDK chain.
// A unique prefix is used; cleanup touches only objects created by this test.
func TestAWSIntegration(t *testing.T) {
	bucket := os.Getenv("BOBI_TEST_S3_BUCKET")
	if bucket == "" {
		t.Skip("BOBI_TEST_S3_BUCKET is not configured")
	}
	files, err := NewStorage(Config{Driver: DriverS3, S3: S3Config{
		Bucket: strings.TrimSuffix(bucket, "/") + "/bobi-test-" + uuid.NewString(),
		Region: os.Getenv("BOBI_TEST_S3_REGION"),
	}})
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		var token string
		for {
			page, err := files.List(ctx, ListOptions{ContinuationToken: token})
			require.NoError(t, err)
			for _, object := range page.Objects {
				require.NoError(t, files.Delete(ctx, object.Key))
			}
			token = page.ContinuationToken
			if token == "" {
				break
			}
		}
	})
	storageContract(t, files)
}
