# Build stage: Compile Go code in an alpine environment
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Pre-copy go.mod and go.sum for efficient caching of dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Build statically-linked binary for Linux arm/amd64 architectures
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Run stage: Copy binary to a minimal scratch-like Alpine image
FROM alpine:3.19

# Set secure execution workspace
WORKDIR /app

# Copy compiled binary from builder stage
COPY --from=builder /app/main .

# Expose API default port
EXPOSE 8080

# Run binary
CMD ["./main"]
