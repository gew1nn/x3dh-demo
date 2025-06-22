package x3dh

import (
	"testing"
)

func TestGenKeyPair(t *testing.T) {
	priv, pub, err := GenKeyPair()
	if err != nil {
		t.Fatalf("GenKeyPair failed: %v", err)
	}
	if priv == nil {
		t.Fatal("Private key should not be nil")
	}
	if pub == [32]byte{} {
		t.Fatal("Public key should not be zero")
	}
	// Verify the public key matches the private key
	pubFromPriv := priv.PublicKey().Bytes()
	if len(pubFromPriv) != 32 {
		t.Fatal("Public key should be 32 bytes")
	}
	var expectedPub [32]byte
	copy(expectedPub[:], pubFromPriv)
	if pub != expectedPub {
		t.Fatal("Public key mismatch")
	}
}

func TestDH(t *testing.T) {
	priv1, pub1, err := GenKeyPair()
	if err != nil {
		t.Fatalf("GenKeyPair failed: %v", err)
	}
	priv2, pub2, err := GenKeyPair()
	if err != nil {
		t.Fatalf("GenKeyPair failed: %v", err)
	}
	shared1, err := DH(priv1, &pub2)
	if err != nil {
		t.Fatalf("DH failed: %v", err)
	}
	shared2, err := DH(priv2, &pub1)
	if err != nil {
		t.Fatalf("DH failed: %v", err)
	}
	if shared1 != shared2 {
		t.Fatal("DH shared secrets should be equal")
	}
	if shared1 == [32]byte{} {
		t.Fatal("Shared secret should not be zero")
	}
}

func TestKDF(t *testing.T) {
	data1 := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	data2 := [32]byte{33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64}
	result1 := KDF(data1)
	if result1 == [32]byte{} {
		t.Fatal("KDF result should not be zero")
	}
	result2 := KDF(data1, data2)
	if result2 == [32]byte{} {
		t.Fatal("KDF result should not be zero")
	}
	if result1 == result2 {
		t.Fatal("KDF results should be different for different inputs")
	}
	result3 := KDF(data1, data2)
	if result2 != result3 {
		t.Fatal("KDF should be deterministic")
	}
}

func TestEncodeDecode32(t *testing.T) {
	original := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	encoded := encode32(original)
	if len(encoded) != 64 {
		t.Fatal("Encoded string should be 64 characters")
	}
	decoded, err := decode32(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if decoded != original {
		t.Fatal("Decoded data should match original")
	}
}

func TestValidatePublicKey(t *testing.T) {
	validHex := "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"
	err := ValidatePublicKey(validHex)
	if err != nil {
		t.Fatalf("Valid key should not produce error: %v", err)
	}
	invalidHex := "invalid"
	err = ValidatePublicKey(invalidHex)
	if err == nil {
		t.Fatal("Invalid hex should produce error")
	}
	shortKey := "0102030405060708090a0b0c0d0e0f10"
	err = ValidatePublicKey(shortKey)
	if err == nil {
		t.Fatal("Short key should produce error")
	}
}

func TestGetKeyFingerprint(t *testing.T) {
	key := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	fingerprint := GetKeyFingerprint(key)
	expected := "01020304"
	if fingerprint != expected {
		t.Fatalf("Fingerprint mismatch: got %s, expected %s", fingerprint, expected)
	}
}

// Benchmark tests for performance on MPU devices
func BenchmarkGenKeyPair(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _, _ = GenKeyPair()
	}
}

func BenchmarkDH(b *testing.B) {
	priv1, pub1, _ := GenKeyPair()
	priv2, _ , _ := GenKeyPair()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = DH(priv1, &pub1)
		_, _ = DH(priv2, &pub1)
	}
}

func BenchmarkKDF(b *testing.B) {
	data1 := [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
	data2 := [32]byte{33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		KDF(data1, data2)
	}
}

// --- Additional edge/error case tests ---

func TestDH_Errors(t *testing.T) {
	_, pub, err := GenKeyPair()
	if err != nil {
		t.Fatalf("GenKeyPair failed: %v", err)
	}
	// Nil private key
	_, err = DH(nil, &pub)
	if err == nil {
		t.Fatal("DH should error with nil private key")
	}
}

func TestDH_InvalidPublicKey(t *testing.T) {
	priv, _, err := GenKeyPair()
	if err != nil {
		t.Fatalf("GenKeyPair failed: %v", err)
	}
	invalidPub := [32]byte{}
	_, err = DH(priv, &invalidPub)
	if err == nil {
		t.Fatal("DH should error with invalid public key")
	}
}

func TestDecode32_InvalidHex(t *testing.T) {
	_, err := decode32("nothex!!")
	if err == nil {
		t.Fatal("decode32 should error on invalid hex")
	}
}

func TestDecode32_WrongLength(t *testing.T) {
	short := "0102"
	_, err := decode32(short)
	if err == nil {
		t.Fatal("decode32 should error on short input")
	}
	long := "01" + "02" + "03" + "04" + "05" + "06" + "07" + "08" + "09" + "0a" + "0b" + "0c" + "0d" + "0e" + "0f" + "10" + "11" + "12" + "13" + "14" + "15" + "16" + "17" + "18" + "19" + "1a" + "1b" + "1c" + "1d" + "1e" + "1f" + "20" + "21"
	_, err = decode32(long)
	if err == nil {
		t.Fatal("decode32 should error on long input")
	}
}

func TestKDF_ZeroInputs(t *testing.T) {
	result := KDF()
	if result == [32]byte{} {
		t.Fatal("KDF with zero inputs should not return zero array")
	}
}

func TestKDF_ManyInputs(t *testing.T) {
	var inputs [10][32]byte
	for i := range inputs {
		for j := range inputs[i] {
			inputs[i][j] = byte(i + j)
		}
	}
	result := KDF(inputs[:]...)
	if result == [32]byte{} {
		t.Fatal("KDF with many inputs should not return zero array")
	}
} 