---
inclusion: fileMatch
fileMatchPattern: ['*.go']
---

# validategen - Go Struct Validation Code Generator

validategen is part of the devgen toolkit, used to automatically generate Validate() methods for Go structs.

## When to Use validategen?

Use validategen when you need to:
- Validate API request parameters
- Validate configuration structs
- Validate user input
- Perform pre-database validation
- Ensure data integrity across your application
- Reduce boilerplate validation code

## Quick Start

### Step 1: Mark Struct

Add the `validategen:@validate` annotation above your struct definition:

```go
// User user model
// validategen:@validate
type User struct {
    Name  string
    Email string
    Age   int
}
```

### Step 2: Add Field Validation Rules

Add validation annotations in field comments:

```go
// User user model
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
}
```

### Step 3: Run Code Generation

```bash
devgen ./...
```

Generated file will be named `{package}_validate.go`.

### Step 4: Use Generated Method

```go
user := User{
    Name:  "J",       // Too short, min(2)
    Email: "invalid", // Invalid format
    Age:   -1,        // Less than 0
}

if err := user.Validate(); err != nil {
    fmt.Println(err)
    // Output: Name must have at least 2 characters, got 1; Email must be a valid email address; Age must be >= 0, got -1
}
```

## Validation Annotations Quick Reference

### Required Validation

| Annotation | Description | Example |
|------------|-------------|---------|
| @required | Field cannot be empty/zero value | `// validategen:@required` |

**@required Behavior by Type**:
- string: cannot be ""
- int/float: cannot be 0
- bool: must be true
- slice/map: length cannot be 0
- pointer: cannot be nil

```go
// validategen:@validate
type Config struct {
    // validategen:@required
    Host string  // Must not be empty

    // validategen:@required
    Port int  // Must not be 0

    // validategen:@required
    Enabled bool  // Must be true

    // validategen:@required
    Tags []string  // Must have at least one element
}
```

### Numeric/Length Validation

| Annotation | Description | Applies To |
|------------|-------------|------------|
| @min(n) | Minimum value/length | numbers, strings, slices, maps |
| @max(n) | Maximum value/length | numbers, strings, slices, maps |
| @len(n) | Exact length | strings, slices, maps |
| @gt(n) | Greater than | numbers, string length, slice length |
| @gte(n) | Greater than or equal | numbers, string length, slice length |
| @lt(n) | Less than | numbers, string length, slice length |
| @lte(n) | Less than or equal | numbers, string length, slice length |

```go
// validategen:@validate
type Config struct {
    // validategen:@min(1)
    // validategen:@max(65535)
    Port int  // 1 <= Port <= 65535

    // validategen:@min(2)
    // validategen:@max(50)
    Name string  // 2 <= len(Name) <= 50

    // validategen:@len(6)
    Code string  // len(Code) == 6

    // validategen:@gt(0)
    ID int64  // ID > 0

    // validategen:@gte(1)
    // validategen:@lte(10)
    Items []string  // 1 <= len(Items) <= 10
}
```

### Equality Validation

| Annotation | Description | Example |
|------------|-------------|---------|
| @eq(value) | Must equal specified value | `@eq(1)`, `@eq(active)` |
| @ne(value) | Cannot equal specified value | `@ne(0)`, `@ne(deleted)` |
| @oneof(a,b,c) | Must be one of specified values | `@oneof(admin, user, guest)` |

```go
// validategen:@validate
type Request struct {
    // validategen:@eq(2)
    Version int  // Version == 2

    // validategen:@ne(deleted)
    Status string  // Status != "deleted"

    // validategen:@oneof(GET, POST, PUT, DELETE)
    Method string  // Method must be one of the HTTP methods
}
```

### Format Validation

| Annotation | Description | Example |
|------------|-------------|---------|
| @email | Email format | user@example.com |
| @url | URL format | https://example.com |
| @uuid | UUID format | 550e8400-e29b-41d4-a716-446655440000 |
| @ip | IP address (v4 or v6) | 192.168.1.1 |
| @ipv4 | IPv4 address | 192.168.1.1 |
| @ipv6 | IPv6 address | ::1 |
| @alpha | Letters only | abc |
| @alphanum | Letters and numbers | abc123 |
| @numeric | Numeric string | 12345 |

```go
// validategen:@validate
type Contact struct {
    // validategen:@required
    // validategen:@email
    Email string

    // validategen:@url
    Website string  // Optional, but must be valid URL if provided

    // validategen:@uuid
    TraceID string

    // validategen:@ipv4
    ServerIP string

    // validategen:@alphanum
    // validategen:@len(8)
    InviteCode string  // 8-character alphanumeric invite code
}
```

### String Content Validation

