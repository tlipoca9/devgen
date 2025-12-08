# validategen

[中文](README.md) | English

`validategen` is a Go struct validation code generation tool that generates `Validate()` methods for structs annotated with `validategen:@validate`.

## Installation

```bash
go install github.com/tlipoca9/devgen/cmd/validategen@latest
```

## Usage

```bash
validategen ./...              # All packages
validategen ./pkg/models       # Specific package
```

## Annotations

### @validate - Mark Struct for Validation

Add annotation above struct definition to indicate that a `Validate()` method should be generated:

```go
// User model
// validategen:@validate
type User struct {
    // validategen:@required
    Name string
}
```

### Field-Level Validation Annotations

Add validation rules in field comments. Multiple rules can be combined.

---

## Validation Rules

### 1. @required - Required Validation

Validates that field is not empty/zero value.

| Field Type | Validation Logic |
|------------|------------------|
| `string` | Cannot be empty string `""` |
| `int/float/...` | Cannot be `0` |
| `bool` | Must be `true` |
| `slice/map` | Length cannot be `0` |
| `pointer` | Cannot be `nil` |

```go
// validategen:@validate
type User struct {
    // validategen:@required
    Name string  // Required string

    // validategen:@required
    Age int  // Required number (cannot be 0)

    // validategen:@required
    IsActive bool  // Must be true

    // validategen:@required
    Tags []string  // Slice cannot be empty

    // validategen:@required
    Profile *Profile  // Pointer cannot be nil
}
```

---

### 2. @min(n) - Minimum Value/Length

| Field Type | Validation Logic |
|------------|------------------|
| `string` | String length >= n |
| `slice/map` | Element count >= n |
| `int/float/...` | Value >= n |

---

### 3. @max(n) - Maximum Value/Length

| Field Type | Validation Logic |
|------------|------------------|
| `string` | String length <= n |
| `slice/map` | Element count <= n |
| `int/float/...` | Value <= n |

---

### 4. @len(n) - Exact Length

| Field Type | Validation Logic |
|------------|------------------|
| `string` | String length == n |
| `slice/map` | Element count == n |

---

### 5. @gt(n) - Greater Than

| Field Type | Validation Logic |
|------------|------------------|
| `string` | String length > n |
| `slice/map` | Element count > n |
| `int/float/...` | Value > n |

---

### 6. @gte(n) - Greater Than or Equal

| Field Type | Validation Logic |
|------------|------------------|
| `string` | String length >= n |
| `slice/map` | Element count >= n |
| `int/float/...` | Value >= n |

---

### 7. @lt(n) - Less Than

| Field Type | Validation Logic |
|------------|------------------|
| `string` | String length < n |
| `slice/map` | Element count < n |
| `int/float/...` | Value < n |

---

### 8. @lte(n) - Less Than or Equal

| Field Type | Validation Logic |
|------------|------------------|
| `string` | String length <= n |
| `slice/map` | Element count <= n |
| `int/float/...` | Value <= n |

---

### 9. @eq(value) - Equal

Validates that field value equals specified value. Supports `string`, `int/float`, `bool` types.

```go
// validategen:@validate
type Config struct {
    // validategen:@eq(1)
    Version int  // Version must equal 1

    // validategen:@eq(active)
    Status string  // Status must equal "active"

    // validategen:@eq(true)
    Enabled bool  // Must be true
}
```

---

### 10. @ne(value) - Not Equal

Validates that field value does not equal specified value. Supports `string`, `int/float`, `bool` types.

---

### 11. @oneof(a, b, c) - Enum Values

Validates that field value is one of specified values. Supports `string` and numeric types.

```go
// validategen:@validate
type User struct {
    // validategen:@oneof(admin, user, guest)
    Role string  // Role must be admin, user, or guest

    // validategen:@oneof(1, 2, 3)
    Level int  // Level must be 1, 2, or 3
}
```

---

### 12. @email - Email Format

Validates string is a valid email address. Empty strings skip validation.

---

### 13. @url - URL Format

Validates string is a valid URL. Empty strings skip validation.

---

### 14. @uuid - UUID Format

