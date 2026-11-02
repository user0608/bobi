package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

type s3Storage struct {
	client   *s3.Client
	uploader *transfermanager.Client
	bucket   string
	prefix   string
	identity string
	timeout  time.Duration
}

var _ Storage = (*s3Storage)(nil)

func newS3Storage(config S3Config, timeout time.Duration) (Storage, error) {
	bucket, prefix, err := splitBucket(config.Bucket)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(config.Region) == "" {
		return nil, fmt.Errorf("s3 region is required")
	}
	if (config.AccessKey == "") != (config.SecretKey == "") {
		return nil, fmt.Errorf("s3 access_key and secret_key must be provided together")
	}
	if config.SessionToken != "" && config.AccessKey == "" {
		return nil, fmt.Errorf("s3 session_token requires access_key and secret_key")
	}
	opts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(config.Region)}
	if config.AccessKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.AccessKey, config.SecretKey, config.SessionToken,
		)))
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws storage configuration: %w", err)
	}
	return makeS3Storage(s3.NewFromConfig(cfg), bucket, prefix,
		"s3:"+config.Region+":"+bucket+"/"+prefix, timeout), nil
}

func newMinIOStorage(config MinIOConfig, timeout time.Duration) (Storage, error) {
	bucket, prefix, err := splitBucket(config.Bucket)
	if err != nil {
		return nil, err
	}
	endpoint, err := url.Parse(config.Endpoint)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") ||
		endpoint.User != nil || (endpoint.Path != "" && endpoint.Path != "/") ||
		endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, fmt.Errorf("minio endpoint must be an http(s) origin without credentials, path, query, or fragment")
	}
	if config.AccessKey == "" || config.SecretKey == "" {
		return nil, fmt.Errorf("minio access_key and secret_key are required")
	}
	if config.Region == "" {
		config.Region = "us-east-1"
	}
	cfg := aws.Config{
		Region: config.Region,
		Credentials: credentials.NewStaticCredentialsProvider(
			config.AccessKey, config.SecretKey, "",
		),
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
	}
	client := s3.NewFromConfig(cfg, func(opts *s3.Options) {
		opts.BaseEndpoint = aws.String(strings.TrimSuffix(config.Endpoint, "/"))
		opts.UsePathStyle = true
	})
	return makeS3Storage(client, bucket, prefix,
		"minio:"+config.Endpoint+":"+bucket+"/"+prefix, timeout), nil
}

func makeS3Storage(client *s3.Client, bucket, prefix, identity string, timeout time.Duration) *s3Storage {
	return &s3Storage{
		client: client,
		uploader: transfermanager.New(client, func(opts *transfermanager.Options) {
			opts.Concurrency = 2
			opts.PartSizeBytes = 8 * 1024 * 1024
			opts.MultipartUploadThreshold = opts.PartSizeBytes
			opts.FailTimeout = 10 * time.Second
			opts.RequestChecksumCalculation = client.Options().RequestChecksumCalculation
		}),
		bucket: bucket, prefix: prefix, identity: identity, timeout: timeout,
	}
}

func (s *s3Storage) Put(ctx context.Context, key string, body io.Reader, opts PutOptions) error {
	key, err := normalizeKey(key, false)
	if err != nil {
		return err
	}
	if body == nil {
		return fmt.Errorf("storage body is required")
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	input := &transfermanager.UploadObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(s.prefix + key),
		Body: contextReader{ctx: ctx, r: body},
	}
	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}
	_, err = s.uploader.UploadObject(ctx, input)
	return remoteError(err)
}

func (s *s3Storage) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	key, err := normalizeKey(key, false)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(s.prefix + key),
	})
	if err != nil {
		cancel()
		return nil, remoteError(err)
	}
	return &contextReadCloser{
		contextReader: contextReader{ctx: ctx, r: output.Body},
		closer:        output.Body, cancel: cancel,
	}, nil
}

func (s *s3Storage) Delete(ctx context.Context, key string) error {
	key, err := normalizeKey(key, false)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	_, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(s.prefix + key),
	})
	err = remoteError(err)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	return err
}

func (s *s3Storage) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	key, err := normalizeKey(key, false)
	if err != nil {
		return ObjectInfo{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	output, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(s.prefix + key),
	})
	if err != nil {
		return ObjectInfo{}, remoteError(err)
	}
	return ObjectInfo{
		Key: key, Size: aws.ToInt64(output.ContentLength),
		LastModified: aws.ToTime(output.LastModified), ContentType: aws.ToString(output.ContentType),
	}, nil
}

func (s *s3Storage) List(ctx context.Context, opts ListOptions) (ListResult, error) {
	prefix, after, scope, limit, err := prepareList(opts, s.identity)
	if err != nil {
		return ListResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket), Prefix: aws.String(s.prefix + prefix),
		MaxKeys: aws.Int32(int32(limit)),
	}
	if after != "" {
		input.StartAfter = aws.String(s.prefix + after)
	}
	output, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return ListResult{}, remoteError(err)
	}
	result := ListResult{Objects: []ObjectInfo{}}
	var lastKey string
	for _, object := range output.Contents {
		fullKey := aws.ToString(object.Key)
		if !strings.HasPrefix(fullKey, s.prefix+prefix) {
			continue
		}
		key := strings.TrimPrefix(fullKey, s.prefix)
		lastKey = key
		// Objects created outside this API may include directory markers or
		// keys that cannot be represented portably. Do not expose them.
		if _, err := normalizeKey(key, false); err != nil {
			continue
		}
		result.Objects = append(result.Objects, ObjectInfo{
			Key: key, Size: aws.ToInt64(object.Size), LastModified: aws.ToTime(object.LastModified),
		})
	}
	if aws.ToBool(output.IsTruncated) && lastKey != "" {
		result.ContinuationToken = nextToken(scope, lastKey)
	}
	return result, nil
}

func (s *s3Storage) Copy(ctx context.Context, sourceKey, destinationKey string) error {
	sourceKey, err := normalizeKey(sourceKey, false)
	if err != nil {
		return err
	}
	destinationKey, err = normalizeKey(destinationKey, false)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	info, err := s.Stat(ctx, sourceKey)
	if err != nil {
		return err
	}
	if sourceKey == destinationKey {
		return nil
	}
	// CopyObject supports at most 5 GiB. Larger objects are streamed through
	// the multipart uploader without buffering the whole object in memory.
	if info.Size > 5*1024*1024*1024 {
		body, err := s.Open(ctx, sourceKey)
		if err != nil {
			return err
		}
		defer body.Close()
		return s.Put(ctx, destinationKey, body, PutOptions{ContentType: info.ContentType})
	}
	_, err = s.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(s.prefix + destinationKey),
		CopySource: aws.String(url.PathEscape(s.bucket + "/" + s.prefix + sourceKey)),
	})
	return remoteError(err)
}

func remoteError(err error) error {
	if err == nil {
		return nil
	}
	if apiError, ok := errors.AsType[smithy.APIError](err); ok {
		switch apiError.ErrorCode() {
		case "NoSuchKey", "NotFound":
			return fmt.Errorf("%w: %w", ErrNotFound, err)
		}
	}
	return err
}
