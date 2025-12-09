---
inclusion: fileMatch
fileMatchPattern: ['*.go']
---

# enumgen - Go Enum Code Generator

enumgen is part of the devgen toolkit, used to automatically generate helper methods for Go enum types.

## When to Use enumgen?

Use enumgen when you need to:
- Define type-safe enums (instead of bare int/string constants)
- Convert between enum values and strings (String() method)
- Serialize/deserialize enums to/from JSON
- Store enums in databases (SQL driver interface)
- Validate whether a value is a valid enum value
- Get all possible enum values for dropdowns or validation

## Quick Start

### Step 1: Define Enum Type

Add the `enumgen:@enum(...)` annotation in the comment above your type definition:

```go
// Status represents order status
// enumgen:@enum(string, json)
type Status int

const (
    StatusPending   Status = iota + 1  // Pending
    StatusConfirmed                     // Confirmed
    StatusShipped                       // Shipped
    StatusDelivered                     // Delivered
)
```

### Step 2: Run Code Generation

```bash
# In your project root directory
devgen ./...

# Or process specific package
devgen ./pkg/models
```

Generated file will be named `{package}_enum.go`, for example `models_enum.go`.

### Step 3: Use Generated Code

```go
status := StatusPending

// Check if valid (always generated)
if status.IsValid() {
    fmt.Println("Valid status")
}

// String representation (requires 'string' parameter)
fmt.Println(status.String())  // Output: "Pending"

// JSON serialization (requires 'json' parameter)
data, _ := json.Marshal(status)  // "Pending"

// Parse from string
status, err := StatusEnums.Parse("Confirmed")
if err != nil {
    // Handle invalid string
}

// Get all valid values
allStatuses := StatusEnums.List()
// Returns: []Status{StatusPending, StatusConfirmed, StatusShipped, StatusDelivered}
```

## Annotation Parameters

| Parameter | Purpose | Generated Methods |
|-----------|---------|-------------------|
| string | Implement fmt.Stringer interface | String() string |
| json | Implement JSON serialization interfaces | MarshalJSON() / UnmarshalJSON() |
| text | Implement text serialization interfaces | MarshalText() / UnmarshalText() |
| sql | Implement database interfaces | Value() / Scan() |

**Common Combinations**:
- `enumgen:@enum(string)` - Only need print output
- `enumgen:@enum(string, json)` - Common for API interfaces
- `enumgen:@enum(string, json, sql)` - Need database storage

### Example: Different Parameter Combinations

```go
// Basic: Only string representation
// enumgen:@enum(string)
type Priority int

const (
    PriorityLow Priority = iota + 1
    PriorityMedium
    PriorityHigh
)

// API: String + JSON
// enumgen:@enum(string, json)
type OrderStatus int

const (
    OrderStatusPending OrderStatus = iota + 1
    OrderStatusPaid
    OrderStatusShipped
)

// Full: String + JSON + SQL
// enumgen:@enum(string, json, sql)
type UserRole int

const (
    UserRoleGuest UserRole = iota + 1
    UserRoleUser
    UserRoleAdmin
)
```

## Generated Helper Variable

For a `Status` type, enumgen generates a `StatusEnums` helper variable:

```go
// Get all valid values
allStatuses := StatusEnums.List()
// Returns: []Status{StatusPending, StatusConfirmed, StatusShipped, StatusDelivered}

// Check if value is valid
isValid := StatusEnums.Contains(StatusPending)  // true
isValid = StatusEnums.Contains(Status(999))     // false

// Parse from string
status, err := StatusEnums.Parse("Pending")
if err != nil {
    // Handle invalid string
}

// Get string name
name := StatusEnums.Name(StatusPending)  // "Pending"

// Get all names
names := StatusEnums.Names()  // []string{"Pending", "Confirmed", "Shipped", "Delivered"}

// Check if name exists
hasName := StatusEnums.ContainsName("Pending")  // true
```

### Example: Using Helper Variable

```go
// Validate user input
func ValidateStatus(input string) (Status, error) {
    status, err := StatusEnums.Parse(input)
    if err != nil {
        return 0, fmt.Errorf("invalid status: %s. Valid values: %v", 
            input, StatusEnums.Names())
    }
    return status, nil
}

// Populate dropdown options
func GetStatusOptions() []string {
    return StatusEnums.Names()
}

// Check if status is in a set
func IsActiveStatus(status Status) bool {
    activeStatuses := []Status{StatusConfirmed, StatusShipped}
    for _, s := range activeStatuses {
        if status == s {
            return true
        }
    }
    return false
}
```

## Generated Type Methods

