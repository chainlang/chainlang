package crypto

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashSize is the length of a SHA256 hash in bytes.
const HashSize = 32

// Hash represents a 32-byte SHA256 hash.
type Hash [HashSize]byte

// HashBytes computes SHA256 of data.
func HashBytes(data []byte) Hash {
	return sha256.Sum256(data)
}

// HashFromHex decodes a hex string into Hash. Returns zero hash on error.
func HashFromHex(s string) (h Hash) {
	b, _ := hex.DecodeString(s)
	if len(b) != HashSize {
		return h
	}
	copy(h[:], b)
	return h
}

// Bytes returns the hash as a slice.
func (h Hash) Bytes() []byte { return h[:] }

// Hex returns hex-encoded hash.
func (h Hash) Hex() string { return hex.EncodeToString(h[:]) }

// IsZero returns true if h is the zero hash.
func (h Hash) IsZero() bool {
	for i := range h {
		if h[i] != 0 {
			return false
		}
	}
	return true
}
