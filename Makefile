.PHONY: help check build run clean test deps docker fmt format

# Variables
APP_NAME := myapp
BUILD_DIR := bin
MAIN_PATH := ./cmd/myapp

# Default target
help:
	@echo "Available make targets:"
	@echo "  help    - Show this help message"
	@echo "  check   - Check if required dependencies are installed"
	@echo "  deps    - Install dependencies"
	@echo "  build   - Build for current platform"
	@echo "  run     - Run the application"
	@echo "  clean   - Clean up build artifacts"
	@echo "  test    - Run tests"
	@echo "  fmt     - Format Go code"
	@echo "  format  - Alias for fmt"
	@echo "  docker  - Build cross-platform binaries using Docker"

# Check if all required dependencies are installed
check:
	@echo "Checking required dependencies..."
	@echo "Go:"
	@go version || (echo "Go is not installed" && exit 1)
	@echo "Docker (for cross-compilation):"
	@docker --version || echo "Docker is not installed (needed only for cross-compilation)"
	@echo "All dependencies checked."

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	@echo "Dependencies installed."

# Build for current platform
build:
	@echo "Building $(APP_NAME) for current platform..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

# Run the application
run: build
	@echo "Running $(APP_NAME)..."
	./$(BUILD_DIR)/$(APP_NAME)

# Clean up build artifacts
clean:
	@echo "Cleaning up..."
	rm -rf $(BUILD_DIR)
	@echo "Clean complete."

# Run tests
test:
	@echo "Running tests..."
	go test ./...
	@echo "Tests complete."

# Build cross-platform binaries using Docker
docker:
	@echo "Building cross-platform binaries using Docker..."
	docker build -t $(APP_NAME)-builder .
	docker create --name $(APP_NAME)-temp $(APP_NAME)-builder
	rm -rf $(BUILD_DIR)/windows $(BUILD_DIR)/linux $(BUILD_DIR)/macos
	mkdir -p $(BUILD_DIR)/windows/amd64 $(BUILD_DIR)/linux/amd64 $(BUILD_DIR)/macos/arm64
	docker cp $(APP_NAME)-temp:/app/bin/windows/amd64/myapp.exe $(BUILD_DIR)/windows/amd64/
	docker cp $(APP_NAME)-temp:/app/bin/linux/amd64/myapp $(BUILD_DIR)/linux/amd64/
	docker cp $(APP_NAME)-temp:/app/bin/macos/arm64/myapp $(BUILD_DIR)/macos/arm64/
	docker rm $(APP_NAME)-temp
	@echo "Cross-platform builds complete:"
	@echo "  - Windows (AMD64): $(BUILD_DIR)/windows/amd64/myapp.exe"
	@echo "  - Linux (AMD64): $(BUILD_DIR)/linux/amd64/myapp"
	@echo "  - macOS (ARM64): $(BUILD_DIR)/macos/arm64/myapp"

# Format Go code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...
	@echo "Formatting complete."

# Alias for fmt
format: fmt
