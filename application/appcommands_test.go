package application

import (
	"testing"
)

func TestCreateHash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "d41d8cd98f00b204e9800998ecf8427e"},
		{"secret", "5ebe2294ecd0e0f08eab7690d2a6ee69"},
		{"hello", "5d41402abc4b2a76b9719d911017c592"},
	}
	for _, tt := range tests {
		got := createHash(tt.input)
		if got != tt.expected {
			t.Errorf("createHash(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCreateHashLength(t *testing.T) {
	hash := createHash("anyinput")
	if len(hash) != 32 {
		t.Errorf("MD5 hex string should be 32 chars, got %d", len(hash))
	}
}
