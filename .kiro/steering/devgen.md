---
inclusion: fileMatch
fileMatchPattern: ['**/devgen.toml', '**/*.go']
---

# devgen - Go Code Generation Toolkit

devgen is a Go code generation toolkit that automatically generates boilerplate code through annotations.

## When to Use devgen?

Use devgen when you need to:
- Generate String(), JSON, SQL methods for enum types → use enumgen
- Generate Validate() validation methods for structs → use validategen
- Run multiple code generators together → use devgen to run them all
- Develop custom code generation tools → use the genkit framework
- Integrate AI-powered code generation guidance → use the rules system

## Installation

```bash
# Install devgen (includes all built-in tools)
go install github.com/tlipoca9/devgen/cmd/devgen@latest

# Or install individual tools
go install github.com/tlipoca9/devgen/cmd/enumgen@latest
go install github.com/tlipoca9/devgen/cmd/validategen@latest
```

## Quick Start

### Step 1: Add Annotations to Your Code

```go
// Status represents order status
// enumgen:@enum(string, json)
type Status int

const (
    StatusPending Status = iota + 1
    StatusActive
    StatusCompleted
)
```

### Step 2: Run Code Generation

```bash
# Process current directory and subdirectories
devgen ./...

# Process specific package
devgen ./pkg/models
```

### Step 3: Use Generated Code

```go
status := StatusPending
fmt.Println(status.String())  // "Pending"

// JSON serialization
data, _ := json.Marshal(status)  // "Pending"

// Validation
if status.IsValid() {
    // ...
}
```

## Command Line Usage

### Basic Usage

```bash
# Run all generators on current directory and subdirectories
devgen ./...

# Process specific package
devgen ./pkg/models

# Process all packages under a directory
devgen ./internal/...
```

### Dry-Run Mode

Validate annotations without writing files:

```bash
# Validate annotations, show files that would be generated
devgen --dry-run ./...

# JSON format output (for IDE integration)
devgen --dry-run --json ./...
```

JSON output example:
```json
{
  "success": true,
  "files": {
    "/path/to/models_enum.go": "// Code generated...",
    "/path/to/models_validate.go": "// Code generated..."
  },
  "stats": {
    "packagesLoaded": 5,
    "filesGenerated": 2,
    "errorCount": 0,
    "warningCount": 0
  }
}
```

### View Tool Configuration

```bash
# TOML format (human-readable)
devgen config

# JSON format (IDE/tool integration)
devgen config --json
```

### Generate AI Rules

```bash
# List supported AI agents
devgen rules --list-agents

# Preview rules (output to stdout)
devgen rules --agent kiro

# Write rules files
devgen rules --agent kiro -w
```

## Configuration File

devgen uses a `devgen.toml` configuration file to load plugins. The config file is searched from the current directory upward.

### Basic Configuration

```toml
# Plugin configuration
[[plugins]]
name = "myplugin"        # Plugin name
path = "./plugins/mygen" # Plugin path (relative or absolute)
type = "source"          # source (source code) or plugin (.so file)
```

### Multiple Plugins

```toml
[[plugins]]
name = "jsongen"
path = "./tools/jsongen"
type = "source"

[[plugins]]
name = "mockgen"
path = "./tools/mockgen.so"
type = "plugin"
```

## Built-in Tools

### enumgen - Enum Code Generator

Generates helper methods for Go enum types:

```go
// Status order status
// enumgen:@enum(string, json, sql)
type Status int

const (
    StatusPending Status = iota + 1
    StatusActive
    StatusCompleted
)
```

Generates: String(), MarshalJSON(), UnmarshalJSON(), Value(), Scan(), IsValid(), and more.

**See detailed usage**: AI assistants can access the complete documentation through the `enumgen` rule.

### validategen - Validation Code Generator

Generates Validate() methods for Go structs:

```go
// User user model
// validategen:@validate
type User struct {
    // validategen:@required
    // validategen:@email
    Email string

    // validategen:@min(0)
    // validategen:@max(150)
    Age int
}
```

Generates `func (x User) Validate() error` method.

**See detailed usage**: AI assistants can access the complete documentation through the `validategen` rule.

## Annotation Syntax

devgen tools use annotations to mark types and fields for processing.

### Basic Format

```
tool:@annotation
tool:@annotation(arg1, arg2)
tool:@annotation(key=value)
```

### Type Annotations

Place in the comment above the type definition:

```go
// MyType type description
// enumgen:@enum(string, json)
type MyType int
```

### Field Annotations

Place in the comment above the field:

```go
type User struct {
    // validategen:@required
    // validategen:@email
    Email string
}
```

## VSCode Extension

