# X3DH Protocol Demonstration

This project provides a simple command-line demonstration of the **Triple Diffie-Hellman (X3DH)** key agreement protocol. It shows how two parties, Alice (the initiator) and Bob (the responder), can establish a shared secret key over an untrusted network, even if they are not online simultaneously.

The system consists of a central server that stores key bundles and messages, and two clients representing Alice and Bob.

## How It Works

The implementation follows the core X3DH flow:

1.  **Server**: A lightweight HTTP server backed by Redis that acts as a "post office." It stores public key bundles and forwards encrypted initial messages.
2.  **Bob (Responder)**: On first run, Bob generates long-term identity keys (`bob_private_keys.json`) and a set of public keys (IK, SPK, OTK). He signs his SPK with his Ed25519 key and uploads this entire public "bundle" to the server.
3.  **Alice (Initiator)**: On first run, Alice generates her own identity key (`alice_private_keys.json`). To send a message, she:
    *   Fetches Bob's public key bundle from the server.
    *   Verifies the signature on Bob's signed pre-key.
    *   Generates a new ephemeral key pair for the session.
    *   Performs the four X3DH Diffie-Hellman calculations to derive a shared secret.
    *   Uses the secret to encrypt her message and sends it to the server for Bob.
4.  **Bob (Message Retrieval)**: When Bob checks for messages, he fetches the encrypted data from the server, performs his side of the X3DH calculations to derive the *exact same shared secret*, and decrypts the message.

## Prerequisites

*   **Go** (version 1.20 or newer)
*   **Redis** (must be running on `localhost:6379`)

## How to Run

Open three separate terminal windows.

### Step 1: Start the Server

In your first terminal, start the server. It will connect to Redis and begin listening for requests.

```bash
go run ./cmd/server/main.go
```
> _Leave this terminal running._

### Step 2: Register Bob (One-Time Setup)

In your second terminal, act as Bob. The first time Bob uses the service, he must generate his keys and register his public bundle with the server. A `bob_private_keys.json` file will be created.

```bash
go run ./cmd/bob/main.go -action=register
```

> **Note:** If you want to re-register, you must first delete `bob_private_keys.json`.

### Step 3: Alice Sends a Message

In your third terminal, act as Alice. On first run, this will create an `alice_private_keys.json` file for her identity.

```bash
go run ./cmd/alice/main.go
```
The program will then fetch Bob's bundle, derive the shared key, and prompt you to enter a message. Type a message and press Enter.

### Step 4: Bob Checks His Messages

Return to Bob's terminal (the second one). Bob can now "come online" to check his mail.

```bash
go run ./cmd/bob/main.go -action=check
```
Bob will download the encrypted message, derive the shared key, and successfully decrypt the message from Alice.

## Project Structure

- `cmd/server/`: The central HTTP server.
- `cmd/alice/`: The command-line client for the initiator (Alice).
- `cmd/bob/`: The command-line client for the responder (Bob).
- `internal/x3dh/`: Contains the core cryptographic logic for the X3DH protocol and shared data types.

## **Target Use Cases**

- **IoT Devices**: Secure communication between sensors and controllers
- **Edge Computing**: Secure data exchange between edge nodes
- **Offline-First Applications**: Devices that need to communicate when connectivity is intermittent
- **Raspberry Pi Projects**: Secure messaging between Pi devices
- **Embedded Systems**: Lightweight secure communication protocols

## **Architecture**

The project consists of three independent components:

1.  **Server (`cmd/server`)**: A simple HTTP server that acts as a "post office." It has no knowledge of private keys and its only job is to store public key bundles and forward the initial encrypted message from one user to another.
2.  **Bob (`cmd/bob`)**: A command-line client that represents the "responder." He can `register` his public keys with the server and then `check` for any messages waiting for him.
3.  **Alice (`cmd/alice`)**: A command-line client that represents the "initiator." She fetches a user's key bundle from the server, computes a shared key, and sends an initial encrypted message to that user via the server.

## **Security Features**

