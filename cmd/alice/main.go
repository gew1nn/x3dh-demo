package main

import (
	"bufio"
	"bytes"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/crypto/chacha20poly1305"
	"x3dh-demo/internal/x3dh"
)

const serverURL = "http://localhost:8080"

// Bundle is the same JSON format Bob uses
// type Bundle struct {
// 	IK      string `json:"ik"`
// 	SPK     string `json:"spk"`
// 	OTK     string `json:"otk"`
// 	Ed25519 string `json:"ed25519"`
// 	Sig     string `json:"sig"`
// }

// AlicePrivateKeys holds the long-term private key for Alice.
type AlicePrivateKeys struct {
	IKaPriv []byte `json:"ika_priv"`
}

// ──────────────────────────────────────────────────────────────
// Вспомогательные функции
// ──────────────────────────────────────────────────────────────

// encode32(pub) → hex-string
func encode32(pk [32]byte) string { return hex.EncodeToString(pk[:]) }

// decode32(hex) → [32]byte
func decode32(s string) (out [32]byte) {
	raw, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	copy(out[:], raw)
	return
}

func main() {
	// 1. Load or generate Alice's Identity Key
	var alicePriv *ecdh.PrivateKey
	var alicePub [32]byte

	keyFile := "alice_private_keys.json"
	privateKeyBlob, err := os.ReadFile(keyFile)
	if os.IsNotExist(err) {
		log.Println("Generating Alice's identity key...")
		priv, pub, err := x3dh.GenKeyPair()
		if err != nil {
			log.Fatalf("Failed to generate Alice's key pair: %v", err)
		}
		alicePriv = priv
		alicePub = pub

		// Save the private key
		keysToSave := AlicePrivateKeys{IKaPriv: alicePriv.Bytes()}
		blob, _ := json.MarshalIndent(keysToSave, "", "  ")
		if err := os.WriteFile(keyFile, blob, 0600); err != nil {
			log.Fatalf("Failed to save Alice's private key: %v", err)
		}
		log.Printf("Alice's identity key saved to %s", keyFile)
	} else if err != nil {
		log.Fatalf("Failed to read %s: %v", keyFile, err)
	} else {
		// Load the private key
		var loadedKeys AlicePrivateKeys
		if err := json.Unmarshal(privateKeyBlob, &loadedKeys); err != nil {
			log.Fatalf("Failed to unmarshal Alice's private key: %v", err)
		}
		curve := ecdh.X25519()
		priv, err := curve.NewPrivateKey(loadedKeys.IKaPriv)
		if err != nil {
			log.Fatalf("Failed to load Alice's private key: %v", err)
		}
		alicePriv = priv
		pubBytes := alicePriv.PublicKey().Bytes()
		copy(alicePub[:], pubBytes)
		log.Println("Loaded Alice's identity key.")
	}

	// 2. Fetch Bob's bundle from the server
	resp, err := http.Get(serverURL + "/bundle/bob")
	if err != nil {
		log.Fatalf("Failed to fetch Bob's bundle: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Server returned an error for bundle request: %s - %s", resp.Status, string(body))
	}

	var bobBundle x3dh.Bundle
	if err := json.NewDecoder(resp.Body).Decode(&bobBundle); err != nil {
		log.Fatalf("Failed to decode Bob's bundle: %v", err)
	}

	// 3. Verify the signature on Bob's Signed Pre-key
	edPub, _ := hex.DecodeString(bobBundle.Ed25519)
	sig, _ := hex.DecodeString(bobBundle.Sig)
	spkBytes := decode32(bobBundle.SPK)
	if !ed25519.Verify(edPub, spkBytes[:], sig) {
		log.Fatal("SPK signature verification failed!")
	}

	// 4. Generate Alice's EPHEMERAL keys
	privEKa, pubEKa, err := x3dh.GenKeyPair()
	if err != nil {
		log.Fatalf("Failed to generate ephemeral key pair: %v", err)
	}

	// 5. Decode Bob's public keys
	IKbPub32 := decode32(bobBundle.IK)
	SPKbPub32 := decode32(bobBundle.SPK)
	OTKbPub32 := decode32(bobBundle.OTK)

	// 6. Calculate the shared secret (X3DH)
	// DH1 = DH(IKa, SPKb)
	DH1, err := x3dh.DH(alicePriv, &SPKbPub32)
	if err != nil {
		log.Fatalf("DH1 failed: %v", err)
	}
	// DH2 = DH(EKa, IKb)
	DH2, err := x3dh.DH(privEKa, &IKbPub32)
	if err != nil {
		log.Fatalf("DH2 failed: %v", err)
	}
	// DH3 = DH(EKa, SPKb)
	DH3, err := x3dh.DH(privEKa, &SPKbPub32)
	if err != nil {
		log.Fatalf("DH3 failed: %v", err)
	}
	// DH4 = DH(EKa, OTKb)
	DH4, err := x3dh.DH(privEKa, &OTKbPub32)
	if err != nil {
		log.Fatalf("DH4 failed: %v", err)
	}

	master := x3dh.KDF(DH1, DH2, DH3, DH4)
	log.Println("Session key derived " + hex.EncodeToString(master[:]))

	// 7. Get message from user and encrypt it
	log.Print("Enter a message to send to Bob: ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	plaintext := []byte(strings.TrimSpace(input))

	key := master[:]
	aead, _ := chacha20poly1305.New(key)
	nonce := make([]byte, aead.NonceSize())
	rand.Read(nonce)
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	// 8. Send the initial message to the server for Bob
	initialMessage := x3dh.InitialMessage{
		AliceIK:    encode32(alicePub),
		AliceEKa:   encode32(pubEKa),
		Nonce:      hex.EncodeToString(nonce),
		Ciphertext: hex.EncodeToString(ciphertext),
		Sender:     "alice",
	}

	msgJSON, _ := json.Marshal(initialMessage)
	resp, err = http.Post(serverURL+"/send/bob", "application/json", bytes.NewBuffer(msgJSON))
	if err != nil {
		log.Fatalf("Failed to send message to server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Server returned an error during send: %s - %s", resp.Status, string(body))
	}

	log.Println("Encrypted message was sent")
}