Validates string is a valid UUID (8-4-4-4-12 format). Empty strings skip validation.

---

### 15. @ip - IP Address

Validates string is a valid IP address (IPv4 or IPv6). Empty strings skip validation.

---

### 16. @ipv4 - IPv4 Address

Validates string is a valid IPv4 address. Empty strings skip validation.

---

### 17. @ipv6 - IPv6 Address

Validates string is a valid IPv6 address. Empty strings skip validation.

---

### 18. @duration - Duration Format

Validates string is a valid Go duration format (e.g., `1h30m`, `500ms`). Empty strings skip validation.

```go
// validategen:@validate
type Config struct {
    // validategen:@duration
    Timeout string  // Must be a valid duration format
}
```

---

### 19. @duration_min(duration) - Minimum Duration

Validates duration string value is not less than specified value. Empty strings skip validation.

```go
// validategen:@validate
type Config struct {
    // validategen:@duration_min(1s)
    Timeout string  // Timeout at least 1 second

    // validategen:@duration_min(100ms)
    RetryInterval string  // Retry interval at least 100 milliseconds
}
```

---

### 20. @duration_max(duration) - Maximum Duration

Validates duration string value is not greater than specified value. Empty strings skip validation.

```go
// validategen:@validate
type Config struct {
    // validategen:@duration_max(1h)
    Timeout string  // Timeout at most 1 hour

    // validategen:@duration_max(30s)
    RetryInterval string  // Retry interval at most 30 seconds
}
```

---

### 21. @duration + @duration_min + @duration_max Combined

These three annotations can be combined. Generated code merges into a single block, parsing only once:

```go
// validategen:@validate
type Config struct {
    // validategen:@duration
    // validategen:@duration_min(1s)
    // validategen:@duration_max(1h)
    RetryInterval string  // Valid duration, range 1s ~ 1h
}
```

---

### 22. @alpha - Letters Only

Validates string contains only letters (a-zA-Z). Empty strings skip validation.

---

### 23. @alphanum - Alphanumeric

Validates string contains only letters and numbers (a-zA-Z0-9). Empty strings skip validation.

---

### 24. @numeric - Numbers Only

Validates string contains only digits (0-9). Empty strings skip validation.

---

### 25. @contains(substring) - Contains Substring

Validates string contains specified substring.

---

### 26. @excludes(substring) - Excludes Substring

Validates string does not contain specified substring.

---

### 27. @startswith(prefix) - Prefix Match

Validates string starts with specified prefix.

---

### 28. @endswith(suffix) - Suffix Match

Validates string ends with specified suffix.

---

### 29. @regex(pattern) - Regular Expression

Validates string matches specified regex pattern. Empty strings skip validation.

```go
// validategen:@validate
type Product struct {
    // validategen:@regex(^[A-Z]{2}-\d{4}$)
    ProductCode string  // Format: XX-0000 (two uppercase letters-four digits)

    // validategen:@regex(^\d{4}-\d{2}-\d{2}$)
    Date string  // Format: YYYY-MM-DD
}
```

---

### 30. @format(type) - Format Validation

Validates string is valid specified format. Supports `json`, `yaml`, `toml`, `csv`. Empty strings skip validation.

| Format | Description |
|--------|-------------|
| `json` | Validated using `encoding/json.Valid` |
| `yaml` | Validated by parsing with `gopkg.in/yaml.v3` |
| `toml` | Validated by parsing with `github.com/BurntSushi/toml` |
| `csv` | Validated by parsing with `encoding/csv` |

---

### 31. @method(MethodName) - Call Validation Method

Calls validation method on nested struct or custom type. For pointer types, nil check is performed first.

```go
// Address
type Address struct {
    Street string
    City   string
}

// Validate validates address
func (a Address) Validate() error {
    if a.Street == "" {
        return fmt.Errorf("street is required")
    }
    if a.City == "" {
        return fmt.Errorf("city is required")
    }
    return nil
}

// validategen:@validate
type User struct {
    // validategen:@method(Validate)
    Address Address  // Calls Address.Validate()

    // validategen:@method(Validate)
    OptionalAddress *Address  // Calls Validate() when not nil
}
```

