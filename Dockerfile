# Stage 1: Build
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Disable Go modules (single-file build)
ENV GO111MODULE=off

# Copy the Go script
COPY cleaner.go .

# Build the binary
RUN go build -o cleanup cleaner.go

# Stage 2: Minimal runtime
FROM alpine:latest
WORKDIR /app

# Copy built binary and certificates (for https connections)
COPY --from=builder /app/cleanup .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Run the binary
ENTRYPOINT ["./cleanup"]