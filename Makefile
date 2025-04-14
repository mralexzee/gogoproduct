.PHONY: help check build run clean test deps fmt format validate dockerbuild dockerall dockertest dockerfmt dockervalidate

# Variables
APP_NAME := gogoproduct
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
	@echo "  dockerbuild   - Build binary for current platform using Docker"
	@echo "  dockerall     - Build binaries for multiple platforms using Docker"
	@echo "  dockertest    - Run tests using Docker"
	@echo "  dockerfmt     - Format Go code using Docker"
	@echo "  dockervalidate - Run fmt, test, and build inside Docker"

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

# Build binary for current platform using Docker
dockerbuild:
	@echo "Building $(APP_NAME) for current platform using Docker..."
	docker build --target builder-current -t $(APP_NAME)-builder-current .
	docker create --name $(APP_NAME)-temp-current $(APP_NAME)-builder-current
	mkdir -p $(BUILD_DIR)
	docker cp $(APP_NAME)-temp-current:/app/bin/$(APP_NAME) $(BUILD_DIR)/
	docker rm $(APP_NAME)-temp-current
	@echo "Docker build complete: $(BUILD_DIR)/$(APP_NAME)"

# Build cross-platform binaries using Docker
dockerall:
	@echo "Building cross-platform binaries using Docker..."
	docker build --target builder-all -t $(APP_NAME)-builder-all .
	docker create --name $(APP_NAME)-temp-all $(APP_NAME)-builder-all
	rm -rf $(BUILD_DIR)/windows $(BUILD_DIR)/linux $(BUILD_DIR)/macos
	mkdir -p $(BUILD_DIR)/windows/amd64 $(BUILD_DIR)/linux/amd64 $(BUILD_DIR)/linux/arm64 $(BUILD_DIR)/macos/arm64
	docker cp $(APP_NAME)-temp-all:/app/bin/windows/amd64/$(APP_NAME).exe $(BUILD_DIR)/windows/amd64/
	docker cp $(APP_NAME)-temp-all:/app/bin/linux/amd64/$(APP_NAME) $(BUILD_DIR)/linux/amd64/
	docker cp $(APP_NAME)-temp-all:/app/bin/linux/arm64/$(APP_NAME) $(BUILD_DIR)/linux/arm64/
	docker cp $(APP_NAME)-temp-all:/app/bin/macos/arm64/$(APP_NAME) $(BUILD_DIR)/macos/arm64/
	docker rm $(APP_NAME)-temp-all
	@echo "Cross-platform builds complete:"
	@echo "  - macOS (ARM64): $(BUILD_DIR)/macos/arm64/$(APP_NAME) (primary platform)"
	@echo "  - Windows (AMD64): $(BUILD_DIR)/windows/amd64/$(APP_NAME).exe"
	@echo "  - Linux (AMD64): $(BUILD_DIR)/linux/amd64/$(APP_NAME)"
	@echo "  - Linux (ARM64): $(BUILD_DIR)/linux/arm64/$(APP_NAME)"

# Run tests using Docker
dockertest:
	@echo "Running tests using Docker..."
	docker build --target tester -t $(APP_NAME)-tester .
	docker run --rm $(APP_NAME)-tester
	@echo "Docker tests complete."

# Format Go code using Docker
dockerfmt:
	@echo "Formatting Go code using Docker..."
	docker build --target formatter -t $(APP_NAME)-formatter .
	docker run --rm -v $(PWD):/app $(APP_NAME)-formatter
	@echo "Docker formatting complete."

# Run fmt, test, and build inside Docker to validate code
dockervalidate:
	@echo "Validating code using Docker..."
	docker build --target validator -t $(APP_NAME)-validator .
	docker run --rm $(APP_NAME)-validator
	@echo "Docker validation complete."

# Format Go code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...
	@echo "Formatting complete."

# Alias for fmt
format: fmt

# Format, run tests, run build to validate code
validate: fmt test build
	@echo "Validation complete."
