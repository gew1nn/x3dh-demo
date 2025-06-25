package main

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"

	"golang.org/x/crypto/chacha20poly1305"
	"x3dh-demo/internal/x3dh"
)

const serverURL = "http://localhost:8080"



// BobPrivateKeys holds the long-term private keys for Bob.
type BobPrivateKeys struct {
	IKbPriv  []byte `json:"ikb_priv"`
	SPKbPriv []byte `json:"spkb_priv"`
	EdPriv   []byte `json:"ed_priv"`
	OTKbPriv []byte `json:"otkb_priv"`
}

// --- Helper Functions ---

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

func loadEd25519PrivateKey(keyBytes []byte) ed25519.PrivateKey {
	if len(keyBytes) != ed25519.PrivateKeySize {
		panic("invalid Ed25519 private key size")
	}
	return ed25519.PrivateKey(keyBytes)
}

// --- Main Application Logic ---

func main() {
	action := flag.String("action", "check", "Action to perform: 'register' or 'check'")
	flag.Parse()

	switch *action {
	case "register":
		register()
	case "check":
		checkMessages()
	default:
		log.Fatalf("Invalid action: %s. Use 'register' or 'check'.", *action)
	}
}

func register() {
	// 1. Load or Generate Bob's keys
	_, err := os.ReadFile("bob_private_keys.json")
	if os.IsNotExist(err) {
		log.Println("Generating keys...")

		// Generate X25519 key pairs
		IKbPriv, IKbPub, err := x3dh.GenKeyPair()
		if err != nil {
			log.Fatalf("Failed to generate IKb key pair: %v", err)
		}
		SPKbPriv, SPKbPub, err := x3dh.GenKeyPair()
		if err != nil {
			log.Fatalf("Failed to generate SPKb key pair: %v", err)
		}
		OTKbPriv, OTKbPub, err := x3dh.GenKeyPair()
		if err != nil {
			log.Fatalf("Failed to generate OTKb key pair: %v", err)
		}

		// Generate Ed25519 key pair manually
		seed := make([]byte, ed25519.SeedSize)
		if _, err := rand.Read(seed); err != nil {
			log.Fatalf("Failed to generate Ed25519 seed: %v", err)
		}
		edPriv := ed25519.NewKeyFromSeed(seed)
		edPub := edPriv.Public().(ed25519.PublicKey)

		log.Println("Saving keys...")
		// Save all private keys
		keysToSave := BobPrivateKeys{
			IKbPriv:  IKbPriv.Bytes(),
			SPKbPriv: SPKbPriv.Bytes(),
			EdPriv:   []byte(edPriv), // Convert ed25519.PrivateKey to []byte
			OTKbPriv: OTKbPriv.Bytes(),
		}
		blob, _ := json.MarshalIndent(keysToSave, "", "  ")
		_ = os.WriteFile("bob_private_keys.json", blob, 0600)

		// Sign the SPK with the Ed25519 private key
		sig := ed25519.Sign(edPriv, SPKbPub[:])

		// Create the bundle to upload
		bundle := x3dh.Bundle{
			IK:      encode32(IKbPub),
			SPK:     encode32(SPKbPub),
			OTK:     encode32(OTKbPub),
			Ed25519: hex.EncodeToString(edPub), // edPub is the public key
			Sig:     hex.EncodeToString(sig),
		}

		log.Println("Registering with server...")
		// Upload bundle to server
		bundleJSON, _ := json.Marshal(bundle)
		resp, err := http.Post(serverURL+"/register/bob", "application/json", bytes.NewBuffer(bundleJSON))
		if err != nil {
			log.Fatalf("Failed to register with server: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Fatalf("Server returned an error during registration: %s - %s", resp.Status, string(body))
		}
		log.Println("Registration successful.")

	} else if err == nil {
		log.Fatal("Keys already exist. Registration should only happen once. To re-register, delete bob_private_keys.json")
	} else {
		panic(err)
	}
}

func checkMessages() {
	// 1. Load Bob's private keys. He can't decrypt without them.
	privateKeyBlob, err := os.ReadFile("bob_private_keys.json")
	if os.IsNotExist(err) {
		log.Fatal("Private keys not found. Please run with -action=register first.")
	} else if err != nil {
		panic(err)
	}

	var loadedKeys BobPrivateKeys
	if err := json.Unmarshal(privateKeyBlob, &loadedKeys); err != nil {
		panic(err)
	}
	curve := ecdh.X25519()
	IKbPriv, _ := curve.NewPrivateKey(loadedKeys.IKbPriv)
	SPKbPriv, _ := curve.NewPrivateKey(loadedKeys.SPKbPriv)
	OTKbPriv, _ := curve.NewPrivateKey(loadedKeys.OTKbPriv)

	// 2. Poll the server for new messages
	resp, err := http.Get(serverURL + "/messages/bob")
	if err != nil {
		log.Fatalf("Failed to check for messages: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Println("No new messages found.")
		return
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Server returned an error: %s - %s", resp.Status, string(body))
	}

	// 3. Decode the message from Alice (new response format)
	var respData struct {
		Message      x3dh.InitialMessage `json:"message"`
		MessagesLeft int            `json:"messages_left"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		log.Fatalf("Failed to decode message from server: %v", err)
	}
	msg := respData.Message

	log.Println("Received an initial message from Alice.")

	// 4. Derive the session key
	IKaPub32 := decode32(msg.AliceIK)
	EKaPub32 := decode32(msg.AliceEKa)

	// The order of these DH operations must match the order Alice uses
	// to ensure the KDF produces the same session key.
	// DH1 = DH(SPKb, IKa)
	DH1, err := x3dh.DH(SPKbPriv, &IKaPub32)
	if err != nil {
		log.Fatalf("DH1 failed: %v", err)
	}
	// DH2 = DH(IKb, EKa)
	DH2, err := x3dh.DH(IKbPriv, &EKaPub32)
	if err != nil {
		log.Fatalf("DH2 failed: %v", err)
	}
	// DH3 = DH(SPKb, EKa)
	DH3, err := x3dh.DH(SPKbPriv, &EKaPub32)
	if err != nil {
		log.Fatalf("DH3 failed: %v", err)
	}
	// DH4 = DH(OTKb, EKa)
	DH4, err := x3dh.DH(OTKbPriv, &EKaPub32)
	if err != nil {
		log.Fatalf("DH4 failed: %v", err)
	}

	master := x3dh.KDF(DH1, DH2, DH3, DH4)
	log.Println("Session key derived " + hex.EncodeToString(master[:]))

	// 5. Decrypt the message
	key := master[:]
	aead, _ := chacha20poly1305.New(key)
	nonce, _ := hex.DecodeString(msg.Nonce)
	ciphertext, _ := hex.DecodeString(msg.Ciphertext)
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		log.Fatalf("DECRYPTION FAILED: %v", err)
	}
	log.Println("Decrypted message from Alice:", string(plaintext))

	if respData.MessagesLeft > 0 {
		log.Printf("You still have %d messages left.", respData.MessagesLeft)
	}
}