---

## Advanced Features

### postValidate Hook

If struct defines a `postValidate(errs []string) error` method, the generated `Validate()` method will call it after all field validations, passing the collected error list.

```go
// validategen:@validate
type User struct {
    // validategen:@required
    Role string

    // validategen:@gte(0)
    Age int
}

// postValidate custom validation logic
func (x User) postValidate(errs []string) error {
    if x.Role == "admin" && x.Age < 18 {
        errs = append(errs, "admin must be at least 18 years old")
    }
    if len(errs) > 0 {
        return fmt.Errorf("%s", strings.Join(errs, "; "))
    }
    return nil
}
```

### Multiple Rules

A field can use multiple validation rules simultaneously:

```go
// validategen:@validate
type User struct {
    // validategen:@required
    // validategen:@min(2)
    // validategen:@max(50)
    // validategen:@alpha
    Name string  // Required, 2-50 characters, letters only

    // validategen:@required
    // validategen:@email
    Email string  // Required and valid email format

    // validategen:@required
    // validategen:@min(1)
    // validategen:@max(65535)
    Port int  // Required, range 1-65535
}
```

---

## Complete Example

### Definition

```go
package models

import "fmt"

// Address
type Address struct {
    Street string
    City   string
}

// Validate validates address
func (a Address) Validate() error {
    if a.Street == "" {
        return fmt.Errorf("street is required")
    }
    if a.City == "" {
        return fmt.Errorf("city is required")
    }
    return nil
}

// User model
// validategen:@validate
type User struct {
    // validategen:@required
    // validategen:@gt(0)
    ID int64

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

    // validategen:@required
    // validategen:@min(8)
    Password string

    // validategen:@oneof(admin, user, guest)
    Role string

    // validategen:@url
    Website string

    // validategen:@uuid
    UUID string

    // validategen:@ip
    IP string

    // validategen:@alphanum
    // validategen:@len(6)
    Code string

    // validategen:@method(Validate)
    Address Address

    // validategen:@method(Validate)
    OptionalAddress *Address
}

// postValidate custom validation
func (x User) postValidate(errs []string) error {
    if x.Role == "admin" && x.Age < 18 {
        errs = append(errs, "admin must be at least 18 years old")
    }
    if len(errs) > 0 {
        return fmt.Errorf("%s", strings.Join(errs, "; "))
    }
    return nil
}
```

Run code generation:

```bash
validategen ./...
```

### Usage

```go
package main

import (
    "fmt"
    
    "example.com/models"
)

func main() {
    // Valid user
    user := models.User{
        ID:       1,
        Name:     "John Doe",
        Email:    "john@example.com",
        Age:      25,
        Password: "password123",
        Role:     "user",
        Code:     "ABC123",
        Address:  models.Address{Street: "123 Main St", City: "New York"},
    }
    
    if err := user.Validate(); err != nil {
        fmt.Println("Validation failed:", err)
    } else {
        fmt.Println("Validation passed!")
    }
    
    // Invalid user
    invalidUser := models.User{
        ID:       0,  // Invalid: required and gt(0)
        Name:     "",  // Invalid: required
        Email:    "invalid-email",  // Invalid: email format
        Password: "short",  // Invalid: min(8)
        Role:     "invalid",  // Invalid: oneof
    }
    
    if err := invalidUser.Validate(); err != nil {
        fmt.Println("Validation failed:", err)
        // Output: ID is required; ID must be greater than 0, got 0; Name is required; ...
    }
}
```

---

## Annotation Quick Reference