| Annotation | Description | Example |
|------------|-------------|---------|
| @contains(s) | Must contain substring | `@contains(@)` |
| @excludes(s) | Cannot contain substring | `@excludes(<script>)` |
| @startswith(s) | Must start with prefix | `@startswith(https://)` |
| @endswith(s) | Must end with suffix | `@endswith(.go)` |
| @regex(pattern) | Match regular expression | `@regex(^[A-Z]{2}-\\d{4}$)` |

```go
// validategen:@validate
type Input struct {
    // validategen:@startswith(https://)
    SecureURL string

    // validategen:@excludes(<script>)
    // validategen:@excludes(javascript:)
    UserContent string  // Prevent XSS

    // validategen:@regex(^1[3-9]\\d{9}$)
    PhoneNumber string  // Chinese mobile phone format

    // validategen:@contains(@)
    // validategen:@contains(.)
    EmailLike string  // Must contain @ and .

    // validategen:@endswith(.pdf)
    DocumentPath string  // Must be a PDF file
}
```

### Duration Validation

| Annotation | Description | Example |
|------------|-------------|---------|
| @duration | Valid Go duration format | 1h30m, 500ms |
| @duration_min(d) | Minimum duration | `@duration_min(1s)` |
| @duration_max(d) | Maximum duration | `@duration_max(24h)` |

```go
// validategen:@validate
type Config struct {
    // validategen:@duration
    // validategen:@duration_min(100ms)
    // validategen:@duration_max(30s)
    Timeout string  // Valid duration, 100ms <= timeout <= 30s

    // validategen:@duration
    // validategen:@duration_min(1m)
    RetryInterval string  // At least 1 minute

    // validategen:@duration
    MaxAge string  // Any valid duration
}
```

### Format Validation (JSON/YAML/TOML/CSV)

| Annotation | Description |
|------------|-------------|
| @format(json) | Valid JSON format |
| @format(yaml) | Valid YAML format |
| @format(toml) | Valid TOML format |
| @format(csv) | Valid CSV format |

```go
// validategen:@validate
type Template struct {
    // validategen:@format(json)
    JSONTemplate string

    // validategen:@format(yaml)
    YAMLConfig string

    // validategen:@format(toml)
    TOMLConfig string

    // validategen:@format(csv)
    CSVData string
}
```

### Enum Type Validation

| Annotation | Description |
|------------|-------------|
| @oneof_enum(Type) | Must be a valid enum value |

```go
// Used with enumgen
// enumgen:@enum(string, json)
type Status int

const (
    StatusPending Status = iota + 1
    StatusActive
    StatusInactive
)

// validategen:@validate
type User struct {
    // validategen:@oneof_enum(Status)
    Status Status  // Automatically uses StatusEnums.Contains() for validation
}
```

**Cross-Package Enums**:

```go
// validategen:@validate
type Request struct {
    // validategen:@oneof_enum(github.com/myorg/pkg/types.Status)
    Status types.Status  // Automatically adds import
}
```

### Nested Struct Validation

| Annotation | Description |
|------------|-------------|
| @method(MethodName) | Call specified method for validation |

```go
type Address struct {
    Street string
    City   string
}

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
    BillingAddress *Address  // Calls Validate() if not nil
}
```

## Advanced Features

### postValidate Hook

For custom validation logic (like cross-field validation), define a postValidate method:

```go
// validategen:@validate
type User struct {
    // validategen:@required
    Role string

    // validategen:@gte(0)
    Age int
}

// postValidate custom validation logic
// Generated Validate() method calls this after all field validations
func (x User) postValidate(errs []string) error {
    // Cross-field validation: admin must be at least 18 years old
    if x.Role == "admin" && x.Age < 18 {
        errs = append(errs, "admin must be at least 18 years old")
    }

    if len(errs) > 0 {
        return fmt.Errorf("%s", strings.Join(errs, "; "))
    }
    return nil
}
```

### Example: Complex Cross-Field Validation

```go
// validategen:@validate
type DateRange struct {
    // validategen:@required
    StartDate time.Time

    // validategen:@required
    EndDate time.Time
}

func (x DateRange) postValidate(errs []string) error {
    // Ensure EndDate is after StartDate
    if !x.EndDate.After(x.StartDate) {
        errs = append(errs, "end date must be after start date")
    }

    if len(errs) > 0 {
        return fmt.Errorf("%s", strings.Join(errs, "; "))
    }
    return nil
}
```

### Multiple Rules Combination

A field can use multiple validation rules, executed in order:

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

### Example: Comprehensive Field Validation

