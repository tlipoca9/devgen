.PHONY: all build test lint clean tidy help vscode publish generate

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# VSCode extension version (strip 'v' prefix, use latest tag on HEAD or fallback to describe)
VSCODE_VERSION := $(shell tag=$$(git tag --points-at HEAD 2>/dev/null | sort -V | tail -1); if [ -n "$$tag" ]; then echo "$$tag"; else git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0"; fi | sed 's/^v//')

# Default target
all: tidy generate lint test build

# Build all packages
build:
	go build -ldflags "$(LDFLAGS)" -o _output/bin/ ./cmd/...

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/...

# Run tests with coverage (uses ginkgo if available, otherwise go test)
test:
	@mkdir -p _output/tests
	@if command -v ginkgo >/dev/null 2>&1; then \
		ginkgo -v --keep-going --cover --coverprofile=_output/tests/coverage.out ./...; \
	else \
		go test -v -coverprofile=_output/tests/coverage.out ./...; \
	fi

# Run linter (requires golangci-lint)
lint:
	golangci-lint run --fix ./...

# Tidy dependencies
tidy:
	go mod tidy

# Run code generation
generate:
	go run ./cmd/devgen --include-tests ./...

# Clean build artifacts
clean:
	rm -rf _output
	go clean ./...

# Build VSCode extension
vscode:
	@cd vscode-devgen && sed -i '' 's/"version": "[^"]*"/"version": "$(VSCODE_VERSION)"/' package.json
	cd vscode-devgen && npm run compile && npm run package

# Install VSCode extension to current IDE
# Auto-detects IDE from TERM_PROGRAM env var: vscode, codebuddy
# Usage: make vscode-install [IDE=code|buddycn]
vscode-install: vscode
	@IDE_CMD=""; \
	if [ -n "$(IDE)" ]; then \
		IDE_CMD="$(IDE)"; \
	elif [ "$$TERM_PROGRAM" = "vscode" ]; then \
		IDE_CMD="code"; \
	elif [ "$$TERM_PROGRAM" = "codebuddy" ]; then \
		IDE_CMD="buddycn"; \
	elif command -v buddycn >/dev/null 2>&1; then \
		IDE_CMD="buddycn"; \
	elif command -v code >/dev/null 2>&1; then \
		IDE_CMD="code"; \
	else \
		echo "Error: No IDE detected. Set IDE=code|buddycn"; exit 1; \
	fi; \
	echo "Installing to $$IDE_CMD..."; \
	cd vscode-devgen && $$IDE_CMD --install-extension devgen-$(VSCODE_VERSION).vsix

# Publish a new version: bump version, build vscode extension, amend commit, and re-tag
# Usage: make publish RELEASE_VERSION=0.1.4
publish:
	@if [ -z "$(RELEASE_VERSION)" ]; then echo "Usage: make publish RELEASE_VERSION=x.y.z"; exit 1; fi
	@echo "Publishing version $(RELEASE_VERSION)..."
	@# Update package.json version
	@cd vscode-devgen && sed -i '' 's/"version": "[^"]*"/"version": "$(RELEASE_VERSION)"/' package.json
	@# Build vscode extension
	cd vscode-devgen && npm run compile && npm run package
	@# Amend last commit with version bump
	git add vscode-devgen/package.json
	git commit --amend --no-edit
	@# Delete old tag if exists and create new tag
	-git tag -d v$(RELEASE_VERSION) 2>/dev/null || true
	git tag -a v$(RELEASE_VERSION) -m "Release v$(RELEASE_VERSION)"
	@echo "Done! Published v$(RELEASE_VERSION)"
	@echo "To push: git push origin main --tags --force-with-lease"

# Install development tools
tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/onsi/ginkgo/v2/ginkgo@latest

# Help
help:
	@echo "Available targets:"
	@echo "  all           - Run tidy, generate, lint, test, and build (default)"
	@echo "  build         - Build all packages"
	@echo "  generate      - Run code generation (go generate)"
	@echo "  test          - Run tests with coverage (ginkgo or go test)"
	@echo "  lint          - Run golangci-lint"
	@echo "  tidy          - Tidy dependencies"
	@echo "  clean         - Clean build artifacts"
	@echo "  vscode        - Build VSCode extension"
	@echo "  vscode-install- Build and install VSCode extension (auto-detects IDE)"
	@echo "  publish       - Publish new version (Usage: make publish RELEASE_VERSION=x.y.z)"
	@echo "  tools         - Install development tools"
