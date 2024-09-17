package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
)

type keyFile struct {
	D string `json:"d"`
	X string `json:"x"`
	Y string `json:"y"`
}

// SaveKey writes a private key to path as JSON (hex-encoded D, X, Y).
func SaveKey(path string, priv *ecdsa.PrivateKey) error {
	if priv == nil {
		return nil
	}
	k := keyFile{
		D: hex.EncodeToString(priv.D.Bytes()),
		X: hex.EncodeToString(priv.PublicKey.X.Bytes()),
		Y: hex.EncodeToString(priv.PublicKey.Y.Bytes()),
	}
	data, err := json.MarshalIndent(k, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// LoadKey reads a private key from path.
func LoadKey(path string) (*ecdsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var k keyFile
	if err := json.Unmarshal(data, &k); err != nil {
		return nil, err
	}
	d, err := hex.DecodeString(k.D)
	if err != nil || len(d) == 0 {
		return nil, err
	}
	x, err := hex.DecodeString(k.X)
	if err != nil {
		return nil, err
	}
	y, err := hex.DecodeString(k.Y)
	if err != nil {
		return nil, err
	}
	priv := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{Curve: elliptic.P256()},
		D:         new(big.Int).SetBytes(d),
	}
	priv.PublicKey.X = new(big.Int).SetBytes(x)
	priv.PublicKey.Y = new(big.Int).SetBytes(y)
	return priv, nil
}