```go
// validategen:@validate
type RegistrationForm struct {
    // validategen:@required
    // validategen:@min(3)
    // validategen:@max(20)
    // validategen:@alphanum
    Username string  // Required, 3-20 chars, alphanumeric

    // validategen:@required
    // validategen:@min(8)
    // validategen:@max(100)
    // validategen:@contains(A-Z)
    // validategen:@contains(a-z)
    // validategen:@contains(0-9)
    Password string  // Required, 8-100 chars, must contain upper, lower, and digit

    // validategen:@required
    // validategen:@email
    Email string  // Required, valid email

    // validategen:@gte(18)
    // validategen:@lte(120)
    Age int  // 18-120 years old

    // validategen:@url
    Website string  // Optional, but must be valid URL if provided
}
```

## Complete Working Example

### Definition File (models/user.go)

```go
package models

import (
    "fmt"
    "strings"
)

// Address address
type Address struct {
    // validategen:@required
    Street string
    // validategen:@required
    City string
    // validategen:@len(6)
    ZipCode string
}

func (a Address) Validate() error {
    var errs []string
    if a.Street == "" {
        errs = append(errs, "street is required")
    }
    if a.City == "" {
        errs = append(errs, "city is required")
    }
    if len(a.ZipCode) != 6 {
        errs = append(errs, "zipcode must be 6 characters")
    }
    if len(errs) > 0 {
        return fmt.Errorf("%s", strings.Join(errs, "; "))
    }
    return nil
}

// User user model
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

    // validategen:@method(Validate)
    Address Address

    // validategen:@method(Validate)
    BillingAddress *Address
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

### Generate Code

```bash
cd models
devgen ./...
```

Output:
```
ℹ Loading packages...
✓ Loaded 1 package(s)
✓ Generated models_validate.go
```

### Usage Example (main.go)

```go
package main

import (
    "fmt"
    "myapp/models"
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
        Address:  models.Address{Street: "123 Main St", City: "NYC", ZipCode: "100001"},
    }

    if err := user.Validate(); err != nil {
        fmt.Println("Validation failed:", err)
    } else {
        fmt.Println("Validation passed!")
    }

    // Invalid user
    invalidUser := models.User{
        ID:       0,           // Invalid: gt(0)
        Name:     "",          // Invalid: required
        Email:    "invalid",   // Invalid: email format
        Password: "short",     // Invalid: min(8)
        Role:     "superuser", // Invalid: oneof
    }

    if err := invalidUser.Validate(); err != nil {
        fmt.Println("Validation failed:", err)
        // Output: All errors, separated by semicolons
    }
}
```

## Common Errors

### 1. Forgot to Add @validate Annotation

```go
// ❌ Wrong: No @validate, won't generate Validate() method
type User struct {
    // validategen:@required
    Name string
}

// ✅ Correct: Add @validate
// validategen:@validate
type User struct {
    // validategen:@required
    Name string
}
```

### 2. Annotation Format Errors

```go
// ❌ Wrong: Missing colon
// validategen@required
Name string

// ❌ Wrong: Missing @
// validategen:required
Name string

// ✅ Correct
// validategen:@required
Name string
```

### 3. Parameter Type Mismatch

```go
// ❌ Wrong: @email only works with string
// validategen:@email
Age int

// ❌ Wrong: @min parameter must be a number
// validategen:@min(abc)
Count int

// ✅ Correct
// validategen:@min(0)
Count int
```

### 4. Empty String Skips Format Validation

Note: @email, @url, @uuid, etc. skip validation for empty strings. If the field is required, add @required:

```go
// ❌ Wrong: Empty string passes validation
// validategen:@email
Email string

// ✅ Correct: Required + format validation
// validategen:@required
// validategen:@email
Email string
```

### 5. Annotation in Wrong Location

```go
// ❌ Wrong: Type annotation on field
type User struct {
    // validategen:@validate  // This is a field annotation location!
    Name string
}

// ✅ Correct: Type annotation on type
// validategen:@validate
type User struct {
    Name string
}
```

### 6. Using @oneof_enum Without enumgen

```go
// ❌ Wrong: Status is not an enum type
type Status string

// validategen:@validate
type User struct {
    // validategen:@oneof_enum(Status)  // Won't work!
    Status Status
}

// ✅ Correct: Use enumgen first
// enumgen:@enum(string, json)
type Status int

const (
    StatusActive Status = iota + 1
    StatusInactive
)

// validategen:@validate
type User struct {
    // validategen:@oneof_enum(Status)  // Now works!
    Status Status
}
```

## Integration with enumgen

validategen works seamlessly with enumgen-generated enums:

```go
// enumgen:@enum(string, json)
type Role int

const (
    RoleAdmin Role = iota + 1
    RoleUser
    RoleGuest
)

