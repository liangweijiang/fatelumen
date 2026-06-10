package storage

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"fatelumen/backend/internal/pkg/logger"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Storage struct {
	client     *s3.Client
	bucket     string
	publicBase string
	log        *logger.Logger
}

func NewR2Storage(accountID, accessKeyID, secretAccessKey, bucket, publicBase string, log *logger.Logger) (*R2Storage, error) {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(),
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("r2 config: %w", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = &endpoint
		o.UsePathStyle = false
	})

	return &R2Storage{
		client:     client,
		bucket:     bucket,
		publicBase: strings.TrimRight(publicBase, "/"),
		log:        log,
	}, nil
}

func (r *R2Storage) logError(msg string, args ...any) {
	if r.log != nil {
		r.log.Error(msg, args...)
	}
}

func (r *R2Storage) Put(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &r.bucket,
		Key:         &key,
		Body:        bytes.NewReader(data),
		ContentType: &contentType,
	})
	if err != nil {
		r.logError("r2 upload failed", "err", err, "key", key)
		return "", fmt.Errorf("r2 put: %w", err)
	}
	return fmt.Sprintf("%s/%s", r.publicBase, key), nil
}
