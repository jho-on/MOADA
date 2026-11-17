// Package storage provides access to encrypted file objects kept in Garage.
//
// The backend only ever stores opaque encrypted bytes. This package does not
// perform encryption or decryption.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// ErrNotFound is returned when an object does not exist.
var ErrNotFound = errors.New("object not found")

// Object describes a stored object.
type Object struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
}

// S3Options configures the S3-compatible client.
type S3Options struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool
}

// Store is the object storage contract used by the application.
type Store interface {
	Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Stat(ctx context.Context, key string) (Object, error)
	Remove(ctx context.Context, key string) error
	Ready(ctx context.Context) error
}

// S3 stores objects in a Garage bucket through the S3 API.
type S3 struct {
	client *minio.Client
	bucket string
}

var _ Store = (*S3)(nil)

// NewS3 creates a Garage S3 client. It does not perform network requests.
func NewS3(options S3Options) (*S3, error) {
	client, err := minio.New(options.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(options.AccessKey, options.SecretKey, ""),
		Secure:       options.UseSSL,
		Region:       options.Region,
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}

	return &S3{client: client, bucket: options.Bucket}, nil
}

// EnsureBucket creates the configured bucket when it does not exist yet.
func (s *S3) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check bucket %q: %w", s.bucket, err)
	}
	if exists {
		return nil
	}

	if err := s.client.MakeBucket(ctx, s.bucket, minio.MakeBucketOptions{}); err != nil {
		// Another instance may have created the bucket concurrently.
		exists, checkErr := s.client.BucketExists(ctx, s.bucket)
		if checkErr == nil && exists {
			return nil
		}
		return fmt.Errorf("create bucket %q: %w", s.bucket, err)
	}

	return nil
}

func (s *S3) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}
	return nil
}

func (s *S3) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	object, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, mapError(err)
	}

	// GetObject is lazy; Stat forces the first request so missing objects and
	// credential problems are reported here instead of while reading.
	if _, err := object.Stat(); err != nil {
		_ = object.Close()
		return nil, mapError(err)
	}

	return object, nil
}

func (s *S3) Stat(ctx context.Context, key string) (Object, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return Object{}, mapError(err)
	}

	return Object{
		Key:          info.Key,
		Size:         info.Size,
		ContentType:  info.ContentType,
		ETag:         info.ETag,
		LastModified: info.LastModified,
	}, nil
}

func (s *S3) Remove(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove object %q: %w", key, err)
	}
	return nil
}

func (s *S3) Ready(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucket)
	if err != nil {
		return fmt.Errorf("check bucket %q: %w", s.bucket, err)
	}
	if !exists {
		return fmt.Errorf("bucket %q does not exist", s.bucket)
	}
	return nil
}

func mapError(err error) error {
	response := minio.ToErrorResponse(err)
	if response.Code == "NoSuchKey" || response.Code == "NoSuchObject" || response.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w: %s", ErrNotFound, err)
	}
	return fmt.Errorf("access object: %w", err)
}
