package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
)

func TestMapErrorMarksMissingObjects(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{name: "no such key", err: minio.ErrorResponse{Code: "NoSuchKey"}},
		{name: "not found status", err: minio.ErrorResponse{Code: "NotFound", StatusCode: 404}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := mapError(tc.err); !errors.Is(err, ErrNotFound) {
				t.Fatalf("mapError() = %v, want ErrNotFound", err)
			}
		})
	}

	other := errors.New("connection refused")
	if err := mapError(other); errors.Is(err, ErrNotFound) {
		t.Fatalf("mapError() = %v, do not want ErrNotFound", err)
	}
}

// TestS3RoundTrip talks to a real Garage instance. It is skipped unless the
// MOADA_TEST_S3_* variables are provided.
func TestS3RoundTrip(t *testing.T) {
	endpoint := os.Getenv("MOADA_TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("set MOADA_TEST_S3_ENDPOINT to run the S3 integration test")
	}

	store, err := NewS3(S3Options{
		Endpoint:  endpoint,
		AccessKey: os.Getenv("MOADA_TEST_S3_ACCESS_KEY"),
		SecretKey: os.Getenv("MOADA_TEST_S3_SECRET_KEY"),
		Bucket:    os.Getenv("MOADA_TEST_S3_BUCKET"),
		Region:    os.Getenv("MOADA_TEST_S3_REGION"),
		UseSSL:    os.Getenv("MOADA_TEST_S3_USE_SSL") == "true",
	})
	if err != nil {
		t.Fatalf("NewS3() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := store.EnsureBucket(ctx); err != nil {
		t.Fatalf("EnsureBucket() error = %v", err)
	}

	key := fmt.Sprintf("tests/%d", time.Now().UnixNano())
	content := []byte("moada integration test")

	if err := store.Put(ctx, key, bytes.NewReader(content), int64(len(content)), "text/plain"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if err := store.Remove(cleanupCtx, key); err != nil {
			t.Errorf("Remove() cleanup error = %v", err)
		}
	})

	info, err := store.Stat(ctx, key)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Size != int64(len(content)) {
		t.Errorf("Stat().Size = %d, want %d", info.Size, len(content))
	}

	reader, err := store.Open(ctx, key)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer reader.Close()

	read, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !bytes.Equal(read, content) {
		t.Errorf("downloaded content = %q, want %q", read, content)
	}

	if _, err := store.Stat(ctx, "tests/does-not-exist"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Stat() missing object error = %v, want ErrNotFound", err)
	}
}
