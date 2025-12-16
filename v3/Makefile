# Variables
BINARY_NAME=prisma
BUILD_DIR=bin
GO=go
GOFLAGS=-v
LDFLAGS=-ldflags="-s -w"

.PHONY: all build test lint fmt clean install generate-mocks docs help

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/prisma

# Build for multiple platforms
build-release:
	@echo "Building for multiple platforms..."
	@GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/prisma
	@GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/prisma
	@GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/prisma
	@GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/prisma

# Run tests
test:
	@echo "Running unit tests..."
	@$(GO) test -v -race -coverprofile=coverage.out ./...

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	@$(GO) test -v -tags=integration ./test/integration/...

# Run E2E tests
test-e2e:
	@echo "Running E2E tests..."
	@$(GO) test -v ./test/e2e/...

# Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	@$(GO) test -bench=. -benchmem ./test/benchmark/...

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	@gofmt -s -w .
	@goimports -w .

# Install the binary
install:
	@echo "Installing $(BINARY_NAME)..."
	@$(GO) install ./cmd/prisma

# Generate mocks
generate-mocks:
	@echo "Generating mocks..."
	@go install github.com/golang/mock/mockgen@latest
	@mockgen -source=internal/core/schema/domain/interfaces.go -destination=test/mocks/schema_mock.go -package=mocks
	@mockgen -source=internal/core/migration/domain/interfaces.go -destination=test/mocks/migration_mock.go -package=mocks
	@mockgen -source=internal/repository/interface.go -destination=test/mocks/repository_mock.go -package=mocks

# Generate documentation
docs:
	@echo "Generating documentation..."
	@godoc -http=:6060

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out

# Show help
help:
	@echo "Available targets:"
	@echo "  build            - Build the binary"
	@echo "  build-release    - Build for multiple platforms"
	@echo "  test             - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-e2e         - Run E2E tests"
	@echo "  benchmark        - Run benchmarks"
	@echo "  lint             - Run linter"
	@echo "  fmt              - Format code"
	@echo "  install          - Install the binary"
	@echo "  generate-mocks   - Generate mocks"
	@echo "  docs             - Generate documentation"
	@echo "  clean            - Clean build artifacts"
	@echo "  help             - Show this help message"

.DEFAULT_GOAL := help
