.PHONY: all build test clean fmt lint deps verify help test-coverage

# Default target
all: fmt lint test build

# Build the binary
build:
	go build -o cctop .

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -f cctop
	rm -f coverage.out coverage.html

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Update dependencies
deps:
	go mod download
	go mod tidy

# Verify dependencies
verify:
	go mod verify

# Show help
help:
	@echo "Available targets:"
	@echo "  make all          - Format, lint, test, and build (default)"
	@echo "  make build        - Build the binary"
	@echo "  make test         - Run tests"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make clean        - Remove build artifacts"
	@echo "  make fmt          - Format Go code"
	@echo "  make lint         - Run linter (requires golangci-lint)"
	@echo "  make deps         - Download and tidy dependencies"
	@echo "  make verify       - Verify dependencies"
	@echo "  make help         - Show this help message"