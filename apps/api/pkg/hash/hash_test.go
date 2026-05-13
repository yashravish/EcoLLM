package hash

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestGenerateAPIKey_Format(t *testing.T) {
	plain, hash, prefix, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}

	// Plaintext must start with the well-known prefix.
	if !strings.HasPrefix(plain, keyPrefix) {
		t.Errorf("plaintext %q does not start with %q", plain, keyPrefix)
	}

	// Hash must be a valid bcrypt hash.
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)); err != nil {
		t.Errorf("hash does not match plaintext: %v", err)
	}

	// Display prefix: "eco-sk-" + 3 chars of encoded token = 10 chars.
	if !strings.HasPrefix(prefix, keyPrefix) {
		t.Errorf("prefix %q does not start with %q", prefix, keyPrefix)
	}
	if len(prefix) != 10 {
		t.Errorf("prefix len = %d, want 10", len(prefix))
	}

	// The prefix must be a left-slice of the plaintext.
	if !strings.HasPrefix(plain, prefix) {
		t.Errorf("plaintext %q does not begin with prefix %q", plain, prefix)
	}
}

func TestGenerateAPIKey_HashNotEqualToPlaintext(t *testing.T) {
	plain, hash, _, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}
	if plain == hash {
		t.Error("plaintext and hash must not be equal")
	}
}

func TestGenerateAPIKey_Uniqueness(t *testing.T) {
	const n = 10
	keys := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		plain, _, _, err := GenerateAPIKey()
		if err != nil {
			t.Fatalf("GenerateAPIKey() error on iteration %d: %v", i, err)
		}
		if keys[plain] {
			t.Errorf("duplicate key generated: %q", plain)
		}
		keys[plain] = true
	}
}

func TestGenerateAPIKey_SufficientLength(t *testing.T) {
	plain, _, _, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}
	// 32 raw bytes base64-encoded = 43 chars; plus "eco-sk-" prefix.
	const minLen = len(keyPrefix) + 40
	if len(plain) < minLen {
		t.Errorf("plaintext length %d < minimum %d", len(plain), minLen)
	}
}

func TestCompareAPIKey_Match(t *testing.T) {
	plain, hash, _, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}
	if err := CompareAPIKey(plain, hash); err != nil {
		t.Errorf("CompareAPIKey() unexpected error: %v", err)
	}
}

func TestCompareAPIKey_Mismatch(t *testing.T) {
	plain, hash, _, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}
	if err := CompareAPIKey(plain+"wrong", hash); err == nil {
		t.Error("CompareAPIKey() expected error for wrong key, got nil")
	}
}

func TestCompareAPIKey_InvalidHash(t *testing.T) {
	if err := CompareAPIKey("any-key", "not-a-bcrypt-hash"); err == nil {
		t.Error("CompareAPIKey() expected error for invalid hash, got nil")
	}
}

func TestCompareAPIKey_EmptyKey(t *testing.T) {
	_, hash, _, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error: %v", err)
	}
	if err := CompareAPIKey("", hash); err == nil {
		t.Error("CompareAPIKey() expected error for empty key, got nil")
	}
}
