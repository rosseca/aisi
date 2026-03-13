.PHONY: build test test-coverage test-short clean install lint

BINARY_NAME := aisi
BUILD_DIR := build
VERSION := 1.0.0
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Build targets
build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/aisi

build-all: build-darwin build-linux

build-darwin:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/aisi
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/aisi

build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/aisi
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/aisi

# Test targets
test:
	go test ./... -v

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-short:
	go test ./... -short

# Development
lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

# Installation
install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) ~/go/bin/

# Cleanup
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build for current platform"
	@echo "  build-all     - Build for all platforms (darwin/linux, amd64/arm64)"
	@echo "  build-darwin  - Build for macOS (Intel + Apple Silicon)"
	@echo "  build-linux   - Build for Linux (amd64 + arm64)"
	@echo "  test          - Run all tests with verbose output"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  test-short    - Run only short tests"
	@echo "  lint          - Run golangci-lint"
	@echo "  fmt           - Format code"
	@echo "  tidy          - Run go mod tidy"
	@echo "  install       - Install binary to ~/go/bin"
	@echo "  clean         - Remove build artifacts"
