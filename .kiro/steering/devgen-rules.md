---
inclusion: always
fileMatchPattern: ['**/*.go']
---

# devgen AI Rules System

devgen provides an AI Rules system for generating documentation that AI coding assistants (like Kiro, CodeBuddy, Cursor, etc.) can understand.

## What are AI Rules?

AI Rules are structured documentation in Markdown format with YAML frontmatter that helps AI coding assistants understand how to use specific tools or frameworks. When you're writing code, AI assistants use these rules to provide more accurate suggestions.

## When to Use AI Rules?

Use AI Rules when you need to:
- Provide context-aware documentation to AI assistants
- Help AI understand your custom code generation tools
- Enable AI to suggest correct annotation usage
- Integrate your devgen plugins with AI coding assistants

## Commands

### List Supported AI Agents

```bash
devgen rules --list-agents
```

Output:
```
Supported AI agents:

  kiro          .kiro/steering/*.md
  codebuddy     .codebuddy/rules/*.md
  cursor        .cursor/rules/*.md

Usage: devgen rules --agent <name> [-w]
```

### Preview Rules Content

```bash
# Output to terminal (don't write files)
devgen rules --agent kiro
```

This displays all tool rules content, allowing you to review and confirm before writing.

### Generate Rules Files

```bash
# Write files
devgen rules --agent kiro -w
```

Output:
```
ℹ Generating rules for kiro...
✓ Generated 6 rule file(s) in .kiro/steering
  • .kiro/steering/enumgen.md
  • .kiro/steering/validategen.md
  • .kiro/steering/devgen.md
  • .kiro/steering/devgen-plugin.md
  • .kiro/steering/devgen-genkit.md
  • .kiro/steering/devgen-rules.md
```

## Agent-Specific Formats

Different AI assistants use different frontmatter formats. devgen automatically adapts rules for each agent using an adapter system.

### Kiro Format

Kiro uses YAML frontmatter with `inclusion` and `fileMatchPattern` fields:

```markdown
---
inclusion: fileMatch
fileMatchPattern: ['**/*.go']
---

# enumgen - Go Enum Code Generator

enumgen is part of the devgen toolkit...
```

**Frontmatter Fields**:
- `inclusion`: `always` (always loaded) or `fileMatch` (loaded when file matches pattern)
- `fileMatchPattern`: Array of glob patterns for file matching

### CodeBuddy Format

CodeBuddy uses YAML frontmatter with `description`, `globs`, and `alwaysApply` fields:

```markdown
---
description: Go enum code generator usage guide
globs: **/*.go
alwaysApply: false
---

# enumgen - Go Enum Code Generator

enumgen is part of the devgen toolkit...
```

**Frontmatter Fields**:
- `description`: Brief summary for context loading
- `globs`: File pattern (single string)
- `alwaysApply`: Whether to always include in context (boolean)

### Cursor Format

Cursor uses the same format as CodeBuddy:

```markdown
---
description: Go enum code generator usage guide
globs: **/*.go
alwaysApply: false
---

# enumgen - Go Enum Code Generator

enumgen is part of the devgen toolkit...
```

## Available Rules

| Rule Name | Description |
|-----------|-------------|
| enumgen | Enum code generation tool usage guide |
| validategen | Struct validation code generation tool usage guide |
| devgen | devgen comprehensive usage guide |
| devgen-plugin | Plugin development guide |
| devgen-genkit | genkit API reference |
| devgen-rules | AI Rules system documentation (this document) |

## Telling AI to Check Other Tool Rules

When you need to use a devgen tool, you can tell the AI assistant to check the corresponding rule:

### Example Conversation

**You**: I want to define an enum type for order status

**AI**: (After checking enumgen rule) Sure, you can define it like this:
```go
// OrderStatus order status
// enumgen:@enum(string, json)
type OrderStatus int

const (
    OrderStatusPending OrderStatus = iota + 1
    OrderStatusPaid
    OrderStatusShipped
)
```

## Implementing RuleTool for Custom Plugins

