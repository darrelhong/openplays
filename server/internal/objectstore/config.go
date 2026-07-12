package objectstore

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"unicode"
)

type Provider string

const ProviderR2 Provider = "r2"

type Config struct {
	Provider        Provider
	Bucket          string
	PublicBaseURL   string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
}

func ConfigFromEnv() (Config, error) {
	cfg := Config{
		Provider:        Provider(strings.ToLower(strings.TrimSpace(os.Getenv("OBJECT_STORE_PROVIDER")))),
		Bucket:          strings.TrimSpace(os.Getenv("OBJECT_STORE_BUCKET")),
		PublicBaseURL:   strings.TrimSpace(os.Getenv("OBJECT_STORE_PUBLIC_BASE_URL")),
		Endpoint:        strings.TrimSpace(os.Getenv("OBJECT_STORE_ENDPOINT")),
		AccessKeyID:     strings.TrimSpace(os.Getenv("OBJECT_STORE_ACCESS_KEY_ID")),
		SecretAccessKey: strings.TrimSpace(os.Getenv("OBJECT_STORE_SECRET_ACCESS_KEY")),
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Provider != ProviderR2 {
		return fmt.Errorf("unsupported object store provider %q; available provider: %q", c.Provider, ProviderR2)
	}
	if err := validateBucket(c.Bucket); err != nil {
		return err
	}
	if err := validatePublicBaseURL(c.PublicBaseURL); err != nil {
		return err
	}
	if c.Endpoint == "" {
		return errors.New("R2 endpoint is required")
	}
	if err := validateEndpointURL(c.Endpoint); err != nil {
		return err
	}
	if c.AccessKeyID == "" || c.SecretAccessKey == "" {
		return errors.New("R2 access key ID and secret access key are required")
	}
	return nil
}

func validateBucket(bucket string) error {
	if len(bucket) < 3 || len(bucket) > 63 ||
		!isLowerAlphaNumeric(bucket[0]) ||
		!isLowerAlphaNumeric(bucket[len(bucket)-1]) ||
		strings.Contains(bucket, ".") ||
		net.ParseIP(bucket) != nil {
		return errors.New("object store bucket must contain 3-63 lowercase letters, numbers, or hyphens")
	}
	for i := range len(bucket) {
		char := bucket[i]
		if !isLowerAlphaNumeric(char) && char != '-' {
			return errors.New("object store bucket must contain 3-63 lowercase letters, numbers, or hyphens")
		}
	}
	return nil
}

func isLowerAlphaNumeric(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= '0' && char <= '9'
}

func validateEndpointURL(value string) error {
	if hasUnsafeURLCharacters(value) || strings.ContainsAny(value, "?#") {
		return errors.New("R2 endpoint must be an HTTP or HTTPS origin")
	}
	parsed, err := url.Parse(value)
	if err != nil ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") ||
		(parsed.Scheme == "http" && !isLoopbackHost(parsed.Hostname())) ||
		parsed.Host == "" || parsed.User != nil || parsed.Opaque != "" ||
		!validURLPort(parsed) ||
		(parsed.Path != "" && parsed.Path != "/") ||
		parsed.ForceQuery || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("R2 endpoint must be an HTTP or HTTPS origin")
	}
	return nil
}

func validatePublicBaseURL(value string) error {
	if hasUnsafeURLCharacters(value) || strings.ContainsAny(value, "?#") {
		return errors.New("object store public base URL must be HTTPS, or loopback HTTP for development")
	}
	parsed, err := url.Parse(value)
	if err != nil ||
		(parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname()))) ||
		parsed.Host == "" || parsed.User != nil || parsed.Opaque != "" ||
		!validURLPort(parsed) || !validBasePath(parsed.Path) ||
		parsed.ForceQuery || parsed.RawQuery != "" || parsed.Fragment != "" {
		return errors.New("object store public base URL must be HTTPS, or loopback HTTP for development")
	}
	return nil
}

func validURLPort(parsed *url.URL) bool {
	if strings.HasSuffix(parsed.Host, ":") {
		return false
	}
	port := parsed.Port()
	if port == "" {
		return true
	}
	number, err := strconv.Atoi(port)
	return err == nil && number >= 1 && number <= 65535
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validBasePath(path string) bool {
	if hasUnsafeURLCharacters(path) || strings.Contains(path, "\\") {
		return false
	}
	if path == "" || path == "/" {
		return true
	}
	for _, segment := range strings.Split(strings.Trim(path, "/"), "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func hasUnsafeURLCharacters(value string) bool {
	return strings.IndexFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) != -1
}
