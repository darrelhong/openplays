package objectstore

import (
	"context"
	"fmt"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gocloud.dev/blob/s3blob"
)

func New(ctx context.Context, cfg Config) (*Store, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	awsCfg := aws.Config{
		Region:      "auto",
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	}
	client := s3.NewFromConfig(awsCfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(cfg.Endpoint)
		options.UsePathStyle = usesPathStyleEndpoint(cfg.Endpoint)
	})
	bucket, err := s3blob.OpenBucket(ctx, client, cfg.Bucket, nil)
	if err != nil {
		return nil, fmt.Errorf("open R2 bucket: %w", err)
	}

	store, err := NewWithBucket(bucket, cfg.PublicBaseURL)
	if err != nil {
		_ = bucket.Close()
		return nil, err
	}
	return store, nil
}

// Local S3-compatible stores such as MinIO do not normally have wildcard DNS
// configured, so address buckets as /bucket/key instead of bucket.host/key.
func usesPathStyleEndpoint(endpoint string) bool {
	parsed, err := url.Parse(endpoint)
	return err == nil && isLoopbackHost(parsed.Hostname())
}
