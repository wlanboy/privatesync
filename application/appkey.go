package application

import (
	"encoding/base64"

	"golang.org/x/crypto/scrypt"
)

const (
	scryptN      = 1 << 15
	scryptR      = 8
	scryptP      = 1
	aesKeyLength = 32
)

/*deriveKey turns the user's plaintext secret and a stored salt into an AES-256 key via scrypt*/
func deriveKey(secret string, saltB64 string) ([]byte, error) {
	salt, err := base64.StdEncoding.DecodeString(saltB64)
	if err != nil {
		return nil, err
	}
	return scrypt.Key([]byte(secret), salt, scryptN, scryptR, scryptP, aesKeyLength)
}