```go
status := StatusPending

// Check if valid (always generated)
if status.IsValid() {
    // Process valid status
}

// String() method (requires 'string' parameter)
fmt.Println(status)  // Output: "Pending"
fmt.Printf("Status: %s\n", status)  // Output: "Status: Pending"

// JSON serialization (requires 'json' parameter)
data, _ := json.Marshal(status)  // "Pending"

var s Status
json.Unmarshal([]byte(`"Confirmed"`), &s)  // s = StatusConfirmed

// Text serialization (requires 'text' parameter)
text, _ := status.MarshalText()  // []byte("Pending")

var s Status
s.UnmarshalText([]byte("Confirmed"))  // s = StatusConfirmed

// SQL database (requires 'sql' parameter)
value, _ := status.Value()  // driver.Value for database storage

var s Status
s.Scan(someValue)  // Load from database
```

### Example: JSON API Usage

```go
type Order struct {
    ID     int64  `json:"id"`
    Status Status `json:"status"`
}

// Serialize to JSON
order := Order{ID: 1, Status: StatusPending}
data, _ := json.Marshal(order)
fmt.Println(string(data))
// Output: {"id":1,"status":"Pending"}

// Deserialize from JSON
var order Order
json.Unmarshal([]byte(`{"id":1,"status":"Confirmed"}`), &order)
fmt.Println(order.Status)  // StatusConfirmed
```

## Customizing Enum Value Names

By default, enum value string names automatically remove the type name prefix:
- `StatusPending` → `"Pending"`
- `StatusConfirmed` → `"Confirmed"`

To customize names, use the `@name` annotation:

```go
// ErrorCode error code
// enumgen:@enum(string, json)
type ErrorCode int

const (
    // enumgen:@name(ERR_NOT_FOUND)
    ErrorCodeNotFound ErrorCode = 404

    // enumgen:@name(ERR_INTERNAL)
    ErrorCodeInternal ErrorCode = 500

    // enumgen:@name(ERR_BAD_REQUEST)
    ErrorCodeBadRequest ErrorCode = 400
)
```

Usage:
```go
fmt.Println(ErrorCodeNotFound.String())  // "ERR_NOT_FOUND"
code, _ := ErrorCodeEnums.Parse("ERR_INTERNAL")  // ErrorCodeInternal
```

### Example: Custom Names for API Compatibility

```go
// HTTPMethod HTTP method
// enumgen:@enum(string, json)
type HTTPMethod int

const (
    // enumgen:@name(GET)
    HTTPMethodGet HTTPMethod = iota + 1
    
    // enumgen:@name(POST)
    HTTPMethodPost
    
    // enumgen:@name(PUT)
    HTTPMethodPut
    
    // enumgen:@name(DELETE)
    HTTPMethodDelete
)

// Usage
method := HTTPMethodGet
fmt.Println(method.String())  // "GET"

// Parse from HTTP request
method, err := HTTPMethodEnums.Parse(r.Method)
```

## String Underlying Type

enumgen also supports `string` as the underlying type:

```go
// Color color enum
// enumgen:@enum(string, json)
type Color string

const (
    ColorRed   Color = "red"
    ColorGreen Color = "green"
    ColorBlue  Color = "blue"
)
```

**Note**: String type enums:
- ✅ Support IsValid(), String(), JSON, SQL methods
- ❌ Don't support `@name` annotation (string value itself is the name)
- ❌ Don't generate Name(), Names(), ContainsName() methods

### Example: String Enum Usage

```go
// Environment environment type
// enumgen:@enum(string, json)
type Environment string

const (
    EnvironmentDevelopment Environment = "development"
    EnvironmentStaging     Environment = "staging"
    EnvironmentProduction  Environment = "production"
)

// Usage
env := EnvironmentProduction
fmt.Println(env.String())  // "production"

// Validation
if env.IsValid() {
    fmt.Println("Valid environment")
}

// Get all values
allEnvs := EnvironmentEnums.List()
// []Environment{"development", "staging", "production"}
```

## Supported Underlying Types

| Type | Supported |
|------|-----------|
| int, int8, int16, int32, int64 | ✅ |
| uint, uint8, uint16, uint32, uint64 | ✅ |
| string | ✅ |
| float32, float64 | ❌ |
| bool | ❌ |

### Example: Different Integer Types

```go
// Small enum: use int8
// enumgen:@enum(string, json)
type DayOfWeek int8

const (
    Monday DayOfWeek = iota + 1
    Tuesday
    Wednesday
    Thursday
    Friday
    Saturday
    Sunday
)

// Large enum: use int32
// enumgen:@enum(string, json)
type CountryCode int32

const (
    CountryCodeUSA CountryCode = 1
    CountryCodeUK  CountryCode = 44
    CountryCodeJP  CountryCode = 81
    // ... many more
)
```

