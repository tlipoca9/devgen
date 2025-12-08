# devgen

[中文](README.md) | English

A Go code generation toolkit that automatically generates boilerplate code through annotations, reducing repetitive manual coding.

## Installation

```bash
# Install devgen (includes all tools)
go install github.com/tlipoca9/devgen/cmd/devgen@latest

# Or install individually
go install github.com/tlipoca9/devgen/cmd/enumgen@latest
go install github.com/tlipoca9/devgen/cmd/validategen@latest
```

## Usage

```bash
devgen ./...                    # Run all generators
devgen --dry-run ./...          # Validate annotations without writing files
devgen --dry-run --json ./...   # JSON format output for IDE integration
enumgen ./...                   # Run enum generator only
validategen ./...               # Run validation generator only
```

## Tools

### enumgen - Enum Code Generator

Automatically generates serialization, deserialization, and validation methods for Go enum types.

```go
// Status represents status
// enumgen:@enum(string, json, sql)
type Status int

const (
    StatusPending Status = iota + 1
    StatusActive
    // enumgen:@name(Cancelled)
    StatusCanceled  // Custom name
)
```

**Supported Options**:
- `string` - Generate `String()` method
- `json` - Generate `MarshalJSON()` / `UnmarshalJSON()`
- `text` - Generate `MarshalText()` / `UnmarshalText()`
- `sql` - Generate `Value()` / `Scan()` for database operations

**Generated Helper Methods**:
- `IsValid()` - Check if enum value is valid
- `{Type}Enums.List()` - Return all valid enum values
- `{Type}Enums.Parse(s)` - Parse enum from string
- `{Type}Enums.Name(v)` - Get string name of enum value

See [enumgen README](cmd/enumgen/README_EN.md) for details.

---

### validategen - Validation Code Generator

Automatically generates `Validate()` methods for Go structs.

```go
// User model
// validategen:@validate
type User struct {
    // validategen:@required
    // validategen:@min(2)
    // validategen:@max(50)
    Name string

    // validategen:@required
    // validategen:@email
    Email string

    // validategen:@gte(0)
    // validategen:@lte(150)
    Age int

    // validategen:@oneof(admin, user, guest)
    Role string
}
```

**Validation Annotations**:

| Category | Annotations |
|----------|-------------|
| Required | `@required` |
| Range | `@min(n)` `@max(n)` `@len(n)` `@gt(n)` `@gte(n)` `@lt(n)` `@lte(n)` |
| Equality | `@eq(v)` `@ne(v)` `@oneof(a, b, c)` |
| Format | `@email` `@url` `@uuid` `@ip` `@ipv4` `@ipv6` |
| Character | `@alpha` `@alphanum` `@numeric` |
| String | `@contains(s)` `@excludes(s)` `@startswith(s)` `@endswith(s)` |
| Regex | `@regex(pattern)` |
| Data Format | `@format(json\|yaml\|toml\|csv)` |
| Nested | `@method(MethodName)` |

**Advanced Features**:
- `postValidate(errs []string) error` hook for custom validation logic

See [validategen README](cmd/validategen/README_EN.md) for details.

---

### vscode-devgen - VSCode Extension

Provides editor support for devgen annotations: syntax highlighting, auto-completion, parameter validation hints.

[![VS Marketplace](https://img.shields.io/visual-studio-marketplace/v/tlipoca9.devgen)](https://marketplace.visualstudio.com/items?itemName=tlipoca9.devgen)

Install from [VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=tlipoca9.devgen).

---

### Plugin System

devgen supports extending functionality through a plugin mechanism, allowing users to develop custom code generation tools using the genkit framework.

Two plugin types are supported:
- **source** - Go source code, compiled at runtime (recommended)
- **plugin** - Pre-compiled Go plugin (.so)

Plugins can implement the `ConfigurableTool` interface for self-describing configuration, and the VSCode extension will automatically retrieve annotation metadata via `devgen config --json`.

See [Plugin Development Guide](docs/plugin_EN.md) and [Examples](examples/plugin/).

## Build

```bash
make build    # Build all tools
make test     # Run tests
make vscode   # Build VSCode extension
```

## Release Notes

- [v0.2.2](docs/release/v0.2.2_EN.md) - 2025-12-08
- [v0.2.1](docs/release/v0.2.1_EN.md) - 2025-12-07
- [v0.2.0](docs/release/v0.2.0_EN.md) - 2025-12-07
- [v0.1.3](docs/release/v0.1.3_EN.md) - 2025-12-07
- [v0.1.2](docs/release/v0.1.2_EN.md) - 2025-12-07
- [v0.1.1](docs/release/v0.1.1_EN.md) - 2025-12-07
- [v0.1.0](docs/release/v0.1.0_EN.md) - 2025-12-07

## License

MIT
