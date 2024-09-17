package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math/big"
)

// AddressSize is length of an account address (20 bytes, Ethereum-style).
const AddressSize = 20

// Address is a 20-byte account identifier derived from pubkey.
type Address [AddressSize]byte

// Signature is an ECDSA signature (R || S, 64 bytes for P-256).
type Signature []byte

// GenerateKey creates a new ECDSA P-256 key pair.
func GenerateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

// PubkeyToAddress derives a 20-byte address from an ECDSA public key (SHA256 of pubkey).
func PubkeyToAddress(pub *ecdsa.PublicKey) Address {
	if pub == nil {
		return Address{}
	}
	b := elliptic.MarshalCompressed(elliptic.P256(), pub.X, pub.Y)
	h := sha256.Sum256(b)
	var a Address
	copy(a[:], h[:AddressSize])
	return a
}

// AddressFromHex decodes a hex string into Address.
func AddressFromHex(s string) (Address, error) {
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != AddressSize {
		return Address{}, errors.New("invalid address hex")
	}
	var a Address
	copy(a[:], b)
	return a, nil
}

// Hex returns hex-encoded address.
func (a Address) Hex() string { return hex.EncodeToString(a[:]) }

// Sign signs the hash with the private key. Returns 64 bytes R||S.
func Sign(priv *ecdsa.PrivateKey, hash Hash) (Signature, error) {
	r, s, err := ecdsa.Sign(rand.Reader, priv, hash[:])
	if err != nil {
		return nil, err
	}
	// P-256: 32-byte R and 32-byte S
	sig := make(Signature, 64)
	r.FillBytes(sig[0:32])
	s.FillBytes(sig[32:64])
	return sig, nil
}

// Verify verifies an ECDSA signature over the given hash.
func Verify(pub *ecdsa.PublicKey, hash Hash, sig Signature) bool {
	if pub == nil || len(sig) < 64 {
		return false
	}
	r := new(big.Int).SetBytes(sig[0:32])
	s := new(big.Int).SetBytes(sig[32:64])
	return ecdsa.Verify(pub, hash[:], r, s)
}

// HashMessage returns SHA256 of message (used for signing txs).
func HashMessage(msg []byte) Hash {
	return sha256.Sum256(msg)
}

// MarshalPubkey returns compressed public key bytes (33 bytes for P-256).
func MarshalPubkey(pub *ecdsa.PublicKey) []byte {
	if pub == nil {
		return nil
	}
	return elliptic.MarshalCompressed(elliptic.P256(), pub.X, pub.Y)
}

// UnmarshalPubkey parses compressed public key bytes.
func UnmarshalPubkey(b []byte) *ecdsa.PublicKey {
	if len(b) < 33 {
		return nil
	}
	x, y := elliptic.UnmarshalCompressed(elliptic.P256(), b)
	if x == nil {
		return nil
	}
	return &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}
}