If you develop a custom plugin, you can implement the `RuleTool` interface to provide AI rules:

### Step 1: Implement RuleTool Interface

```go
package main

import "github.com/tlipoca9/devgen/genkit"

type MyGenerator struct{}

func (m *MyGenerator) Name() string { return "mygen" }

func (m *MyGenerator) Run(gen *genkit.Generator, log *genkit.Logger) error {
    // ... generate code
    return nil
}

// Rules returns AI rules
func (m *MyGenerator) Rules() []genkit.Rule {
    return []genkit.Rule{
        {
            Name:        "mygen",
            Description: "mygen code generator usage guide",
            Globs:       []string{"**/*.go"},
            AlwaysApply: false,
            Content:     mygenRuleContent,
        },
    }
}

var Tool genkit.Tool = &MyGenerator{}
```

### Step 2: Write Rule Content

Recommended: Store rule content in a separate .md file and embed it using go:embed:

```go
// rules/embed.go
package rules

import _ "embed"

//go:embed mygen.md
var MygenRule string
```

```markdown
<!-- rules/mygen.md -->
# mygen - Custom Code Generator

## When to Use mygen?

Use mygen when you need to:
- Feature 1
- Feature 2

## Quick Start

### Step 1: Add Annotation

```go
// MyType example type
// mygen:@gen
type MyType struct {
    Name string
}
```

### Step 2: Run Generation

```bash
devgen ./...
```

## Annotation Reference

| Annotation | Description | Example |
|------------|-------------|---------|
| @gen | Generate code | `mygen:@gen` |
| @gen(option) | Generate with option | `mygen:@gen(json)` |

## Complete Example

...

## Common Errors

### 1. Error Name

**Cause**: ...
**Solution**: ...
```

### Step 3: Reference in Generator

```go
package main

import (
    "github.com/tlipoca9/devgen/genkit"
    "myapp/plugins/mygen/rules"
)

func (m *MyGenerator) Rules() []genkit.Rule {
    return []genkit.Rule{
        {
            Name:        "mygen",
            Description: "mygen code generator usage guide",
            Globs:       []string{"**/*.go"},
            Content:     rules.MygenRule,  // Use embedded content
        },
    }
}
```

## Rule Structure Field Explanation

```go
type Rule struct {
    // Name is the rule filename (without extension)
    // Example: "enumgen" generates "enumgen.md"
    Name string

    // Description is a brief summary
    // AI assistants use this to decide whether to load this rule
    // Be clear about what this rule covers
    Description string

    // Globs are file matching patterns
    // When users open matching files, AI may auto-load this rule
    // Example: []string{"**/*.go", "**/*_enum.go"}
    Globs []string

    // AlwaysApply indicates whether to always load
    // true: rule is always in AI's context
    // false: loaded based on Globs or user request
    AlwaysApply bool

    // Content is the actual rule content (Markdown format)
    // This is what the AI will read, so be detailed and clear
    Content string
}
```

## Adapter System

devgen uses an adapter system to transform source rules into agent-specific formats. This allows you to maintain a single source of truth and generate rules for multiple AI assistants.

### How Adapters Work

1. **Source Rules**: Canonical English rules stored in `cmd/*/rules/` directories
2. **Adapter**: Transforms rules based on agent requirements
3. **Output**: Agent-specific rules with appropriate frontmatter

```
Source Rules (cmd/*/rules/*.md)
        ↓
    Adapter (Kiro/CodeBuddy/Cursor)
        ↓
Generated Rules (.kiro/steering/*.md, etc.)
```

### Creating Custom Adapters

You can create custom adapters for proprietary AI assistants by implementing the `AgentAdapter` interface:

```go
package main

import "github.com/tlipoca9/devgen/genkit"

type MyAgentAdapter struct{}

func (a *MyAgentAdapter) Name() string {
    return "myagent"
}

func (a *MyAgentAdapter) OutputDir() string {
    return ".myagent/rules"
}

func (a *MyAgentAdapter) Transform(rule genkit.Rule) (filename string, content string, err error) {
    // Build custom frontmatter
    frontmatter := fmt.Sprintf(`---
