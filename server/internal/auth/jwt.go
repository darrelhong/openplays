package auth

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

// splitJWT splits a raw JWT into its 3 base64url-encoded parts.
func splitJWT(raw string) ([]string, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid JWT: expected 3 parts")
	}
	return parts, nil
}

// decodeSegment base64url-decodes a JWT segment and unmarshals into dst.
func decodeSegment(seg string, dst any) error {
	b, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dst)
}

// verifyRS256 verifies an RS256 signature over the given message.
func verifyRS256(message, sig string, key *rsa.PublicKey) error {
	sigBytes, err := base64.RawURLEncoding.DecodeString(sig)
	if err != nil {
		return err
	}
	hash := sha256.Sum256([]byte(message))
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, hash[:], sigBytes)
}
