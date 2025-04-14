FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .

RUN apk add --no-cache make

# Build for different platforms
RUN mkdir -p /app/bin/windows/amd64 /app/bin/linux/amd64 /app/bin/macos/arm64

# Windows AMD64
RUN CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o /app/bin/windows/amd64/myapp.exe ./cmd/myapp

# Linux AMD64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bin/linux/amd64/myapp ./cmd/myapp

# macOS ARM64
RUN CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o /app/bin/macos/arm64/myapp ./cmd/myapp
