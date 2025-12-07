# enumgen

[中文](README.md) | English

`enumgen` is a Go enum code generation tool that generates helper methods for types annotated with `enumgen:@enum`.

## Installation

```bash
go install github.com/tlipoca9/devgen/cmd/enumgen@latest
```

## Usage

```bash
enumgen ./...              # All packages
enumgen ./pkg/status       # Specific package
```

## Annotations

### @enum - Mark Enum Type

Add annotation above type definition to specify methods to generate:

```go
// Status represents status
// enumgen:@enum(string, json, text, sql)
type Status int

const (
    StatusPending Status = iota
    StatusActive
    StatusInactive
)
```

Supported options:
- `string` - Generate `String()` method
- `json` - Generate `MarshalJSON()` / `UnmarshalJSON()` methods
- `text` - Generate `MarshalText()` / `UnmarshalText()` methods
- `sql` - Generate `Value()` (driver.Valuer) / `Scan()` (sql.Scanner) methods

### @enum.name - Custom Value Name

By default, enum value string names automatically strip the type name prefix (e.g., `StatusPending` → `Pending`).

Use `@enum.name` to customize names:

```go
// Level represents log level
// enumgen:@enum(string, json)
type Level int

const (
    // enumgen:@enum.name(DEBUG)
    LevelDebug Level = iota + 1
    // enumgen:@enum.name(INFO)
    LevelInfo
    // enumgen:@enum.name(WARN)
    LevelWarn
    // enumgen:@enum.name(ERROR)
    LevelError
)
```

**Note**: `@enum.name` values cannot be duplicated, otherwise an error will be reported.

## Generated Code

For a `Status` type annotated with `enumgen:@enum(string, json, text, sql)`, the following code is generated:

### Type Methods

Regardless of options selected, `IsValid()` method is always generated:

```go
// IsValid reports whether x is a valid Status.
func (x Status) IsValid() bool {
    return StatusEnums.Contains(x)
}
```

Methods generated based on annotation options:

**string option:**
```go
// String returns the string representation of Status.
func (x Status) String() string {
    return StatusEnums.Name(x)
}
```

**json option:**
```go
// MarshalJSON implements json.Marshaler.
func (x Status) MarshalJSON() ([]byte, error) {
    return json.Marshal(StatusEnums.Name(x))
}

// UnmarshalJSON implements json.Unmarshaler.
func (x *Status) UnmarshalJSON(data []byte) error {
    var s string
    if err := json.Unmarshal(data, &s); err != nil {
        return err
    }
    v, err := StatusEnums.Parse(s)
    if err != nil {
        return err
    }
    *x = v
    return nil
}
```

**text option:**
```go
// MarshalText implements encoding.TextMarshaler.
func (x Status) MarshalText() ([]byte, error) {
    return []byte(StatusEnums.Name(x)), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (x *Status) UnmarshalText(data []byte) error {
    v, err := StatusEnums.Parse(string(data))
    if err != nil {
        return err
    }
    *x = v
    return nil
}
```

**sql option:**
```go
// Value implements driver.Valuer.
func (x Status) Value() (driver.Value, error) {
    return StatusEnums.Name(x), nil
}

// Scan implements sql.Scanner.
func (x *Status) Scan(src any) error {
    if src == nil {
        return nil
    }
    var s string
    switch v := src.(type) {
    case string:
        s = v
    case []byte:
        s = string(v)
    default:
        return fmt.Errorf("cannot scan %T into Status", src)
    }
    v, err := StatusEnums.Parse(s)
    if err != nil {
        return err
    }
    *x = v
    return nil
}
```

### Helper Variable StatusEnums

Regardless of options selected, helper variable and type are generated:

```go
// StatusEnums is the enum helper for Status.
var StatusEnums = _StatusEnums{
    values: []Status{
        StatusPending,
        StatusActive,
        StatusInactive,
    },
    names: map[Status]string{
        StatusPending:  "Pending",
        StatusActive:   "Active",
        StatusInactive: "Inactive",
    },
    byName: map[string]Status{
        "Pending":  StatusPending,
        "Active":   StatusActive,
        "Inactive": StatusInactive,
    },
}

// _StatusEnums provides enum metadata and validation for Status.
type _StatusEnums struct {
    values []Status
    names  map[Status]string
    byName map[string]Status
}
```

### Helper Methods

| Method | Description |
|--------|-------------|
| `IsValid() bool` | Check if current value is valid (type method, always generated) |
| `List() []Status` | Return all valid enum values |
| `Contains(v Status) bool` | Check if value is valid |
| `ContainsName(name string) bool` | Check if name is valid |
| `Parse(s string) (Status, error)` | Parse enum value from string |
| `Name(v Status) string` | Get string name of enum value |
| `Names() []string` | Return all valid names |

## Complete Example

### Definition

```go
package order

// OrderStatus represents order status
// enumgen:@enum(string, json, sql)
type OrderStatus int

const (
    OrderStatusPending    OrderStatus = iota + 1 // Pending
    OrderStatusProcessing                        // Processing
    OrderStatusCompleted                         // Completed
    // enumgen:@enum.name(Cancelled)
    OrderStatusCanceled                          // Canceled (custom name)
)
```

Run code generation:

```bash
enumgen ./...
```

### Usage

```go
package main

import (
    "encoding/json"
    "fmt"
    
    "example.com/order"
)

func main() {
    status := order.OrderStatusPending
    
    // String
    fmt.Println(status.String()) // Output: Pending
    
    // JSON Marshal
    data, _ := json.Marshal(status)
    fmt.Println(string(data)) // Output: "Pending"
    
    // JSON Unmarshal
    var s order.OrderStatus
    json.Unmarshal([]byte(`"Completed"`), &s)
    fmt.Println(s) // Output: Completed
    
    // Parse string
    parsed, err := order.OrderStatusEnums.Parse("Processing")
    if err == nil {
        fmt.Println(parsed) // Output: Processing
    }
    
    // List all values
    for _, v := range order.OrderStatusEnums.List() {
        fmt.Printf("%d: %s\n", v, order.OrderStatusEnums.Name(v))
    }
    // Output:
    // 1: Pending
    // 2: Processing
    // 3: Completed
    // 4: Cancelled
    
    // Validation
    fmt.Println(order.OrderStatusEnums.Contains(order.OrderStatusPending)) // true
    fmt.Println(order.OrderStatusEnums.ContainsName("Invalid"))            // false
}
```

## Testing

### Run Unit Tests

```bash
# Run all tests
go test -v ./cmd/enumgen/generator/...

# Run tests with coverage
go test -v ./cmd/enumgen/generator/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

Current test coverage: **99.6%** (46 test cases)
