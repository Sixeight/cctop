.PHONY: all build test clean fmt lint deps verify help test-coverage fmt-check lint-check

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

# Format code with gofumpt (stricter gofmt)
fmt:
	@go run mvdan.cc/gofumpt -l -w .
	@go run golang.org/x/tools/cmd/goimports -w -local github.com/Sixeight/cctop .

# Run linter with latest golangci-lint
lint:
	@go run github.com/golangci/golangci-lint/cmd/golangci-lint run --fix

# Check code formatting (for CI)
fmt-check:
	@test -z "$$(go run mvdan.cc/gofumpt -l .)" || (echo "Please run 'make fmt' to format code"; exit 1)

# Run quick linting (for CI)
lint-check:
	@go run github.com/golangci/golangci-lint/cmd/golangci-lint run --tests=false --disable-all --enable=errcheck,govet,staticcheck

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
	@echo "  make fmt          - Format code with gofumpt and goimports"
	@echo "  make fmt-check    - Check code formatting (for CI)"
	@echo "  make lint         - Run comprehensive linting with auto-fix"
	@echo "  make lint-check   - Run quick linting checks (for CI)"
	@echo "  make deps         - Download and tidy dependencies"
	@echo "  make verify       - Verify dependencies"
	@echo "  make help         - Show this help message"
