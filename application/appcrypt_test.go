package application

import (
	"bytes"
	"testing"
)

func TestNewFileName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"file.txt", "file..txt.crypt"},
		{"archive.tar.gz", "archive.tar..gz.crypt"},
		{"document.pdf", "document..pdf.crypt"},
	}
	for _, tt := range tests {
		got := newFileName(tt.input)
		if got != tt.expected {
			t.Errorf("newFileName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestOldFileName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"file..txt.crypt", "file.txt"},
		{"document..pdf.crypt", "document.pdf"},
		{"archive.tar..gz.crypt", "archive.tar.gz"},
	}
	for _, tt := range tests {
		got := oldFileName(tt.input)
		if got != tt.expected {
			t.Errorf("oldFileName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestNewOldFileNameRoundtrip(t *testing.T) {
	inputs := []string{"file.txt", "document.pdf", "archive.tar.gz"}
	for _, name := range inputs {
		got := oldFileName(newFileName(name))
		if got != name {
			t.Errorf("roundtrip(%q): got %q", name, got)
		}
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key, _ := deriveKey("mysecretpassword", testSalt())
	original := []byte("Hello, this is test data!")

	encrypted, err := encryptData(original, key)
	if err != nil {
		t.Fatalf("encryptData returned error: %v", err)
	}
	if bytes.Equal(encrypted, original) {
		t.Fatal("encrypted data should differ from original")
	}

	decrypted, err := decryptData(encrypted, key)
	if err != nil {
		t.Fatalf("decryptData returned error: %v", err)
	}
	if !bytes.Equal(decrypted, original) {
		t.Errorf("roundtrip failed: got %q, want %q", decrypted, original)
	}
}

func TestEncryptDifferentSecretFails(t *testing.T) {
	correctKey, _ := deriveKey("correctsecret", testSalt())
	wrongKey, _ := deriveKey("wrongsecret", testSalt())

	original := []byte("sensitive data")
	encrypted, err := encryptData(original, correctKey)
	if err != nil {
		t.Fatalf("encryptData returned error: %v", err)
	}

	if _, err := decryptData(encrypted, wrongKey); err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestEncryptProducesUniqueCiphertext(t *testing.T) {
	key, _ := deriveKey("samesecret", testSalt())
	data := []byte("same plaintext")

	c1, _ := encryptData(data, key)
	c2, _ := encryptData(data, key)
	if bytes.Equal(c1, c2) {
		t.Error("two encryptions of the same plaintext should differ (random nonce)")
	}
}
