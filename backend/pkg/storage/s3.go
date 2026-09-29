package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

type S3 struct {
	client  *s3.Client
	presign *s3.PresignClient
	bucket  string
}

func NewS3(endpoint, accessKey, secretKey, bucket string) *S3 {
	cfg := aws.Config{
		Region:      "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(strings.TrimRight(endpoint, "/"))
		o.UsePathStyle = true
	})
	return &S3{
		client:  client,
		presign: s3.NewPresignClient(client),
		bucket:  bucket,
	}
}

func Open(ctx context.Context, endpoint, accessKey, secretKey, bucket string) (ObjectStore, error) {
	if accessKey == "" || secretKey == "" {
		return NewMemory(), nil
	}
	s := NewS3(endpoint, accessKey, secretKey, bucket)
	if err := s.EnsureBucket(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *S3) EnsureBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	if err == nil {
		return nil
	}
	if !isMissingBucket(err) {
		return fmt.Errorf("head bucket %s: %w", s.bucket, err)
	}
	_, err = s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(s.bucket)})
	if err == nil || isBucketAlreadyThere(err) {
		return nil
	}
	return fmt.Errorf("create bucket %s: %w", s.bucket, err)
}

func isMissingBucket(err error) bool {
	var nfe *types.NotFound
	if errors.As(err, &nfe) {
		return true
	}
	var nsb *types.NoSuchBucket
	if errors.As(err, &nsb) {
		return true
	}
	var api smithy.APIError
	if errors.As(err, &api) {
		switch api.ErrorCode() {
		case "NotFound", "NoSuchBucket", "404":
			return true
		}
	}
	var re *awshttp.ResponseError
	return errors.As(err, &re) && re.HTTPStatusCode() == 404
}

func isBucketAlreadyThere(err error) bool {
	var owned *types.BucketAlreadyOwnedByYou
	if errors.As(err, &owned) {
		return true
	}
	var exists *types.BucketAlreadyExists
	if errors.As(err, &exists) {
		return true
	}
	var api smithy.APIError
	if !errors.As(err, &api) {
		return false
	}
	switch api.ErrorCode() {
	case "BucketAlreadyOwnedByYou", "BucketAlreadyExists":
		return true
	default:
		return false
	}
}

func (s *S3) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	in := &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		Body:          body,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	}
	_, err := s.client.PutObject(ctx, in)
	return err
}

func (s *S3) PresignGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	out, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("presign: %w", err)
	}
	return out.URL, nil
}