title: %s
description: %s
patterns: %v
---

`, rule.Name, rule.Description, rule.Globs)
    
    // Combine frontmatter with content
    content = frontmatter + rule.Content
    filename = rule.Name + ".md"
    
    return filename, content, nil
}

// Register adapter
func init() {
    genkit.RegisterAdapter(&MyAgentAdapter{})
}
```

### AgentAdapter Interface

```go
type AgentAdapter interface {
    // Name returns the agent name (e.g., "kiro", "codebuddy")
    Name() string
    
    // OutputDir returns the directory where rules should be written
    OutputDir() string
    
    // Transform converts a genkit.Rule to agent-specific format
    Transform(rule genkit.Rule) (filename string, content string, err error)
}
```

## Writing High-Quality Rule Content

### 1. Assume the Reader Knows Nothing

Don't assume the AI knows what your tool does. Explain from scratch:

```markdown
## When to Use XXX?

Use XXX when you need to:
- Scenario 1
- Scenario 2
- Scenario 3
```

### 2. Step-by-Step Instructions

Break down instructions into clear steps:

```markdown
## Quick Start

### Step 1: Define Type

```go
// Code example
```

### Step 2: Add Annotation

```go
// Code example
```

### Step 3: Run Generation

```bash
command
```

### Step 4: Use Generated Code

```go
// Code example
```
```

### 3. Plenty of Code Examples

Every feature should have a code example:

```markdown
### @required Annotation

Mark field as required:

```go
// validategen:@validate
type User struct {
    // validategen:@required
    Name string
}
```

Validation effect:
```go
user := User{Name: ""}
err := user.Validate()
// err: "Name is required"
```
```

### 4. List Common Errors

Help AI avoid common mistakes:

```markdown
## Common Errors

### 1. Forgot to Run devgen

```
Error: undefined: StatusEnums
Solution: Run devgen ./...
```

### 2. Annotation Format Error

```go
// ❌ Wrong
// enumgen@enum(string)  // Missing colon

// ✅ Correct
// enumgen:@enum(string)
```
```

### 5. Provide Complete Working Example

At the end of the document, provide a complete, runnable example:

```markdown
## Complete Example

### Definition File (models/user.go)

```go
package models

// User user model
// validategen:@validate
type User struct {
    // validategen:@required
    // validategen:@email
    Email string
}
```

### Usage Example

```go
package main

import "myapp/models"

func main() {
    user := models.User{Email: "test@example.com"}
    if err := user.Validate(); err != nil {
        log.Fatal(err)
    }
}
```
```

## Example: Complete Rule Document

```markdown
# mytool - Custom Code Generator

## When to Use mytool?

Use mytool when you need to:
- Generate boilerplate code for your domain models
- Automate repetitive coding patterns
- Ensure consistency across your codebase

## Quick Start

### Step 1: Add Annotation

```go
// User user model
// mytool:@gen
type User struct {
    Name  string
    Email string
}
```

### Step 2: Run Generation

```bash
devgen ./...
```

### Step 3: Use Generated Code

```go
user := User{Name: "John", Email: "john@example.com"}
info := user.Info()  // Generated method
fmt.Println(info)
```

## Annotation Reference

### @gen

Generates helper methods for the type.

**Syntax**: `mytool:@gen` or `mytool:@gen(options)`

**Options**:
- `json` - Generate JSON methods
- `xml` - Generate XML methods
- `detailed` - Generate detailed info methods

**Examples**:

```go
// Basic usage
// mytool:@gen
type User struct {
    Name string
}

// With options
// mytool:@gen(json, detailed)
type Product struct {
    ID   int
    Name string
}
```

## Complete Example

### Definition File (models/user.go)

```go
package models

// User user model
// mytool:@gen(json, detailed)
type User struct {
    ID    int64
    Name  string
    Email string
}
```

### Generated Code (models/user_gen.go)

```go
// Code generated by mytool. DO NOT EDIT.

package models

