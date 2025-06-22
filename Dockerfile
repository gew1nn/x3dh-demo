# Multi-stage Dockerfile for X3DH Protocol on MPU devices
# Optimized for Raspberry Pi and similar ARM-based devices

# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build for ARM64 (Raspberry Pi 4)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -installsuffix cgo -o alice ./cmd/alice
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -installsuffix cgo -o bob ./cmd/bob
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -a -installsuffix cgo -o server ./cmd/server

# Build for ARM32 (older Raspberry Pi models)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -a -installsuffix cgo -o alice-arm ./cmd/alice
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -a -installsuffix cgo -o bob-arm ./cmd/bob
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm go build -a -installsuffix cgo -o server-arm ./cmd/server

# Runtime stage - minimal Alpine image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
RUN addgroup -g 1001 -S x3dh && \
    adduser -u 1001 -S x3dh -G x3dh

# Create necessary directories
RUN mkdir -p /app/keys /app/logs && \
    chown -R x3dh:x3dh /app

# Switch to non-root user
USER x3dh

# Set working directory
WORKDIR /app

# Copy binaries from builder stage
COPY --from=builder --chown=x3dh:x3dh /app/alice /app/alice-arm /app/bob /app/bob-arm /app/server /app/server-arm ./

# Copy configuration and documentation
COPY --chown=x3dh:x3dh README.md ./

# Create default config
RUN echo '{"server_host":"localhost","server_port":8080,"client_timeout":"30s","retry_attempts":3,"retry_delay":"5s","key_rotation_days":30,"max_otk_count":100,"low_memory_mode":true,"enable_logging":true,"log_level":"info","key_store_path":"./keys/","log_file_path":"./logs/x3dh.log"}' > config.json

# Expose server port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Default command (can be overridden)
CMD ["./server"] 