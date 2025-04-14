FROM golang:1.24 AS base
WORKDIR /app
COPY . .
RUN apt-get update && apt-get install -y make

# Stage for building on current platform (used by dockerbuild)
FROM base AS builder-current
RUN go build -o /app/bin/gogoproduct ./cmd/myapp

# Stage for building all platforms (used by dockerall)
FROM base AS builder-all

# Create directories for all platforms
RUN mkdir -p /app/bin/windows/amd64 /app/bin/linux/amd64 /app/bin/linux/arm64 /app/bin/macos/arm64

# macOS ARM64 (primary platform)
RUN CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o /app/bin/macos/arm64/gogoproduct ./cmd/myapp

# Windows AMD64
RUN CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o /app/bin/windows/amd64/gogoproduct.exe ./cmd/myapp

# Linux AMD64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/bin/linux/amd64/gogoproduct ./cmd/myapp

# Linux ARM64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /app/bin/linux/arm64/gogoproduct ./cmd/myapp

# Stage for running tests (used by dockertest)
FROM base AS tester
CMD ["go", "test", "./..."]

# Stage for formatting code (used by dockerfmt)
FROM base AS formatter
CMD ["go", "fmt", "./..."]

# Stage for validation (used by dockervalidate)
FROM base AS validator
CMD ["make", "validate"]