Install the [vscode-devgen](https://marketplace.visualstudio.com/items?itemName=tlipoca9.devgen) extension to get:

- Annotation syntax highlighting
- Auto-completion
- Parameter validation hints
- Error diagnostics

The extension automatically retrieves annotation configurations from all tools (including plugins) via `devgen config --json`.

## Troubleshooting

### 1. "undefined: XxxEnums" Error

**Cause**: Forgot to run devgen to generate code.

**Solution**:
```bash
devgen ./...
```

### 2. "package errors" Loading Failed

**Cause**: Code has syntax errors, or dependencies are not installed.

**Solution**:
```bash
# First ensure code compiles
go build ./...

# Then run devgen
devgen ./...
```

### 3. Generated Code Conflicts

**Cause**: Manually modified generated files.

**Solution**: Generated files start with `// Code generated`, don't modify them manually. Delete and regenerate:
```bash
rm *_enum.go *_validate.go
devgen ./...
```

### 4. Plugin Loading Failed

**Cause**: Plugin path is incorrect or code has issues.

**Solution**:
```bash
# Check plugin path
cat devgen.toml

# Compile plugin separately to confirm it works
go build ./plugins/myplugin
```

### 5. Annotation Not Working

**Cause**: Annotation format is incorrect.

**Check**:
- Format must be `tool:@name` or `tool:@name(args)`
- Colon and @ symbol are required
- Annotation must be in the correct location (type annotation on type, field annotation on field)

```go
// ❌ Wrong
// enumgen@enum(string)     // Missing colon
// enumgen:enum(string)     // Missing @
// @enum(string)            // Missing tool name

// ✅ Correct
// enumgen:@enum(string)
```

## Complete Working Example

### Define Types (models/order.go)

```go
package models

// OrderStatus order status
// enumgen:@enum(string, json, sql)
type OrderStatus int

const (
    OrderStatusPending    OrderStatus = iota + 1  // Pending
    OrderStatusProcessing                          // Processing
    OrderStatusCompleted                           // Completed
    OrderStatusCanceled                            // Canceled
)

// PaymentMethod payment method
// enumgen:@enum(string, json)
type PaymentMethod string

const (
    PaymentMethodCreditCard PaymentMethod = "credit_card"
    PaymentMethodDebitCard  PaymentMethod = "debit_card"
    PaymentMethodPayPal     PaymentMethod = "paypal"
)

// Order order model
// validategen:@validate
type Order struct {
    // validategen:@required
    // validategen:@gt(0)
    ID int64

    // validategen:@required
    Status OrderStatus

    // validategen:@required
    Payment PaymentMethod

    // validategen:@required
    // validategen:@min(1)
    Items []OrderItem
}

// OrderItem order item
// validategen:@validate
type OrderItem struct {
    // validategen:@required
    // validategen:@min(1)
    ProductID int64

    // validategen:@required
    // validategen:@gt(0)
    Quantity int

    // validategen:@required
    // validategen:@gt(0)
    Price float64
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
✓ Generated models_validate.go
```

### Use Generated Code (main.go)

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
        Items: []models.OrderItem{
            {
                ProductID: 101,
                Quantity:  2,
                Price:     29.99,
            },
        },
    }

    // Validate order
    if err := order.Validate(); err != nil {
        log.Fatal("Validation failed:", err)
    }

    // JSON serialization
    data, _ := json.MarshalIndent(order, "", "  ")
    fmt.Println(string(data))
    // {
    //   "id": 1,
    //   "status": "Pending",
    //   "payment": "credit_card",
    //   "items": [...]
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
    }

    // Display all options in dropdown
    for _, status := range models.OrderStatusEnums.List() {
        fmt.Printf("Value: %d, Name: %s\n", status, status.String())
    }
}
```

## Common Errors

### 1. Annotation Format Errors

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

### 2. Annotation in Wrong Location

```go
// ❌ Wrong: Type annotation on field
type User struct {
    // enumgen:@enum(string)  // This is a field annotation location!
    Name string
}

// ✅ Correct: Type annotation on type
// enumgen:@enum(string)
type Status int
```

### 3. Forgetting to Run devgen

```go
// After adding annotations, you must run devgen
// ❌ Wrong: Just add annotation and try to compile
// enumgen:@enum(string)
type Status int
// go build  // Error: undefined: StatusEnums

// ✅ Correct: Run devgen first
// devgen ./...
// go build  // Success
```

### 4. Modifying Generated Files

```go
// ❌ Wrong: Editing generated files
// File: models_enum.go
// Code generated by enumgen. DO NOT EDIT.
func (x Status) String() string {
    // Your custom changes here  // Will be overwritten!
}

// ✅ Correct: Add custom methods in separate file
// File: models_custom.go
func (x Status) CustomMethod() string {
    // Your custom logic here
}
```

### 5. Using Unsupported Types

```go
// ❌ Wrong: float64 not supported for enums
// enumgen:@enum(string)
type Score float64

// ❌ Wrong: bool not supported for enums
// enumgen:@enum(string)
type Flag bool

// ✅ Correct: Use int or string
// enumgen:@enum(string)
type Score int

// enumgen:@enum(string)
type Flag string
```

## Integration with Single Files

For single-file projects or when you want to generate code for a specific file, you can use `go generate`:

```go
// models.go
package models

//go:generate devgen .

// Status order status
// enumgen:@enum(string, json)
type Status int

const (
    StatusPending Status = iota + 1
    StatusActive
)
```

Then run:
```bash
go generate ./models.go
```

**Note**: This approach is best for single-file use cases. For multi-file projects, running `devgen ./...` directly is recommended.

## Next Steps

- Learn about [enumgen](enumgen.md) for enum code generation
- Learn about [validategen](validategen.md) for validation code generation
- Learn about [plugin development](devgen-plugin.md) to create custom generators
- Learn about [genkit API](devgen-genkit.md) for advanced usage
- Learn about [AI rules system](devgen-rules.md) for AI assistant integration
