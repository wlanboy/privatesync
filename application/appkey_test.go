package application

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func testSalt() string {
	return base64.StdEncoding.EncodeToString([]byte("0123456789abcdef"))
}

func TestDeriveKeyLength(t *testing.T) {
	key, err := deriveKey("mysecret", testSalt())
	if err != nil {
		t.Fatalf("deriveKey returned error: %v", err)
	}
	if len(key) != aesKeyLength {
		t.Errorf("key length = %d, want %d", len(key), aesKeyLength)
	}
}

func TestDeriveKeyDeterministic(t *testing.T) {
	salt := testSalt()
	k1, _ := deriveKey("mysecret", salt)
	k2, _ := deriveKey("mysecret", salt)
	if !bytes.Equal(k1, k2) {
		t.Error("deriveKey should be deterministic for the same secret and salt")
	}
}

func TestDeriveKeyDifferentSecrets(t *testing.T) {
	salt := testSalt()
	k1, _ := deriveKey("secretone", salt)
	k2, _ := deriveKey("secrettwo", salt)
	if bytes.Equal(k1, k2) {
		t.Error("different secrets should derive different keys")
	}
}

func TestDeriveKeyDifferentSalts(t *testing.T) {
	k1, _ := deriveKey("mysecret", testSalt())
	k2, _ := deriveKey("mysecret", base64.StdEncoding.EncodeToString([]byte("fedcba9876543210")))
	if bytes.Equal(k1, k2) {
		t.Error("different salts should derive different keys")
	}
}

func TestDeriveKeyInvalidSalt(t *testing.T) {
	if _, err := deriveKey("mysecret", "not-valid-base64!!"); err == nil {
		t.Error("expected error for invalid base64 salt")
	}
}