| Annotation | Parameter | Applicable Types | Description |
|------------|-----------|------------------|-------------|
| `@validate` | - | struct | Mark struct to generate Validate method |
| `@required` | - | string, number, bool, slice, map, pointer | Required validation |
| `@min(n)` | number | string, slice, map, number | Minimum value/length |
| `@max(n)` | number | string, slice, map, number | Maximum value/length |
| `@len(n)` | number | string, slice, map | Exact length |
| `@gt(n)` | number | string, slice, map, number | Greater than |
| `@gte(n)` | number | string, slice, map, number | Greater than or equal |
| `@lt(n)` | number | string, slice, map, number | Less than |
| `@lte(n)` | number | string, slice, map, number | Less than or equal |
| `@eq(v)` | string/number/bool | string, number, bool | Equal |
| `@ne(v)` | string/number/bool | string, number, bool | Not equal |
| `@oneof(a, b, c)` | comma-separated values | string, number | Enum values |
| `@email` | - | string | Email format |
| `@url` | - | string | URL format |
| `@uuid` | - | string | UUID format |
| `@ip` | - | string | IP address (v4 or v6) |
| `@ipv4` | - | string | IPv4 address |
| `@ipv6` | - | string | IPv6 address |
| `@duration` | - | string | Duration format (e.g., 1h30m, 500ms) |
| `@duration_min(d)` | duration | string | Minimum duration |
| `@duration_max(d)` | duration | string | Maximum duration |
| `@alpha` | - | string | Letters only |
| `@alphanum` | - | string | Alphanumeric |
| `@numeric` | - | string | Numbers only |
| `@contains(s)` | string | string | Contains substring |
| `@excludes(s)` | string | string | Excludes substring |
| `@startswith(s)` | string | string | Prefix match |
| `@endswith(s)` | string | string | Suffix match |
| `@regex(pattern)` | regex | string | Regex match |
| `@format(type)` | json, yaml, toml, csv | string | Format validation |
| `@method(name)` | method name | struct, pointer, custom type | Call validation method |

---

## Supported Numeric Types

validategen supports all Go built-in numeric types:

- Signed integers: `int`, `int8`, `int16`, `int32`, `int64`
- Unsigned integers: `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- Floating point: `float32`, `float64`
- Alias types: `byte` (uint8), `rune` (int32), `uintptr`

---

## Testing and Benchmarks

### Run Unit Tests

```bash
# Run all tests
go test -v ./cmd/validategen/generator/...

# Run tests with coverage
go test -v ./cmd/validategen/generator/... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

### Run Benchmarks

