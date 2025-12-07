.PHONY: all build test lint clean tidy help vscode

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Default target
all: tidy lint test build

# Build all packages
build:
	go build -ldflags "$(LDFLAGS)" -o _output/bin/ ./cmd/...

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/...

# Run tests with coverage (uses ginkgo if available, otherwise go test)
test:
	@mkdir -p _output/tests
	@if command -v ginkgo >/dev/null 2>&1; then \
		ginkgo -v --cover --coverprofile=_output/tests/coverage.out ./...; \
	else \
		go test -v -coverprofile=_output/tests/coverage.out ./...; \
	fi

# Run linter (requires golangci-lint)
lint:
	golangci-lint run --fix ./...

# Tidy dependencies
tidy:
	go mod tidy

# Clean build artifacts
clean:
	rm -rf _output
	go clean ./...

# Generate VSCode extension configuration from devgen.toml files
vscode:
	go run ./cmd/vscgen
	@cd vscode-devgen && sed -i '' 's/"version": "[^"]*"/"version": "$(VERSION)"/' package.json
	cd vscode-devgen && npm run compile && npm run package

# Install development tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/onsi/ginkgo/v2/ginkgo@latest

# Help
help:
	@echo "Available targets:"
	@echo "  all    - Run lint, test, and build (default)"
	@echo "  build  - Build all packages"
	@echo "  test   - Run tests with coverage (ginkgo or go test)"
	@echo "  lint   - Run golangci-lint"
	@echo "  tidy   - Tidy dependencies"
	@echo "  clean  - Clean build artifacts"
	@echo "  vscode - Generate and build VSCode extension"
	@echo "  tools  - Install development tools"
