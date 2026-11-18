package application

import (
	"bytes"
	"crypto/rand"
	"errors"
	"sync"
	"testing"
)

var (
	keyOnce  sync.Once
	keyCache [2][]byte
)

/*testKeys returns two different derived keys; scrypt is slow, so they are computed once*/
func testKeys(t *testing.T) (key, otherKey []byte) {
	t.Helper()
	keyOnce.Do(func() {
		keyCache[0], _ = deriveKey("mysecretpassword", testSalt())
		keyCache[1], _ = deriveKey("othersecret", testSalt())
	})
	if keyCache[0] == nil || keyCache[1] == nil {
		t.Fatal("deriveKey failed")
	}
	return keyCache[0], keyCache[1]
}

func randomBytes(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return b
}

func encryptBytes(t *testing.T, data, key []byte, relPath string, modTime int64) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := encryptStream(&buf, bytes.NewReader(data), key, relPath, modTime); err != nil {
		t.Fatalf("encryptStream: %v", err)
	}
	return buf.Bytes()
}

func decryptBytes(data, key []byte, relPath string) ([]byte, fileHeader, error) {
	var buf bytes.Buffer
	h, err := decryptStream(&buf, bytes.NewReader(data), key, relPath)
	return buf.Bytes(), h, err
}

func TestStreamRoundtripSizes(t *testing.T) {
	key, _ := testKeys(t)
	sizes := []int{0, 1, chunkSize - 1, chunkSize, chunkSize + 1, 3 * chunkSize, 3*chunkSize + 17}
	for _, size := range sizes {
		original := randomBytes(t, size)
		encrypted := encryptBytes(t, original, key, "dir/file.bin", 1700000000)

		if got, want := int64(len(encrypted)), encryptedSize(int64(size)); got != want {
			t.Errorf("size %d: encrypted length %d, encryptedSize says %d", size, got, want)
		}
		decrypted, h, err := decryptBytes(encrypted, key, "dir/file.bin")
		if err != nil {
			t.Fatalf("size %d: decryptStream: %v", size, err)
		}
		if !bytes.Equal(decrypted, original) {
			t.Errorf("size %d: roundtrip mismatch", size)
		}
		if h.ModTime != 1700000000 {
			t.Errorf("size %d: ModTime = %d", size, h.ModTime)
		}
	}
}

func TestStreamUniqueCiphertext(t *testing.T) {
	key, _ := testKeys(t)
	data := []byte("same plaintext")
	c1 := encryptBytes(t, data, key, "a", 1)
	c2 := encryptBytes(t, data, key, "a", 1)
	if bytes.Equal(c1, c2) {
		t.Error("two encryptions of the same plaintext should differ (random nonce prefix)")
	}
}

func TestStreamRejectsManipulation(t *testing.T) {
	key, otherKey := testKeys(t)
	original := randomBytes(t, 2*chunkSize+100)
	encrypted := encryptBytes(t, original, key, "dir/file.bin", 1700000000)
	sealed := chunkSize + gcmOverhead

	flipped := bytes.Clone(encrypted)
	flipped[headerSize+10] ^= 0x01

	changedTime := bytes.Clone(encrypted)
	changedTime[len(formatMagic)+1+7] ^= 0x01

	tests := []struct {
		name    string
		data    []byte
		key     []byte
		relPath string
	}{
		{"wrong key", encrypted, otherKey, "dir/file.bin"},
		{"renamed file", encrypted, key, "dir/other.bin"},
		{"moved file", encrypted, key, "file.bin"},
		{"flipped bit", flipped, key, "dir/file.bin"},
		{"changed mtime", changedTime, key, "dir/file.bin"},
		{"last chunk dropped", encrypted[:headerSize+2*sealed], key, "dir/file.bin"},
		{"cut in the middle", encrypted[:headerSize+sealed+50], key, "dir/file.bin"},
		{"only header", encrypted[:headerSize], key, "dir/file.bin"},
		{"appended data", append(bytes.Clone(encrypted), 0x00), key, "dir/file.bin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := decryptBytes(tt.data, tt.key, tt.relPath); !errors.Is(err, errAuthentication) {
				t.Errorf("err = %v, want errAuthentication", err)
			}
		})
	}
}

func TestStreamSwappedChunksRejected(t *testing.T) {
	key, _ := testKeys(t)
	encrypted := encryptBytes(t, randomBytes(t, 3*chunkSize), key, "f", 1)
	sealed := chunkSize + gcmOverhead
	first := headerSize
	second := headerSize + sealed

	swapped := bytes.Clone(encrypted)
	copy(swapped[first:first+sealed], encrypted[second:second+sealed])
	copy(swapped[second:second+sealed], encrypted[first:first+sealed])
	if _, _, err := decryptBytes(swapped, key, "f"); !errors.Is(err, errAuthentication) {
		t.Errorf("err = %v, want errAuthentication", err)
	}
}

func TestReadHeaderInvalidInput(t *testing.T) {
	key, _ := testKeys(t)
	valid := encryptBytes(t, []byte("x"), key, "f", 1)
	wrongVersion := bytes.Clone(valid)
	wrongVersion[len(formatMagic)] = 99

	tests := []struct {
		name string
		data []byte
		want error
	}{
		{"empty", nil, errNotEncrypted},
		{"short", []byte("abc"), errNotEncrypted},
		{"plain text", []byte("this is just some plain text file content"), errNotEncrypted},
		{"wrong version", wrongVersion, errUnsupportedVersion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := decryptBytes(tt.data, key, "f"); !errors.Is(err, tt.want) {
				t.Errorf("err = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestEncryptedSize(t *testing.T) {
	tests := []struct {
		plain int64
		want  int64
	}{
		{0, int64(headerSize) + gcmOverhead},
		{1, int64(headerSize) + 1 + gcmOverhead},
		{chunkSize, int64(headerSize) + chunkSize + gcmOverhead},
		{chunkSize + 1, int64(headerSize) + chunkSize + 1 + 2*gcmOverhead},
	}
	for _, tt := range tests {
		if got := encryptedSize(tt.plain); got != tt.want {
			t.Errorf("encryptedSize(%d) = %d, want %d", tt.plain, got, tt.want)
		}
	}
}
