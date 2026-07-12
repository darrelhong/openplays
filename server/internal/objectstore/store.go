// Package objectstore adds OpenPlays policy and configuration around Go CDK's
// portable blob storage API.
package objectstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"gocloud.dev/blob"
	"gocloud.dev/gcerrors"
)

var (
	ErrInvalidKey      = errors.New("invalid object key")
	ErrInvalidBody     = errors.New("invalid object body")
	ErrInvalidMetadata = errors.New("invalid object metadata")
)

type PutOptions struct {
	ContentType  string
	CacheControl string
}

// Store owns a Go CDK bucket and adds application-wide key and URL policy.
type Store struct {
	bucket        *blob.Bucket
	publicBaseURL string
}

// NewWithBucket wraps a Go CDK bucket. The Store takes ownership of bucket;
// callers must close the Store rather than closing the bucket directly.
func NewWithBucket(bucket *blob.Bucket, publicBaseURL string) (*Store, error) {
	if bucket == nil {
		return nil, errors.New("object store bucket is required")
	}
	if err := validatePublicBaseURL(publicBaseURL); err != nil {
		return nil, err
	}
	return &Store{bucket: bucket, publicBaseURL: strings.TrimRight(publicBaseURL, "/")}, nil
}

func (s *Store) Put(ctx context.Context, key string, body io.Reader, options PutOptions) error {
	if err := ValidateKey(key); err != nil {
		return err
	}
	if body == nil {
		return ErrInvalidBody
	}
	if options.ContentType == "" || invalidMetadataValue(options.ContentType) || invalidMetadataValue(options.CacheControl) {
		return ErrInvalidMetadata
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	err := s.bucket.Upload(ctx, key, body, &blob.WriterOptions{
		ContentType:  options.ContentType,
		CacheControl: options.CacheControl,
	})
	if err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	if err := ValidateKey(key); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	err := s.bucket.Delete(ctx, key)
	if err != nil && gcerrors.Code(err) != gcerrors.NotFound {
		return fmt.Errorf("delete object %q: %w", key, err)
	}
	return nil
}

func (s *Store) PublicURL(key string) (string, error) {
	if err := ValidateKey(key); err != nil {
		return "", err
	}
	segments := strings.Split(key, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return s.publicBaseURL + "/" + strings.Join(segments, "/"), nil
}

func (s *Store) Close() error {
	return s.bucket.Close()
}

func ValidateKey(key string) error {
	if key == "" || len(key) > 1024 || !utf8.ValidString(key) ||
		strings.HasPrefix(key, "/") ||
		strings.Contains(key, "\\") ||
		strings.Contains(key, "../") {
		return ErrInvalidKey
	}
	if strings.IndexFunc(key, unicode.IsControl) != -1 {
		return ErrInvalidKey
	}
	for _, part := range strings.Split(key, "/") {
		if part == "" || part == "." || part == ".." {
			return ErrInvalidKey
		}
	}
	return nil
}

func invalidMetadataValue(value string) bool {
	return !utf8.ValidString(value) || strings.IndexFunc(value, unicode.IsControl) != -1
}