// Info returns detailed information about User
func (x User) Info() string {
    return "User with 3 fields"
}

// MarshalJSON implements json.Marshaler
func (x User) MarshalJSON() ([]byte, error) {
    // ... generated code
}
```

### Usage Example

```go
package main

import (
    "encoding/json"
    "fmt"
    "myapp/models"
)

func main() {
    user := models.User{
        ID:    1,
        Name:  "John Doe",
        Email: "john@example.com",
    }

    // Use generated Info method
    fmt.Println(user.Info())
    // Output: User with 3 fields

    // Use generated JSON methods
    data, _ := json.Marshal(user)
    fmt.Println(string(data))
    // Output: {"id":1,"name":"John Doe","email":"john@example.com"}
}
```

## Common Errors

### 1. Annotation Format Error

```go
// ❌ Wrong: Missing colon
// mytool@gen

// ❌ Wrong: Missing @ symbol
// mytool:gen

// ❌ Wrong: Missing tool name
// @gen

// ✅ Correct
// mytool:@gen
```

### 2. Annotation in Wrong Location

```go
// ❌ Wrong: Type annotation on field
type User struct {
    // mytool:@gen  // This is a field annotation location!
    Name string
}

// ✅ Correct: Type annotation on type
// mytool:@gen
type User struct {
    Name string
}
```

### 3. Forgot to Run devgen

```go
// After adding annotation, you must run devgen
// ❌ Wrong: Just add annotation and try to compile
// mytool:@gen
type User struct {
    Name string
}
// go build  // Error: undefined: User.Info

// ✅ Correct: Run devgen first
// devgen ./...
// go build  // Success
```

## Troubleshooting

### Rules Not Loading

**Symptoms**: AI doesn't seem to know about your tool

**Check**:
1. Verify rules files were generated: `ls .kiro/steering/`
2. Check frontmatter syntax is valid YAML
3. Ensure `fileMatchPattern` or `globs` matches your files
4. Try setting `AlwaysApply: true` for testing

### Rules Content Not Helpful

**Symptoms**: AI gives incorrect suggestions

**Improve**:
1. Add more detailed examples
2. Include common error cases
3. Provide step-by-step instructions
4. Add complete working examples

## Next Steps

- Learn about [plugin development](devgen-plugin.md) to create custom generators
- Learn about [genkit API](devgen-genkit.md) for implementing RuleTool
- Study [enumgen rules](enumgen.md) as a reference implementation
- Study [validategen rules](validategen.md) for comprehensive examples
```

## Best Practices

### 1. Keep Rules Up to Date

When you update your tool, update the rules:

```bash
# After changing tool behavior
devgen rules --agent kiro -w
devgen rules --agent codebuddy -w
devgen rules --agent cursor -w
```

### 2. Test Rules with AI

After generating rules, test them with your AI assistant:
1. Open a Go file
2. Ask AI about your tool
3. Verify AI provides correct suggestions
4. Refine rules based on AI responses

### 3. Use Descriptive Names

```go
// ❌ Bad: unclear
Rule{Name: "r1", Description: "stuff"}

// ✅ Good: clear and descriptive
Rule{Name: "mygen", Description: "mygen code generator usage guide"}
```

### 4. Organize Content Logically

Structure your rule content:
1. **When to Use** - Use cases and scenarios
2. **Quick Start** - Step-by-step getting started
3. **Annotation Reference** - Detailed annotation documentation
4. **Complete Example** - Full working example
5. **Common Errors** - Troubleshooting guide

### 5. Include Visual Aids

Use tables, code blocks, and formatting:

```markdown
| Annotation | Description | Example |
|------------|-------------|---------|
| @gen | Generate code | `mytool:@gen` |
| @skip | Skip generation | `mytool:@skip` |
```

## Next Steps

- Learn about [devgen usage](devgen.md) for command-line usage
- Learn about [plugin development](devgen-plugin.md) to create custom generators
- Learn about [genkit API](devgen-genkit.md) for implementing RuleTool
- Study existing rules as templates for your own
