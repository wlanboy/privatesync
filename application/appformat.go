package application

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
)

/*
File format (version 1):

	header: magic "PSYNC" | version (1 byte) | mtime (int64, unix seconds, big endian) | nonce prefix (7 bytes)
	body:   sequence of AES-256-GCM sealed chunks of chunkSize plaintext bytes each (last one may be shorter or empty)

The nonce of chunk i is nonce prefix | i (uint32, big endian) | last flag (1 byte),
so reordering, dropping or appending chunks is detected. The additional data of
every chunk is header | relative path (slash separated), which binds the content
to its mtime and to its location in the tree.
*/
const (
	formatMagic     = "PSYNC"
	formatVersion   = byte(1)
	chunkSize       = 64 * 1024
	noncePrefixSize = 7
	headerSize      = len(formatMagic) + 1 + 8 + noncePrefixSize
	gcmOverhead     = 16
)

var (
	errNotEncrypted       = errors.New("not a privatesync file (missing header)")
	errUnsupportedVersion = errors.New("unsupported privatesync file version")
	errAuthentication     = errors.New("authentication failed (wrong key, corrupted file or file was renamed)")
	errTooLarge           = errors.New("file too large")
)

type fileHeader struct {
	ModTime     int64
	NoncePrefix [noncePrefixSize]byte
}

func (h fileHeader) marshal() []byte {
	raw := make([]byte, 0, headerSize)
	raw = append(raw, formatMagic...)
	raw = append(raw, formatVersion)
	raw = binary.BigEndian.AppendUint64(raw, uint64(h.ModTime))
	return append(raw, h.NoncePrefix[:]...)
}

func readHeader(r io.Reader) (fileHeader, []byte, error) {
	var h fileHeader
	raw := make([]byte, headerSize)
	if _, err := io.ReadFull(r, raw); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return h, nil, errNotEncrypted
		}
		return h, nil, err
	}
	if string(raw[:len(formatMagic)]) != formatMagic {
		return h, nil, errNotEncrypted
	}
	pos := len(formatMagic)
	if raw[pos] != formatVersion {
		return h, nil, fmt.Errorf("%w: %d", errUnsupportedVersion, raw[pos])
	}
	pos++
	h.ModTime = int64(binary.BigEndian.Uint64(raw[pos : pos+8]))
	pos += 8
	copy(h.NoncePrefix[:], raw[pos:])
	return h, raw, nil
}

/*readFileHeader reads only the header of an encrypted file (not authenticated)*/
func readFileHeader(path string) (fileHeader, error) {
	f, err := os.Open(path)
	if err != nil {
		return fileHeader{}, err
	}
	defer f.Close()
	h, _, err := readHeader(f)
	return h, err
}

/*encryptedSize returns the size of an encrypted file for a plaintext of the given size*/
func encryptedSize(plainSize int64) int64 {
	chunks := (plainSize + chunkSize - 1) / chunkSize
	if chunks == 0 {
		chunks = 1
	}
	return int64(headerSize) + plainSize + chunks*gcmOverhead
}

func newAEAD(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func chunkNonce(prefix [noncePrefixSize]byte, counter uint32, last bool) []byte {
	nonce := make([]byte, 0, noncePrefixSize+5)
	nonce = append(nonce, prefix[:]...)
	nonce = binary.BigEndian.AppendUint32(nonce, counter)
	if last {
		return append(nonce, 1)
	}
	return append(nonce, 0)
}

func additionalData(rawHeader []byte, relPath string) []byte {
	ad := make([]byte, 0, len(rawHeader)+len(relPath))
	ad = append(ad, rawHeader...)
	return append(ad, relPath...)
}

/*isLastChunk reports whether the reader is exhausted after a full chunk was read*/
func isLastChunk(br *bufio.Reader, readErr error) (bool, error) {
	switch {
	case errors.Is(readErr, io.EOF), errors.Is(readErr, io.ErrUnexpectedEOF):
		return true, nil
	case readErr != nil:
		return false, readErr
	}
	if _, err := br.Peek(1); err != nil {
		if errors.Is(err, io.EOF) {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

/*encryptStream writes header and encrypted chunks of src to dst*/
func encryptStream(dst io.Writer, src io.Reader, key []byte, relPath string, modTime int64) error {
	aead, err := newAEAD(key)
	if err != nil {
		return err
	}
	h := fileHeader{ModTime: modTime}
	if _, err := rand.Read(h.NoncePrefix[:]); err != nil {
		return err
	}
	raw := h.marshal()
	if _, err := dst.Write(raw); err != nil {
		return err
	}
	ad := additionalData(raw, relPath)

	br := bufio.NewReaderSize(src, chunkSize)
	buf := make([]byte, chunkSize)
	sealed := make([]byte, 0, chunkSize+aead.Overhead())
	for counter := uint32(0); ; counter++ {
		n, readErr := io.ReadFull(br, buf)
		last, err := isLastChunk(br, readErr)
		if err != nil {
			return err
		}
		sealed = aead.Seal(sealed[:0], chunkNonce(h.NoncePrefix, counter, last), buf[:n], ad)
		if _, err := dst.Write(sealed); err != nil {
			return err
		}
		if last {
			return nil
		}
		if counter == math.MaxUint32 {
			return errTooLarge
		}
	}
}

/*
decryptStream verifies and decrypts src into dst; dst may already contain
plaintext when an error is returned, so callers must discard it in that case
*/
func decryptStream(dst io.Writer, src io.Reader, key []byte, relPath string) (fileHeader, error) {
	h, raw, err := readHeader(src)
	if err != nil {
		return h, err
	}
	aead, err := newAEAD(key)
	if err != nil {
		return h, err
	}
	ad := additionalData(raw, relPath)

	sealedSize := chunkSize + aead.Overhead()
	br := bufio.NewReaderSize(src, sealedSize)
	buf := make([]byte, sealedSize)
	plain := make([]byte, 0, chunkSize)
	for counter := uint32(0); ; counter++ {
		n, readErr := io.ReadFull(br, buf)
		if errors.Is(readErr, io.EOF) {
			// no data left although the previous chunk was not marked as last
			return h, errAuthentication
		}
		last, err := isLastChunk(br, readErr)
		if err != nil {
			return h, err
		}
		plain, err = aead.Open(plain[:0], chunkNonce(h.NoncePrefix, counter, last), buf[:n], ad)
		if err != nil {
			return h, errAuthentication
		}
		if _, err := dst.Write(plain); err != nil {
			return h, err
		}
		if last {
			return h, nil
		}
		if counter == math.MaxUint32 {
			return h, errTooLarge
		}
	}
}
