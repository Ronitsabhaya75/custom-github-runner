# ==========================================================
# Stage 1: Build static runner executable
# ==========================================================
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install compilation prerequisites
RUN apk add --no-cache git ca-certificates

# Copy dependency configuration
COPY go.mod go.sum ./
RUN go mod download

# Copy application source
COPY . .

# Compile optimized static binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o runner cmd/runner/main.go

# ==========================================================
# Stage 2: Package runner inside minimal runtime environment
# ==========================================================
FROM alpine:3.19

# Install base runtime tools (mirroring tools used by actions/runner)
RUN apk add --no-cache \
    bash \
    curl \
    git \
    openssh-client \
    ca-certificates \
    sudo \
    jq

# Create a non-root worker user (UID 1000)
RUN adduser -D -u 1000 runner && \
    echo "runner ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

WORKDIR /home/runner

# Copy native binary from builder stage
COPY --from=builder /app/runner /usr/local/bin/runner

# Adjust execution permissions
RUN chmod +x /usr/local/bin/runner

USER runner

ENTRYPOINT ["/usr/local/bin/runner"]