- **X25519** for Diffie-Hellman key exchange (curve25519)
- **Ed25519** for signing and verifying Bob's Signed Pre-key, preventing tampering
- **ChaCha20-Poly1305** for secure, authenticated encryption of the initial message
- **Asynchronous Flow**: Correctly models the real-world use case of X3DH where parties are not required to be online simultaneously
- **Persistent Keys**: Bob's long-term identity keys are saved locally, simulating a real device
- **Simplified One-Time Pre-key (OTK) Management**: For this demo, Bob generates and registers a single OTK. In a full production system, he would upload a large batch of OTKs

## **X3DH Protocol Flow**

```
Alice (Initiator)                    Server                    Bob (Responder)
     |                                  |                           |
     | 1. Fetch Bob's bundle            |                           |
     |--------------------------------->|                           |
     |                                  |                           |
     | 2. Bundle with IK, SPK, OTK      |                           |
     |<---------------------------------|                           |
     |                                  |                           |
     | 3. Verify SPK signature          |                           |
     |                                  |                           |
     | 4. Generate ephemeral keys       |                           |
     |                                  |                           |
     | 5. Calculate shared secret       |                           |
     |                                  |                           |
     | 6. Encrypt message               |                           |
     |                                  |                           |
     | 7. Send initial message          |                           |
     |--------------------------------->|                           |
     |                                  |                           |
     |                                  | 8. Check for messages     |
     |                                  |<--------------------------|
     |                                  |                           |
     |                                  | 9. Initial message        |
     |                                  |-------------------------->|
     |                                  |                           |
     |                                  | 10. Calculate same secret |
     |                                  |                           |
     |                                  | 11. Decrypt message       |
```

## **How to Run the Demonstration**

Follow these steps in order across three separate terminal windows.

### Step 1: Start the Server

In your first terminal, start the central server. It will listen for requests from Alice and Bob.

```bash
go run ./cmd/server
```
_Leave this terminal running._

### Step 2: Register Bob (One-Time Setup)

In your second terminal, you will act as Bob. The first time Bob uses the service, he must register his keys with the server.

> **Note:** If you have run this before, delete Bob's old private key file first: `rm bob_private_keys.json`

```bash
go run ./cmd/bob -action=register
```
The server will now have Bob's public keys. Bob can now go "offline."

### Step 3: Alice Sends a Message

In your third terminal, you will act as Alice. She will initiate the conversation.

```bash
go run ./cmd/alice
```
The program will:
1.  Fetch Bob's key bundle from the server.
2.  Verify the signature on his keys.
3.  Derive a shared session key.
4.  Prompt you to enter a message.

Type a message and press Enter. Alice will encrypt it and send it to the server to hold for Bob.

### Step 4: Bob Checks His Messages

Now, back in your second terminal (Bob's), Bob "comes online" to check his mail.

```bash
go run ./cmd/bob -action=check
```
Bob will contact the server, download the encrypted message Alice left, derive the **exact same session key**, and successfully decrypt your message.

You have now completed a full, asynchronous, and secure key exchange! 

## **Building for MPU Devices**

### Cross-compilation for Raspberry Pi (ARM64)

```bash
# For Raspberry Pi 4 (ARM64)
GOOS=linux GOARCH=arm64 go build -o alice-arm64 ./cmd/alice
GOOS=linux GOARCH=arm64 go build -o bob-arm64 ./cmd/bob
GOOS=linux GOARCH=arm64 go build -o server-arm64 ./cmd/server
```

### Cross-compilation for ARM32 (older Pi models)

```bash
# For Raspberry Pi 3 and earlier (ARM32)
GOOS=linux GOARCH=arm go build -o alice-arm ./cmd/alice
GOOS=linux GOARCH=arm go build -o bob-arm ./cmd/bob
GOOS=linux GOARCH=arm go build -o server-arm ./cmd/server
```

## **Performance Considerations for MPU Devices**

- **Memory Usage**: ~2-5MB per client (very lightweight)
- **CPU Usage**: Minimal during idle, spikes during key generation
- **Network**: Only requires HTTP/HTTPS connectivity
- **Storage**: ~1KB for private keys per device

## **Future Enhancements**

- [ ] **Multiple OTK Support**: Batch generation and management of one-time keys
- [ ] **Ratcheting**: Implement Double Ratchet for ongoing conversations
- [ ] **Persistence**: Database storage for server instead of in-memory
- [ ] **TLS**: Add HTTPS support for production use
- [ ] **Device Management**: Web interface for managing multiple devices
- [ ] **Metrics**: Performance and security metrics collection 