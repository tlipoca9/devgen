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

### @name - Custom Value Name

By default, enum value string names automatically strip the type name prefix (e.g., `StatusPending` → `Pending`).

Use `@name` to customize names:

```go
// Level represents log level
// enumgen:@enum(string, json)
type Level int

const (
    // enumgen:@name(DEBUG)
    LevelDebug Level = iota + 1
    // enumgen:@name(INFO)
    LevelInfo
    // enumgen:@name(WARN)
    LevelWarn
    // enumgen:@name(ERROR)
    LevelError
)
```

**Note**: `@name` values cannot be duplicated, otherwise an error will be reported.

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
    // enumgen:@name(Cancelled)
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

## Testing and Benchmarks

### Run Unit Tests

```bash
# Run all tests
go test -v ./cmd/enumgen/generator/...

# Run tests with coverage
go test -v ./cmd/enumgen/generator/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

Current test coverage: **99.6%** (46 test cases)

### Run Benchmarks

Benchmarks are implemented using [Ginkgo gmeasure](https://onsi.github.io/ginkgo/#benchmarking-code) to obtain statistically meaningful performance data.

#### Step 1: Run Benchmarks

```bash
# Run tests (including benchmarks)
cd /path/to/devgen
go test -v ./cmd/enumgen/generator/... -count=1
```

#### Step 2: Generate Report Files

```bash
# Generate JSON format report
ginkgo --json-report=benchmark_report.json ./cmd/enumgen/generator/...

# Generate JUnit XML format report
ginkgo --junit-report=benchmark_report.xml ./cmd/enumgen/generator/...

# Generate both formats
ginkgo --json-report=benchmark_report.json --junit-report=benchmark_report.xml ./cmd/enumgen/generator/...
```

#### Step 3: View Detailed Output

Running tests will output detailed benchmark results to the console, including:
- Mean
- StdDev (Standard Deviation)
- Min/Max
- Sample count and iteration count

### Benchmark Results Summary (2025-12-07)

**Test Environment**:
- OS: darwin (macOS)
- Arch: arm64
- CPU: Apple M4 Pro
- 1000 iterations per method, 100 samples

**Test Configuration**:
- RandomSeed: 1765086644
- TotalSpecs: 70
- SuiteSucceeded: true
- RunTime: ~290ms

| Method | Mean (1000x) | Per Op ~= | Description |
|--------|--------------|-----------|-------------|
| IsValid/valid | 1.54µs | 1.5ns | Validate valid enum value |
| IsValid/invalid | 2.77µs | 2.8ns | Validate invalid enum value |
| String/valid | 2.26µs | 2.3ns | Valid value to string |
| String/invalid | 69.87µs | 70ns | Invalid value to string (requires formatting) |
| MarshalJSON/direct | 53.66µs | 54ns | Direct MarshalJSON call |
| MarshalJSON/via_json_Marshal | 124.24µs | 124ns | Via json.Marshal |
| UnmarshalJSON/direct | 95.93µs | 96ns | Direct UnmarshalJSON call |
| UnmarshalJSON/via_json_Unmarshal | 157.38µs | 157ns | Via json.Unmarshal |
| MarshalText | 15.32µs | 15ns | Text serialization |
| UnmarshalText | 15.15µs | 15ns | Text deserialization |
| Value | 11.61µs | 12ns | SQL driver.Valuer |
| Scan/string | 7.61µs | 7.6ns | SQL Scanner (string) |
| Scan/bytes | 30.09µs | 30ns | SQL Scanner ([]byte) |
| Scan/nil | 1.11µs | 1.1ns | SQL Scanner (nil) |
| Parse/valid | 5.29µs | 5.3ns | Parse valid string |
| Parse/invalid | 95.01µs | 95ns | Parse invalid string (requires error creation) |
| Contains/valid | 1.80µs | 1.8ns | Check valid value |
| Contains/invalid | 2.57µs | 2.6ns | Check invalid value |
| ContainsName/valid | 5.36µs | 5.4ns | Check valid name |
| ContainsName/invalid | 6.16µs | 6.2ns | Check invalid name |
| Name/valid | 2.31µs | 2.3ns | Get valid value name |
| Name/invalid | 55.12µs | 55ns | Get invalid value name (requires formatting) |
| List | 301ns | 0.3ns | Return all enum values |
| Names | 24.36µs | 24ns | Return all names |

### Performance Analysis

**Performance Highlights**:
- Core methods (`IsValid`, `String`, `Contains`, `Parse`) perform excellently with valid input (< 10ns/op)
- `List()` method takes only ~0.3ns as it directly returns pre-allocated slice
- Direct `MarshalJSON` call is ~**2.3x faster** than via `json.Marshal`
- Direct `UnmarshalJSON` call is ~**1.6x faster** than via `json.Unmarshal`

**Performance Difference Reasons**:
- Invalid value handling is slower due to error message formatting or default string generation
- `Scan/bytes` is slower than `Scan/string` due to type conversion
- Standard library calls are slower than direct calls due to reflection overhead

### Benchmark Code Location

Benchmark code is located in `cmd/enumgen/generator/generator_benchmark_test.go`, implemented using Ginkgo gmeasure package:

```go
experiment := gmeasure.NewExperiment("EnumGen Benchmark")
AddReportEntry(experiment.Name, experiment)

experiment.SampleDuration("IsValid/valid", func(idx int) {
    for i := 0; i < iterations; i++ {
        _ = gen.GenerateOptionString.IsValid()
    }
}, gmeasure.SamplingConfig{N: samples, Duration: time.Second})
```
