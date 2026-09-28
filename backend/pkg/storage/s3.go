package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
