package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"subject-choice-forum/backend/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

var ErrObjectNotFound = errors.New("object not found")

type ObjectInfo struct {
	Size        int64
	ContentType string
	ETag        string
}

// ObjectStore is deliberately small: database records contain only the key and
// validated metadata; storage providers own bytes and temporary upload URLs.
type ObjectStore interface {
	Check(ctx context.Context) error
	PresignPut(ctx context.Context, key, contentType string, expires time.Duration) (string, error)
	Put(ctx context.Context, key, contentType string, body io.Reader, size int64) error
	Delete(ctx context.Context, key string) error
	Head(ctx context.Context, key string) (ObjectInfo, error)
	Open(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error)
	PublicURL(key string) string
}

func NewObjectStore(ctx context.Context, cfg config.Config) (ObjectStore, error) {
	if cfg.StorageDriver == "s3" {
		return NewS3ObjectStore(ctx, cfg)
	}
	if cfg.StorageDriver != "local" && cfg.StorageDriver != "" {
		return nil, errors.New("unsupported storage driver")
	}
	return NewLocalObjectStore(cfg.MediaUploadDir, cfg.AppBasePath)
}

type LocalObjectStore struct{ root, publicBase string }

func NewLocalObjectStore(root, publicBase string) (*LocalObjectStore, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("local object store root is required")
	}
	if err := os.MkdirAll(root, 0750); err != nil {
		return nil, err
	}
	return &LocalObjectStore{root: filepath.Clean(root), publicBase: strings.TrimRight(publicBase, "/")}, nil
}
func (s *LocalObjectStore) Check(_ context.Context) error {
	info, err := os.Stat(s.root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("local object store root is not a directory")
	}
	return nil
}
func (s *LocalObjectStore) path(key string) (string, error) {
	clean := filepath.Clean("/" + strings.TrimSpace(key))
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" || clean == "." || strings.HasPrefix(clean, "..") {
		return "", errors.New("invalid object key")
	}
	return filepath.Join(s.root, clean), nil
}
func (s *LocalObjectStore) PresignPut(_ context.Context, key, _ string, _ time.Duration) (string, error) {
	return "/api/v1/uploads/images/" + key + "/object", nil
}
func (s *LocalObjectStore) Put(_ context.Context, key, contentType string, body io.Reader, size int64) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0750); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if n, err := io.Copy(f, io.LimitReader(body, size+1)); err != nil {
		return err
	} else if n != size {
		return fmt.Errorf("object size mismatch: got %d want %d", n, size)
	}
	_ = contentType
	return nil
}
func (s *LocalObjectStore) Delete(_ context.Context, key string) error {
	p, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil {
		if os.IsNotExist(err) {
			return ErrObjectNotFound
		}
		return err
	}
	return nil
}
func (s *LocalObjectStore) Head(_ context.Context, key string) (ObjectInfo, error) {
	p, err := s.path(key)
	if err != nil {
		return ObjectInfo{}, err
	}
	i, err := os.Stat(p)
	if os.IsNotExist(err) {
		return ObjectInfo{}, ErrObjectNotFound
	}
	if err != nil {
		return ObjectInfo{}, err
	}
	return ObjectInfo{Size: i.Size()}, err
}
func (s *LocalObjectStore) Open(_ context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	p, err := s.path(key)
	if err != nil {
		return nil, ObjectInfo{}, err
	}
	f, err := os.Open(p)
	if os.IsNotExist(err) {
		return nil, ObjectInfo{}, ErrObjectNotFound
	}
	if err != nil {
		return nil, ObjectInfo{}, err
	}
	i, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, ObjectInfo{}, err
	}
	return f, ObjectInfo{Size: i.Size()}, nil
}
func (s *LocalObjectStore) PublicURL(key string) string {
	return s.publicBase + "/uploads/" + strings.TrimLeft(key, "/")
}

type S3ObjectStore struct {
	client      *s3.Client
	presigner   *s3.PresignClient
	bucket, cdn string
}

func NewS3ObjectStore(ctx context.Context, cfg config.Config) (*S3ObjectStore, error) {
	if cfg.S3Bucket == "" || cfg.S3Region == "" || cfg.S3Endpoint == "" {
		return nil, errors.New("S3_BUCKET, S3_REGION and S3_ENDPOINT are required")
	}
	loadOpts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.S3Region),
		// Aliyun OSS does not accept AWS SDK's streaming checksum trailer.
		awsconfig.WithRequestChecksumCalculation(aws.RequestChecksumCalculationWhenRequired),
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.S3Endpoint)
		o.UsePathStyle = cfg.S3ForcePathStyle
	})
	return &S3ObjectStore{client: client, presigner: s3.NewPresignClient(client), bucket: cfg.S3Bucket, cdn: strings.TrimRight(cfg.S3CDNBaseURL, "/")}, nil
}
func (s *S3ObjectStore) Check(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &s.bucket})
	if err != nil {
		return fmt.Errorf("head S3 bucket: %w", err)
	}
	return nil
}
func (s *S3ObjectStore) PresignPut(ctx context.Context, key, contentType string, expires time.Duration) (string, error) {
	r, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{Bucket: &s.bucket, Key: &key, ContentType: &contentType}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", err
	}
	return r.URL, nil
}
func (s *S3ObjectStore) Put(ctx context.Context, key, contentType string, body io.Reader, size int64) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{Bucket: &s.bucket, Key: &key, ContentType: &contentType, Body: body, ContentLength: aws.Int64(size)})
	return err
}
func (s *S3ObjectStore) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		return fmt.Errorf("delete S3 object: %w", err)
	}
	return nil
}
func (s *S3ObjectStore) Head(ctx context.Context, key string) (ObjectInfo, error) {
	r, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		return ObjectInfo{}, classifyS3ObjectError(err)
	}
	return ObjectInfo{Size: aws.ToInt64(r.ContentLength), ContentType: aws.ToString(r.ContentType), ETag: aws.ToString(r.ETag)}, nil
}
func (s *S3ObjectStore) Open(ctx context.Context, key string) (io.ReadCloser, ObjectInfo, error) {
	r, err := s.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &s.bucket, Key: &key})
	if err != nil {
		return nil, ObjectInfo{}, classifyS3ObjectError(err)
	}
	return r.Body, ObjectInfo{Size: aws.ToInt64(r.ContentLength), ContentType: aws.ToString(r.ContentType), ETag: aws.ToString(r.ETag)}, nil
}

func classifyS3ObjectError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoSuchKey", "NoSuchObject", "NotFound":
			return errors.Join(ErrObjectNotFound, err)
		}
	}
	var responseErr *smithyhttp.ResponseError
	if errors.As(err, &responseErr) && responseErr.HTTPStatusCode() == http.StatusNotFound {
		return errors.Join(ErrObjectNotFound, err)
	}
	return err
}
func (s *S3ObjectStore) PublicURL(key string) string {
	if s.cdn == "" {
		return ""
	}
	parts := strings.Split(strings.TrimLeft(key, "/"), "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return s.cdn + "/" + strings.Join(parts, "/")
}