// validategen:@validate
type User struct {
    // validategen:@oneof_enum(Role)
    Role Role
}
```

When Role gets new values, validation logic automatically includes them without manual updates.

### Example: Multiple Enum Fields

```go
// enumgen:@enum(string, json)
type Status int

const (
    StatusActive Status = iota + 1
    StatusInactive
    StatusPending
)

// enumgen:@enum(string, json)
type Priority int

const (
    PriorityLow Priority = iota + 1
    PriorityMedium
    PriorityHigh
)

// validategen:@validate
type Task struct {
    // validategen:@required
    // validategen:@oneof_enum(Status)
    Status Status

    // validategen:@required
    // validategen:@oneof_enum(Priority)
    Priority Priority
}
```

## Validation Error Messages

Generated validation methods return descriptive error messages:

```go
user := User{
    Name:  "J",       // Too short
    Email: "invalid", // Invalid format
    Age:   -1,        // Negative
}

err := user.Validate()
// Error message: "Name must have at least 2 characters, got 1; Email must be a valid email address; Age must be >= 0, got -1"
```

### Example: Parsing Validation Errors

```go
if err := user.Validate(); err != nil {
    // Split by semicolon to get individual errors
    errors := strings.Split(err.Error(), "; ")
    for _, e := range errors {
        fmt.Println("- ", e)
    }
}
// Output:
// - Name must have at least 2 characters, got 1
// - Email must be a valid email address
// - Age must be >= 0, got -1
```

## Troubleshooting

### Validation Not Working

**Symptoms**: Validate() method not found or not working

**Solutions**:
1. Run `devgen ./...` to generate code
2. Check that `*_validate.go` file was created
3. Ensure `@validate` annotation is on the type
4. Verify field annotations are in comments above fields

### All Fields Pass Validation

**Symptoms**: Invalid data passes validation

**Check**:
1. Verify annotations are correctly formatted
2. Ensure you're calling Validate() on the struct
3. Check that generated code is up to date (re-run devgen)
4. Verify field types match annotation requirements

### postValidate Not Called

**Symptoms**: Custom validation logic not executing

**Check**:
1. Method signature must be: `func (x Type) postValidate(errs []string) error`
2. Method must be defined on the same type
3. Re-run devgen after adding postValidate

### Cross-Package Enum Validation Fails

**Symptoms**: @oneof_enum with imported enum doesn't work

**Solution**:
```go
// ❌ Wrong: Using short package name
// validategen:@oneof_enum(types.Status)

// ✅ Correct: Using full import path
// validategen:@oneof_enum(github.com/myorg/pkg/types.Status)
```

## Best Practices

### 1. Validate at Boundaries

Validate data at system boundaries (API handlers, database operations):

```go
func CreateUser(w http.ResponseWriter, r *http.Request) {
    var user User
    json.NewDecoder(r.Body).Decode(&user)
    
    // Validate immediately
    if err := user.Validate(); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Proceed with valid data
    saveUser(user)
}
```

### 2. Combine with Type Safety

Use enums for type-safe validation:

```go
// enumgen:@enum(string, json)
type UserRole int

const (
    UserRoleAdmin UserRole = iota + 1
    UserRoleUser
    UserRoleGuest
)

// validategen:@validate
type User struct {
    // validategen:@oneof_enum(UserRole)
    Role UserRole  // Type-safe + validated
}
```

### 3. Use Descriptive Error Messages

Add context in postValidate:

```go
func (x User) postValidate(errs []string) error {
    if x.Role == "admin" && x.Age < 18 {
        errs = append(errs, "admin role requires age >= 18, got "+strconv.Itoa(x.Age))
    }
    // ...
}
```

### 4. Validate Nested Structs

Use @method for nested validation:

```go
// validategen:@validate
type Order struct {
    // validategen:@method(Validate)
    Customer User

    // validategen:@method(Validate)
    Items []OrderItem  // Validates each item
}
```

### 5. Keep Validation Logic Simple

For complex business rules, use postValidate or separate validation functions:

```go
// Simple field validation with annotations
// validategen:@validate
type User struct {
    // validategen:@required
    // validategen:@email
    Email string
}

// Complex business logic in separate function
func (x User) ValidateBusinessRules() error {
    // Complex cross-system validation
    if !isEmailDomainAllowed(x.Email) {
        return fmt.Errorf("email domain not allowed")
    }
    return nil
}
```

## Next Steps

- Learn about [enumgen](enumgen.md) for enum code generation
- Learn about [devgen](devgen.md) for overall toolkit usage
- Learn about [plugin development](devgen-plugin.md) to create custom generators
- Study the [generated code](https://github.com/tlipoca9/devgen/tree/main/cmd/validategen/generator/testdata) to understand implementation details
