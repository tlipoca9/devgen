# golangcilint

[中文](README.md) | English

`golangcilint` is a tool that integrates golangci-lint with devgen for IDE diagnostics integration.

## Overview

This tool checks if golangci-lint is configured and installed, then runs it and converts the output to devgen diagnostics format for IDE integration.

**Features:**
- Validation only, no code generation
- Auto-detects golangci-lint config files
- Supports golangci-lint v1 and v2 versions
- Outputs standard devgen diagnostics format

## Installation

```bash
go install github.com/tlipoca9/devgen/cmd/golangcilint@latest
```

**Prerequisites:**
- [golangci-lint](https://golangci-lint.run/usage/install/) must be installed

## Usage

```bash
golangcilint ./...              # All packages
golangcilint ./pkg/models       # Specific package
```

## How It Works

### Auto-Enable Conditions

golangcilint is automatically enabled when:

1. A golangci-lint config file exists in the project root:
   - `.golangci.yml`
   - `.golangci.yaml`
   - `.golangci.toml`
   - `.golangci.json`

2. The `golangci-lint` command is installed on the system

### Execution Flow

1. Find project root directory from loaded packages
2. Check if golangci-lint config file exists
3. Check if golangci-lint is installed
4. Run `golangci-lint run --output.json.path stdout ./...` (v2) or `golangci-lint run --out-format json ./...` (v1)
5. Parse JSON output and convert to devgen diagnostics format

### Diagnostic Output

Each diagnostic contains:
- `Severity` - Severity level (error/warning)
- `Message` - Issue description
- `File` - File path
- `Line` / `Column` - Position information
- `Tool` - Tool name (golangcilint)
- `Code` - Source linter name (e.g., gofmt, govet, etc.)

## Integration with devgen

golangcilint is a built-in tool in devgen and runs automatically during `devgen --dry-run`:

```bash
# Run all validations (including golangci-lint)
devgen --dry-run ./...

# JSON format output for IDE integration
devgen --dry-run --json ./...
```

## VSCode Integration

The VSCode extension automatically runs golangcilint at:

1. **On startup** - Global dry-run validation
2. **On file save** - Single file validation

Diagnostics are displayed in VSCode's Problems panel.

## Example Output

```bash
$ golangcilint ./...
⚠ Found 3 issue(s)
⚠ [gofmt] cmd/example/main.go:10:1: File is not `gofmt`-ed
⚠ [govet] cmd/example/main.go:15:2: printf: fmt.Printf format %d has arg str of wrong type string
⚠ [unused] cmd/example/main.go:20:6: func `unusedFunc` is unused
```

## Configuration

golangcilint itself has no configuration options. It directly uses the project's golangci-lint config file.

For golangci-lint configuration, refer to the [golangci-lint official documentation](https://golangci-lint.run/usage/configuration/).
