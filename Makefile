# Makefile for X3DH Protocol - MPU Device Deployment
# Supports Raspberry Pi and similar ARM-based devices

.PHONY: help build build-arm64 build-arm32 clean test run-server run-alice run-bob docker-build docker-run docker-clean deploy-pi

# Default target
help:
	@echo "X3DH Protocol - MPU Device Deployment"
	@echo "====================================="
	@echo ""
	@echo "Available targets:"
	@echo "  build        - Build for current platform"
	@echo "  build-arm64  - Build for Raspberry Pi 4 (ARM64)"
	@echo "  build-arm32  - Build for older Pi models (ARM32)"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  run-server   - Start the X3DH server"
	@echo "  run-alice    - Run Alice client"
	@echo "  run-bob      - Run Bob client"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run with Docker Compose"
	@echo "  docker-clean - Clean Docker resources"
	@echo "  deploy-pi    - Deploy to Raspberry Pi"
	@echo ""

# Build targets
build:
	@echo "Building X3DH protocol for current platform..."
	go build -o bin/alice ./cmd/alice
	go build -o bin/bob ./cmd/bob
	go build -o bin/server ./cmd/server
	@echo "Build complete! Binaries in ./bin/"

build-arm64:
	@echo "Building X3DH protocol for Raspberry Pi 4 (ARM64)..."
	GOOS=linux GOARCH=arm64 go build -o bin/alice-arm64 ./cmd/alice
	GOOS=linux GOARCH=arm64 go build -o bin/bob-arm64 ./cmd/bob
	GOOS=linux GOARCH=arm64 go build -o bin/server-arm64 ./cmd/server
	@echo "ARM64 build complete! Binaries in ./bin/"

build-arm32:
	@echo "Building X3DH protocol for older Pi models (ARM32)..."
	GOOS=linux GOARCH=arm go build -o bin/alice-arm ./cmd/alice
	GOOS=linux GOARCH=arm go build -o bin/bob-arm ./cmd/bob
	GOOS=linux GOARCH=arm go build -o bin/server-arm ./cmd/server
	@echo "ARM32 build complete! Binaries in ./bin/"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f bob_private_keys.json
	@echo "Clean complete!"

# Test the implementation
test:
	@echo "Running tests..."
	go test ./...
	@echo "Tests complete!"

# Run targets
run-server:
	@echo "Starting X3DH server..."
	go run ./cmd/server

run-alice:
	@echo "Running Alice client..."
	go run ./cmd/alice

run-bob:
	@echo "Running Bob client..."
	go run ./cmd/bob

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker build -t x3dh-protocol:latest .
	@echo "Docker build complete!"

docker-run:
	@echo "Starting X3DH system with Docker Compose..."
	docker-compose up -d
	@echo "System started! Check logs with: docker-compose logs"

docker-clean:
	@echo "Cleaning Docker resources..."
	docker-compose down -v
	docker rmi x3dh-protocol:latest 2>/dev/null || true
	@echo "Docker cleanup complete!"

# Raspberry Pi deployment
deploy-pi:
	@echo "Deploying to Raspberry Pi..."
	@echo "1. Build ARM64 binaries..."
	$(MAKE) build-arm64
	@echo "2. Copy binaries to Pi (adjust PI_HOST as needed)..."
	@echo "   Example: scp bin/*-arm64 pi@raspberrypi.local:~/x3dh/"
	@echo "3. On Pi, run: ./server-arm64"
	@echo "4. In another terminal: ./bob-arm64 -action=register"
	@echo "5. Then: ./alice-arm64"

# Development helpers
dev-setup:
	@echo "Setting up development environment..."
	go mod download
	go mod tidy
	@echo "Development setup complete!"

# Performance testing
bench:
	@echo "Running performance benchmarks..."
	go test -bench=. ./internal/x3dh/
	@echo "Benchmarks complete!"

# Security audit
audit:
	@echo "Running security audit..."
	go list -json -deps . | nancy sleuth
	@echo "Security audit complete!"

# Create release package
release:
	@echo "Creating release package..."
	$(MAKE) build-arm64
	$(MAKE) build-arm32
	tar -czf x3dh-protocol-$(shell date +%Y%m%d).tar.gz \
		bin/ README.md docker-compose.yml Dockerfile \
		--exclude='*.json' --exclude='*.log'
	@echo "Release package created!"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go get -u golang.org/x/crypto/chacha20poly1305
	go mod tidy
	@echo "Dependencies installed!" 