Benchmarks are implemented using [Ginkgo gmeasure](https://onsi.github.io/ginkgo/#benchmarking-code) to obtain statistically meaningful performance data.

#### Step 1: Run Benchmarks

```bash
# Run tests (including benchmarks)
cd /path/to/devgen
go test -v ./cmd/validategen/generator/... -count=1
```

#### Step 2: Generate Report Files

```bash
# Generate JSON format report
ginkgo --json-report=benchmark_report.json ./cmd/validategen/generator/...

# Generate JUnit XML format report
ginkgo --junit-report=benchmark_report.xml ./cmd/validategen/generator/...

# Generate both formats
ginkgo --json-report=benchmark_report.json --junit-report=benchmark_report.xml ./cmd/validategen/generator/...
```

### Benchmark Results Summary

**Test Environment**:
- OS: darwin (macOS)
- Arch: arm64
- CPU: Apple M4 Pro
- 1000 iterations per annotation, 100 samples

#### Simple Validation Annotations (No Regex)

| Annotation | Mean (1000x) | Per Op ~= | Description |
|------------|--------------|-----------|-------------|
| `@required` (string) | 272ns | 0.27ns | String non-empty check |
| `@required` (int) | 273ns | 0.27ns | Numeric non-zero check |
| `@required` (slice) | 269ns | 0.27ns | Slice length check |
| `@required` (pointer) | 288ns | 0.29ns | Pointer non-nil check |
| `@min` / `@max` (int) | 270ns | 0.27ns | Numeric comparison |
| `@min` / `@max` (string len) | 271ns | 0.27ns | String length comparison |
| `@len` | 272ns | 0.27ns | Exact length check |
| `@gt` / `@gte` / `@lt` / `@lte` | 271ns | 0.27ns | Numeric comparison |
| `@eq` / `@ne` | 270ns | 0.27ns | Equality comparison |
| `@oneof` (string, 4 values) | 531ns | 0.53ns | Enum value check |
| `@oneof` (int) | 269ns | 0.27ns | Integer enum check |

#### String Operation Annotations

| Annotation | Mean (1000x) | Per Op ~= | Description |
|------------|--------------|-----------|-------------|
| `@contains` | 4.8µs | 4.8ns | `strings.Contains` |
| `@excludes` | 4.5µs | 4.5ns | `!strings.Contains` |
| `@startswith` | 299ns | 0.30ns | `strings.HasPrefix` |
| `@endswith` | 1.6µs | 1.6ns | `strings.HasSuffix` |

#### Regex Validation Annotations

| Annotation | Mean (1000x) | Per Op ~= | Description |
|------------|--------------|-----------|-------------|
| `@email` (valid) | 180µs | 180ns | Email regex match |
| `@email` (invalid) | 156µs | 156ns | Regex fast fail |
| `@url` (valid) | 253µs | 253ns | URL regex match |
| `@url` (invalid) | 16µs | 16ns | Regex fast fail |
| `@uuid` (valid) | 152µs | 152ns | UUID regex match |
| `@uuid` (invalid) | 1.3µs | 1.3ns | Regex fast fail |
| `@alpha` | 89µs | 89ns | Letters only regex |
| `@alphanum` | 78µs | 78ns | Alphanumeric regex |
| `@numeric` | 89µs | 89ns | Numbers only regex |
| `@regex` (simple) | 45µs | 45ns | Simple custom regex |
| `@regex` (complex) | 246µs | 246ns | Complex regex pattern |

#### Format Validation Annotations

| Annotation | Mean (1000x) | Per Op ~= | Description |
|------------|--------------|-----------|-------------|
| `@format(json)` (valid) | 69µs | 69ns | `json.Valid` |
| `@format(json)` (invalid) | 131µs | 131ns | JSON parse failure |
| `@format(yaml)` (valid) | 3.85ms | 3.85µs | YAML parsing |
| `@format(yaml)` (invalid) | 3.37ms | 3.37µs | YAML parse failure |
| `@format(toml)` (valid) | 1.72ms | 1.72µs | TOML parsing |
| `@format(toml)` (invalid) | 1.31ms | 1.31µs | TOML parse failure |
| `@format(csv)` (valid) | 885µs | 885ns | CSV parsing |
| `@format(csv)` (invalid) | 854µs | 854ns | CSV parse failure |

#### IP Address Validation Annotations

| Annotation | Mean (1000x) | Per Op ~= | Description |
|------------|--------------|-----------|-------------|
| `@ip` (ipv4) | 12.6µs | 12.6ns | `net.ParseIP` |
| `@ip` (ipv6) | 42µs | 42ns | IPv6 parsing slower |
| `@ipv4` | 15.6µs | 15.6ns | IPv4 + To4() check |
| `@ipv6` | 43.6µs | 43.6ns | IPv6 + To4() check |

#### Duration Validation Annotations

| Annotation | Mean (1000x) | Per-op | Description |
|------------|--------------|--------|-------------|
| `@duration` (valid) | 25.45µs | 25.45ns | `time.ParseDuration` |
| `@duration` (invalid) | 67.56µs | 67.56ns | Parse failure |
| `@duration_min` | 11.78µs | 11.78ns | Parse then compare nanoseconds |
| `@duration_max` | ~11µs | ~11ns | Parse then compare nanoseconds |
| Combined | ~11µs | ~11ns | Parse once, compare multiple times |

### Performance Analysis

**Performance Highlights**:
- Simple validation annotations perform excellently (< 1ns/op): `@required`, `@min`, `@max`, `@eq`, `@ne`, etc.
- Pre-compiled regex patterns avoid repeated compilation overhead
- Invalid input is usually faster than valid input because regex can fail fast

**Performance Difference Reasons**:
- Format validation (`@format`) requires full parsing; YAML/TOML are slower (µs level)
- Regex validation (`@email`, `@uuid`, `@regex`) is 100-500x slower than simple comparison
- IP address parsing requires calling `net.ParseIP`, medium performance
- String operations (`@contains`, `@startswith`) performance is between the two
- JSON validation is fastest (using `json.Valid`), CSV is second
