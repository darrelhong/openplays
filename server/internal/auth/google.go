package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// Google's JWKS endpoint for verifying ID tokens.
const googleJWKSURL = "https://www.googleapis.com/oauth2/v3/certs"

// GoogleClaims holds the verified claims from a Google ID token.
type GoogleClaims struct {
	Subject   string // Google user ID ("sub")
	Email     string // verified email
	Name      string // full name
	Picture   string // profile picture URL
	ExpiresAt time.Time
}

// GoogleVerifier verifies Google ID tokens using Google's public JWKS keys.
type GoogleVerifier struct {
	clientID string
	httpGet  func(url string) (*http.Response, error) // injectable for testing

	mu   sync.RWMutex
	keys map[string]*rsa.PublicKey
	exp  time.Time
}

// NewGoogleVerifier creates a verifier for the given Google OAuth client ID.
func NewGoogleVerifier(clientID string) *GoogleVerifier {
	return &GoogleVerifier{
		clientID: clientID,
		httpGet:  http.Get,
	}
}

// Verify validates a Google ID token and returns the claims.
// It checks: signature (RS256), issuer, audience, and expiry.
func (v *GoogleVerifier) Verify(rawToken string) (*GoogleClaims, error) {
	// Parse header to get kid
	parts, err := splitJWT(rawToken)
	if err != nil {
		return nil, err
	}

	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if err := decodeSegment(parts[0], &header); err != nil {
		return nil, fmt.Errorf("decode header: %w", err)
	}
	if header.Alg != "RS256" {
		return nil, fmt.Errorf("unsupported alg: %s", header.Alg)
	}

	// Get signing key
	key, err := v.getKey(header.Kid)
	if err != nil {
		return nil, err
	}

	// Verify signature
	if err := verifyRS256(parts[0]+"."+parts[1], parts[2], key); err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}

	// Decode and validate payload
	var payload struct {
		Iss           string `json:"iss"`
		Aud           string `json:"aud"`
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified any    `json:"email_verified"` // can be bool or string
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Exp           int64  `json:"exp"`
		Iat           int64  `json:"iat"`
	}
	if err := decodeSegment(parts[1], &payload); err != nil {
		return nil, fmt.Errorf("decode payload: %w", err)
	}

	// Check issuer
	if payload.Iss != "accounts.google.com" && payload.Iss != "https://accounts.google.com" {
		return nil, fmt.Errorf("invalid issuer: %s", payload.Iss)
	}

	// Check audience
	if payload.Aud != v.clientID {
		return nil, fmt.Errorf("invalid audience: %s", payload.Aud)
	}

	// Check expiry
	expTime := time.Unix(payload.Exp, 0)
	if time.Now().After(expTime) {
		return nil, errors.New("token expired")
	}

	// Check email verified
	if !isEmailVerified(payload.EmailVerified) {
		return nil, errors.New("email not verified")
	}

	return &GoogleClaims{
		Subject:   payload.Sub,
		Email:     payload.Email,
		Name:      payload.Name,
		Picture:   payload.Picture,
		ExpiresAt: expTime,
	}, nil
}

func isEmailVerified(v any) bool {
	switch ev := v.(type) {
	case bool:
		return ev
	case string:
		return ev == "true"
	default:
		return false
	}
}

// getKey returns the RSA public key for the given key ID, fetching/caching JWKS as needed.
func (v *GoogleVerifier) getKey(kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	if key, ok := v.keys[kid]; ok && time.Now().Before(v.exp) {
		v.mu.RUnlock()
		return key, nil
	}
	v.mu.RUnlock()

	// Fetch fresh keys
	v.mu.Lock()
	defer v.mu.Unlock()

	// Double-check after acquiring write lock
	if key, ok := v.keys[kid]; ok && time.Now().Before(v.exp) {
		return key, nil
	}

	keys, err := v.fetchJWKS()
	if err != nil {
		return nil, fmt.Errorf("fetch JWKS: %w", err)
	}
	v.keys = keys
	v.exp = time.Now().Add(1 * time.Hour)

	key, ok := v.keys[kid]
	if !ok {
		return nil, fmt.Errorf("key %q not found in JWKS", kid)
	}
	return key, nil
}

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
}

func (v *GoogleVerifier) fetchJWKS() (map[string]*rsa.PublicKey, error) {
	resp, err := v.httpGet(googleJWKSURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS HTTP %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("decode JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, fmt.Errorf("decode modulus for kid=%s: %w", k.Kid, err)
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, fmt.Errorf("decode exponent for kid=%s: %w", k.Kid, err)
		}
		n := new(big.Int).SetBytes(nBytes)
		e := int(new(big.Int).SetBytes(eBytes).Int64())

		keys[k.Kid] = &rsa.PublicKey{N: n, E: e}
	}

	return keys, nil
}
