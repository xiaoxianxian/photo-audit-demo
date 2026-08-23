package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOStorage wraps the MinIO client with convenience methods for the audit platform.
type MinIOStorage struct {
	client *minio.Client
	bucket string
	region string
}

// NewMinIOStorage creates a new MinIOStorage instance.
// endpoint: e.g. "localhost:9000"
// accessKey/secretKey: MinIO credentials
// useSSL: whether to use HTTPS
// bucket: the bucket name (will be created if it doesn't exist)
func NewMinIOStorage(endpoint, accessKey, secretKey, region, bucket string, useSSL bool) (*MinIOStorage, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	// Ensure bucket exists.
	exists, err := client.BucketExists(context.Background(), bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket exists: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(context.Background(), bucket, minio.MakeBucketOptions{Region: region}); err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}
	}

	return &MinIOStorage{
		client: client,
		bucket: bucket,
		region: region,
	}, nil
}

// UploadObject uploads data from a reader to the bucket with the given object name.
// It returns the presigned URL for the uploaded object.
func (s *MinIOStorage) UploadObject(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	n, err := s.client.PutObject(ctx, s.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("put object '%s': %w", objectName, err)
	}

	// Return the full object info (includes size, etag, etc.)
	_ = n

	// Generate a presigned URL valid for 7 days.
	url, err := s.client.PresignedGetObject(ctx, s.bucket, objectName, 7*24*time.Hour, nil)
	if err != nil {
		return "", fmt.Errorf("presign get object: %w", err)
	}

	return url.String(), nil
}

// UploadBytes uploads a byte slice to the bucket.
func (s *MinIOStorage) UploadBytes(ctx context.Context, objectName string, data []byte, contentType string) (string, error) {
	return s.UploadObject(ctx, objectName, bytes.NewReader(data), int64(len(data)), contentType)
}

// PresignedURL returns a presigned download URL for an existing object.
func (s *MinIOStorage) PresignedURL(ctx context.Context, objectName string, expiry time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, s.bucket, objectName, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("presign get object '%s': %w", objectName, err)
	}
	return url.String(), nil
}

// DeleteObject removes an object from the bucket.
func (s *MinIOStorage) DeleteObject(ctx context.Context, objectName string) error {
	return s.client.RemoveObject(ctx, s.bucket, objectName, minio.RemoveObjectOptions{})
}

// GenerateObjectName builds a unique object name from a filename and content type.
func GenerateObjectName(filename string, kind string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".bin"
	}
	return fmt.Sprintf("%s/%s/%s%s", kind, uuid.New().String()[:8], uuid.New().String(), ext)
}
