// internal/x3dh/crypto.go
package x3dh

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenKeyPair creates an X25519 key pair and returns (private, public[32], error)
// This function is optimized for MPU devices with limited resources.
func GenKeyPair() (*ecdh.PrivateKey, [32]byte, error) {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, [32]byte{}, fmt.Errorf("failed to generate X25519 key pair: %v", err)
	}
	var pub32 [32]byte
	copy(pub32[:], priv.PublicKey().Bytes())
	return priv, pub32, nil
}

func DH(priv *ecdh.PrivateKey, pub32 *[32]byte) (out [32]byte, err error) {
	if priv == nil {
		return out, fmt.Errorf("private key is nil")
	}
	pub, err := ecdh.X25519().NewPublicKey(pub32[:])
	if err != nil {
		return out, fmt.Errorf("failed to create public key from bytes: %v", err)
	}
	shared, err := priv.ECDH(pub)
	if err != nil {
		return out, fmt.Errorf("failed to compute DH shared secret: %v", err)
	}
	copy(out[:], shared)
	return out, nil
}

func KDF(parts ...[32]byte) (out [32]byte) {
	h := sha256.New()
	for _, p := range parts {
		h.Write(p[:])
	}
	copy(out[:], h.Sum(nil))
	return
}

// encode32 converts a [32]byte public key to hex string for JSON serialization.
func encode32(pk [32]byte) string { 
	return hex.EncodeToString(pk[:]) 
}

// decode32 converts a hex string back to [32]byte for deserialization.
// Returns error instead of panic for better error handling.
func decode32(s string) ([32]byte, error) {
	var out [32]byte
	raw, err := hex.DecodeString(s)
	if err != nil {
		return out, fmt.Errorf("failed to decode hex string: %v", err)
	}
	if len(raw) != 32 {
		return out, fmt.Errorf("expected 32 bytes, got %d", len(raw))
	}
	copy(out[:], raw)
	return out, nil
}

// ValidatePublicKey checks if a hex string represents a valid X25519 public key.
// Useful for input validation in MPU applications.
func ValidatePublicKey(hexKey string) error {
	_, err := decode32(hexKey)
	if err != nil {
		return fmt.Errorf("invalid public key format: %v", err)
	}
	return nil
}

// GetKeyFingerprint returns a short fingerprint of a public key for display purposes.
// Useful for MPU devices with limited display capabilities.
func GetKeyFingerprint(pk [32]byte) string {
	// Return first 8 characters of hex encoding
	return hex.EncodeToString(pk[:4])
}