## Complete Working Example

### Definition File (models/order.go)

```go
package models

// OrderStatus order status
// enumgen:@enum(string, json, sql)
type OrderStatus int

const (
    OrderStatusPending    OrderStatus = iota + 1  // Pending
    OrderStatusProcessing                          // Processing
    OrderStatusCompleted                           // Completed
    // enumgen:@name(Cancelled)
    OrderStatusCanceled                            // Canceled (using British spelling)
)

// PaymentMethod payment method
// enumgen:@enum(string, json)
type PaymentMethod string

const (
    PaymentMethodCreditCard PaymentMethod = "credit_card"
    PaymentMethodDebitCard  PaymentMethod = "debit_card"
    PaymentMethodPayPal     PaymentMethod = "paypal"
    PaymentMethodCrypto     PaymentMethod = "crypto"
)

// Order order model
type Order struct {
    ID      int64         `json:"id"`
    Status  OrderStatus   `json:"status"`
    Payment PaymentMethod `json:"payment"`
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
✓ Generated models_enum.go
```

### Usage Example (main.go)

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "myapp/models"
)

func main() {
    // Create order
    order := models.Order{
        ID:      1,
        Status:  models.OrderStatusPending,
        Payment: models.PaymentMethodCreditCard,
    }

    // JSON serialization
    data, _ := json.MarshalIndent(order, "", "  ")
    fmt.Println(string(data))
    // Output:
    // {
    //   "id": 1,
    //   "status": "Pending",
    //   "payment": "credit_card"
    // }

    // Parse from API request
    var req struct {
        Status models.OrderStatus `json:"status"`
    }
    json.Unmarshal([]byte(`{"status":"Processing"}`), &req)
    fmt.Println(req.Status.String())  // "Processing"

    // Validate user input
    userInput := "InvalidStatus"
    if _, err := models.OrderStatusEnums.Parse(userInput); err != nil {
        fmt.Println("Invalid order status:", userInput)
        fmt.Println("Valid values:", models.OrderStatusEnums.Names())
    }

    // Display all options in dropdown
    fmt.Println("Available statuses:")
    for _, status := range models.OrderStatusEnums.List() {
        fmt.Printf("  Value: %d, Name: %s\n", status, status.String())
    }
    // Output:
    //   Value: 1, Name: Pending
    //   Value: 2, Name: Processing
    //   Value: 3, Name: Completed
    //   Value: 4, Name: Cancelled

    // Check if status is valid
    status := models.OrderStatus(999)
    if !status.IsValid() {
        fmt.Println("Invalid status value:", status)
    }

    // Database operations (requires 'sql' parameter)
    // value, _ := order.Status.Value()  // For INSERT/UPDATE
    // order.Status.Scan(dbValue)        // For SELECT
}
```

## Common Errors

### 1. Forgot to Run devgen

```
Error: undefined: StatusEnums
Solution: Run devgen ./...
```

**Explanation**: After adding annotations, you must run devgen to generate the code.

```bash
# ❌ Wrong: Just add annotation and try to compile
// enumgen:@enum(string)
type Status int
// go build  // Error: undefined: StatusEnums

# ✅ Correct: Run devgen first
devgen ./...
go build  // Success
```

### 2. Unsupported Underlying Type

```go
// ❌ Wrong: float64 not supported
// enumgen:@enum(string)
type Score float64

// ❌ Wrong: bool not supported
// enumgen:@enum(string)
type Flag bool

// ✅ Correct: Use int or string
// enumgen:@enum(string)
type Score int

// enumgen:@enum(string)
type Flag string
```

### 3. Using @name with String Type

```go
// ❌ Wrong: string type doesn't support @name
// enumgen:@enum(string)
type Color string

const (
    // enumgen:@name(RED)  // This will cause an error!
    ColorRed Color = "red"
)

// ✅ Correct: Use the desired string value directly
const (
    ColorRed Color = "RED"
)
```

### 4. Duplicate @name Values

```go
// ❌ Wrong: Duplicate @name values
const (
    // enumgen:@name(Active)
    StatusActive Status = 1
    // enumgen:@name(Active)  // Duplicate!
    StatusEnabled Status = 2
)

// ✅ Correct: Unique names
const (
    // enumgen:@name(Active)
    StatusActive Status = 1
    // enumgen:@name(Enabled)
    StatusEnabled Status = 2
)
```

### 5. Annotation Format Errors

```go
// ❌ Wrong: Missing colon
// enumgen@enum(string)

