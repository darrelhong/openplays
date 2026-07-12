package objectstore

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"gocloud.dev/blob"
	"gocloud.dev/blob/memblob"
)

func newTestStore(t *testing.T, baseURL string) (*Store, *blob.Bucket) {
	t.Helper()
	bucket := memblob.OpenBucket(nil)
	store, err := NewWithBucket(bucket, baseURL)
	if err != nil {
		bucket.Close()
		t.Fatalf("NewWithBucket: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store, bucket
}

func TestStoreContract(t *testing.T) {
	store, bucket := newTestStore(t, "https://images.openplays.app/")
	ctx := context.Background()
	key := "avatars/user 1/photo.jpg"
	options := PutOptions{ContentType: "image/jpeg", CacheControl: "public, max-age=31536000, immutable"}

	if err := store.Put(ctx, key, bytes.NewBufferString("image"), options); err != nil {
		t.Fatalf("Put: %v", err)
	}
	data, err := bucket.ReadAll(ctx, key)
	if err != nil || string(data) != "image" {
		t.Fatalf("ReadAll = %q, %v", data, err)
	}
	attrs, err := bucket.Attributes(ctx, key)
	if err != nil {
		t.Fatalf("Attributes: %v", err)
	}
	if attrs.ContentType != options.ContentType || attrs.CacheControl != options.CacheControl {
		t.Fatalf("attributes = content-type %q, cache-control %q", attrs.ContentType, attrs.CacheControl)
	}
	gotURL, err := store.PublicURL(key)
	if err != nil {
		t.Fatalf("PublicURL: %v", err)
	}
	if want := "https://images.openplays.app/avatars/user%201/photo.jpg"; gotURL != want {
		t.Fatalf("PublicURL = %q, want %q", gotURL, want)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("idempotent Delete: %v", err)
	}
}

func TestPutValidatesBoundary(t *testing.T) {
	store, _ := newTestStore(t, "https://images.openplays.app")
	ctx := context.Background()
	if err := store.Put(ctx, "objects/file", nil, PutOptions{ContentType: "application/octet-stream"}); !errors.Is(err, ErrInvalidBody) {
		t.Fatalf("nil body error = %v", err)
	}
	for _, options := range []PutOptions{
		{},
		{ContentType: "image/jpeg\r\nx-injected: true"},
		{ContentType: "image/jpeg", CacheControl: "public\nprivate"},
	} {
		if err := store.Put(ctx, "objects/file", bytes.NewReader(nil), options); !errors.Is(err, ErrInvalidMetadata) {
			t.Errorf("Put(%#v) error = %v, want ErrInvalidMetadata", options, err)
		}
	}
}

func TestValidateKey(t *testing.T) {
	for _, key := range []string{
		"", "/avatar.jpg", "avatars//avatar.jpg", "avatars/../avatar.jpg", `avatars\avatar.jpg`,
		"avatars/new\nline.jpg", "exports/backup../v1.png", strings.Repeat("a", 1025),
	} {
		if err := ValidateKey(key); !errors.Is(err, ErrInvalidKey) {
			t.Errorf("ValidateKey(%q) = %v, want ErrInvalidKey", key, err)
		}
	}
	if err := ValidateKey("avatars/user/photo.jpg"); err != nil {
		t.Fatalf("valid key rejected: %v", err)
	}
}

func TestPublicURLRejectsInvalidKey(t *testing.T) {
	store, _ := newTestStore(t, "https://images.openplays.app")
	for _, key := range []string{"", "../secret.jpg", "avatars/../secret.jpg", "/avatars/photo.jpg"} {
		url, err := store.PublicURL(key)
		if !errors.Is(err, ErrInvalidKey) || url != "" {
			t.Errorf("PublicURL(%q) = %q, %v", key, url, err)
		}
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("OBJECT_STORE_PROVIDER", " R2 ")
	t.Setenv("OBJECT_STORE_BUCKET", " openplays-avatars ")
	t.Setenv("OBJECT_STORE_PUBLIC_BASE_URL", " https://images.openplays.app ")
	t.Setenv("OBJECT_STORE_ENDPOINT", " https://account.r2.cloudflarestorage.com ")
	t.Setenv("OBJECT_STORE_ACCESS_KEY_ID", " access-key ")
	t.Setenv("OBJECT_STORE_SECRET_ACCESS_KEY", " secret-key ")

	got, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv: %v", err)
	}
	if got.Provider != ProviderR2 || got.Bucket != "openplays-avatars" ||
		got.Endpoint != "https://account.r2.cloudflarestorage.com" || got.AccessKeyID != "access-key" ||
		got.SecretAccessKey != "secret-key" {
		t.Fatalf("ConfigFromEnv() = %#v", got)
	}
}

func TestConfigRejectsMalformedValues(t *testing.T) {
	base := Config{
		Provider: ProviderR2, Bucket: "openplays-avatars", PublicBaseURL: "https://images.openplays.app",
		Endpoint: "https://account.r2.cloudflarestorage.com", AccessKeyID: "access", SecretAccessKey: "secret",
	}
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{"provider", func(c *Config) { c.Provider = "s3" }},
		{"bucket", func(c *Config) { c.Bucket = "UPPERCASE" }},
		{"dotted bucket", func(c *Config) { c.Bucket = "images.openplays.app" }},
		{"base query", func(c *Config) { c.PublicBaseURL = "https://images.example.com?" }},
		{"bare base fragment", func(c *Config) { c.PublicBaseURL = "https://images.example.com#" }},
		{"remote HTTP base", func(c *Config) { c.PublicBaseURL = "http://images.example.com" }},
		{"missing endpoint", func(c *Config) { c.Endpoint = "" }},
		{"endpoint scheme", func(c *Config) { c.Endpoint = "account.r2.cloudflarestorage.com" }},
		{"endpoint path", func(c *Config) { c.Endpoint = "https://account.r2.cloudflarestorage.com/path" }},
		{"bare endpoint fragment", func(c *Config) { c.Endpoint = "https://account.r2.cloudflarestorage.com#" }},
		{"missing access key", func(c *Config) { c.AccessKeyID = "" }},
		{"missing secret key", func(c *Config) { c.SecretAccessKey = "" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base
			tt.mutate(&cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatalf("Validate accepted %#v", cfg)
			}
		})
	}
}

func TestConfigAllowsLoopbackDevelopmentURLs(t *testing.T) {
	cfg := Config{
		Provider: ProviderR2, Bucket: "openplays-avatars",
		Endpoint: "http://localhost:9000", PublicBaseURL: "http://localhost:9000/openplays-avatars",
		AccessKeyID: "access", SecretAccessKey: "secret",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}
