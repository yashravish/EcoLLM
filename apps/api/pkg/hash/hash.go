package hash

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	keyPrefix  = "eco-sk-"
	bcryptCost = 12 // bcrypt cost; high enough to slow brute-force, low enough to not DDoS ourselves
	rawKeyLen  = 32 // bytes of entropy
)

// GenerateAPIKey produces a plaintext API key, its bcrypt hash for storage,
// and the display prefix used for identification without exposing the full key.
//
// Returns (plaintext, hash, prefix).
// The plaintext is shown to the user exactly once and never stored.
func GenerateAPIKey() (plaintext, hash, prefix string, err error) {
	raw := make([]byte, rawKeyLen)
	if _, err = rand.Read(raw); err != nil {
		return "", "", "", fmt.Errorf("generate random bytes: %w", err)
	}

	// URL-safe base64 gives a compact, printable token.
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	plaintext = keyPrefix + encoded

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcryptCost)
	if err != nil {
		return "", "", "", fmt.Errorf("bcrypt hash: %w", err)
	}
	hash = string(hashBytes)

	// Prefix: "eco-sk-" + first 3 chars of the encoded token (safe to display)
	if len(plaintext) >= 10 {
		prefix = plaintext[:10]
	} else {
		prefix = plaintext
	}

	return plaintext, hash, prefix, nil
}

// CompareAPIKey verifies rawKey against the stored bcrypt hash.
// Returns nil if they match.
func CompareAPIKey(rawKey, storedHash string) error {
	return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(rawKey))
}