// ❌ Wrong: Missing @ symbol
// enumgen:enum(string)

// ❌ Wrong: Missing tool name
// @enum(string)

// ✅ Correct
// enumgen:@enum(string)
```

### 6. Annotation in Wrong Location

```go
// ❌ Wrong: Annotation on constant instead of type
type Status int

const (
    // enumgen:@enum(string)  // Wrong location!
    StatusPending Status = iota + 1
)

// ✅ Correct: Annotation on type
// enumgen:@enum(string)
type Status int

const (
    StatusPending Status = iota + 1
)
```

## Integration with validategen

enumgen-generated enums work seamlessly with validategen's `@oneof_enum`:

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
    // validategen:@required
    // validategen:@oneof_enum(Role)
    Role Role
}
```

When Role gets new values, validation logic automatically includes them without manual updates.

### Example: Cross-Package Enum Validation

```go
// In package models
// enumgen:@enum(string, json)
type Status int

const (
    StatusActive Status = iota + 1
    StatusInactive
)

// In package api
import "myapp/models"

// validategen:@validate
type Request struct {
    // validategen:@oneof_enum(myapp/models.Status)
    Status models.Status
}
```

## Advanced Usage

### Enum with Non-Sequential Values

```go
// HTTPStatus HTTP status code
// enumgen:@enum(string, json)
type HTTPStatus int

const (
    HTTPStatusOK                  HTTPStatus = 200
    HTTPStatusCreated             HTTPStatus = 201
    HTTPStatusBadRequest          HTTPStatus = 400
    HTTPStatusUnauthorized        HTTPStatus = 401
    HTTPStatusNotFound            HTTPStatus = 404
    HTTPStatusInternalServerError HTTPStatus = 500
)
```

### Enum with Bit Flags

```go
// Permission permission flags
// enumgen:@enum(string)
type Permission int

const (
    PermissionRead   Permission = 1 << iota  // 1
    PermissionWrite                          // 2
    PermissionExecute                        // 4
    PermissionDelete                         // 8
)

// Note: For bit flags, you typically want to check individual bits
// rather than using IsValid() for combinations
func HasPermission(perms, perm Permission) bool {
    return perms&perm != 0
}
```

### Enum with String Values Matching Constants

```go
// LogLevel log level
// enumgen:@enum(string, json)
type LogLevel string

const (
    LogLevelDebug LogLevel = "DEBUG"
    LogLevelInfo  LogLevel = "INFO"
    LogLevelWarn  LogLevel = "WARN"
    LogLevelError LogLevel = "ERROR"
)
```

## Troubleshooting

### Generated Code Not Found

**Symptoms**: `undefined: StatusEnums` error

**Solutions**:
1. Run `devgen ./...` to generate code
2. Check that `*_enum.go` file was created
3. Ensure the file is in the same package
4. Run `go build ./...` to verify compilation

### String() Returns Numbers

**Symptoms**: `fmt.Println(status)` outputs `1` instead of `"Pending"`

**Cause**: Forgot to include `string` parameter in annotation

**Solution**:
```go
// ❌ Wrong: No 'string' parameter
// enumgen:@enum(json)
type Status int

// ✅ Correct: Include 'string' parameter
// enumgen:@enum(string, json)
type Status int
```

### JSON Serialization Not Working

**Symptoms**: JSON output shows numbers instead of strings

**Cause**: Forgot to include `json` parameter in annotation

**Solution**:
```go
// ❌ Wrong: No 'json' parameter
// enumgen:@enum(string)
type Status int

// ✅ Correct: Include 'json' parameter
// enumgen:@enum(string, json)
type Status int
```

### Parse() Always Returns Error

**Symptoms**: `StatusEnums.Parse("Pending")` returns error

**Possible Causes**:
1. Wrong string value (case-sensitive)
2. Using custom @name but parsing with default name
3. Type has no constants defined

**Solutions**:
```go
// Check exact names
names := StatusEnums.Names()
fmt.Println(names)  // See what names are valid

// Parse is case-sensitive
status, _ := StatusEnums.Parse("Pending")   // ✅ Correct
status, _ := StatusEnums.Parse("pending")   // ❌ Wrong (lowercase)
status, _ := StatusEnums.Parse("PENDING")   // ❌ Wrong (uppercase)
```

## Next Steps

- Learn about [validategen](validategen.md) for struct validation
- Learn about [devgen](devgen.md) for overall toolkit usage
- Learn about [plugin development](devgen-plugin.md) to create custom generators
- Study the [generated code](https://github.com/tlipoca9/devgen/tree/main/cmd/enumgen/generator/testdata) to understand implementation details
