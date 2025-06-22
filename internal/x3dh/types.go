package x3dh

type Bundle struct {
	IK      string `json:"ik"`
	SPK     string `json:"spk"`
	OTK     string `json:"otk"`
	Ed25519 string `json:"ed25519"`
	Sig     string `json:"sig"`
}

type InitialMessage struct {
	AliceIK    string `json:"alice_ik"`
	AliceEKa   string `json:"alice_eka"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
	Sender     string `json:"sender"`
} 