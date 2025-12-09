# AI Rules Adapter System

[中文](rules-adapter.md) | English

The adapter system in devgen allows you to maintain a single source of truth for AI rules and automatically generate agent-specific formats for different AI assistants.

## Overview

The adapter system follows a "source → adapter → output" pattern:

```
Source Rules (cmd/*/rules/*.md)
        ↓
    Adapter (Kiro/CodeBuddy/Cursor)
        ↓
Generated Rules (.kiro/steering/*.md, etc.)
```

### Why Adapters?

Different AI assistants use different frontmatter formats and conventions. Instead of maintaining separate documentation for each assistant, you write rules once and let adapters handle the transformation.

## Built-in Adapters

devgen includes three built-in adapters:

| Adapter | Output Directory | File Extension | Frontmatter Format |
|---------|-----------------|----------------|-------------------|
| **Kiro** | `.kiro/steering/` | `.md` | YAML with `inclusion` and `fileMatchPattern` |
| **CodeBuddy** | `.codebuddy/rules/` | `.mdc` | YAML with `description`, `globs`, `alwaysApply` |
| **Cursor** | `.cursor/rules/` | `.mdc` | YAML with `description`, `globs`, `alwaysApply` |

### Kiro Adapter

Generates rules with Kiro-specific frontmatter:

```markdown
---
inclusion: fileMatch
fileMatchPattern: ['**/*.go', '**/devgen.toml']
---

# Rule Content
...
```

**Frontmatter Fields**:
- `inclusion`: `always` (always loaded) or `fileMatch` (loaded when file matches pattern)
- `fileMatchPattern`: Array of glob patterns for file matching

### CodeBuddy Adapter

Generates rules with CodeBuddy-specific frontmatter:

```markdown
---
description: Brief description for context loading
globs: **/*.go
alwaysApply: false
---

# Rule Content
...
```

**Frontmatter Fields**:
- `description`: Brief summary for AI context loading
- `globs`: File pattern (single string)
- `alwaysApply`: Whether to always include in context (boolean)

### Cursor Adapter

Uses the same format as CodeBuddy:

```markdown
---
description: Brief description for context loading
globs: **/*.go
alwaysApply: false
---

# Rule Content
...
```

## Creating Custom Adapters

You can create custom adapters for proprietary or new AI assistants by implementing the `AgentAdapter` interface.

### AgentAdapter Interface

```go
package genkit

// AgentAdapter transforms rules for a specific AI assistant
type AgentAdapter interface {
    // Name returns the agent name (e.g., "kiro", "codebuddy")
    Name() string
    
    // OutputDir returns the directory where rules should be written
    OutputDir() string
    
    // Transform converts a genkit.Rule to agent-specific format
    Transform(rule Rule) (filename string, content string, err error)
}
```

### Example: Custom Adapter Implementation

Here's a complete example of creating a custom adapter for a hypothetical AI assistant called "MyAI":

```go
package main

import (
    "fmt"
    "strings"
    
    "github.com/tlipoca9/devgen/genkit"
)

// MyAIAdapter implements AgentAdapter for MyAI assistant
type MyAIAdapter struct{}

func (a *MyAIAdapter) Name() string {
    return "myai"
}

func (a *MyAIAdapter) OutputDir() string {
    return ".myai/docs"
}

func (a *MyAIAdapter) Transform(rule genkit.Rule) (string, string, error) {
    // Build custom frontmatter
    var frontmatter strings.Builder
    frontmatter.WriteString("---\n")
    frontmatter.WriteString(fmt.Sprintf("title: %s\n", rule.Name))
    frontmatter.WriteString(fmt.Sprintf("description: %s\n", rule.Description))
    
    // Handle file patterns
    if len(rule.Globs) > 0 {
        frontmatter.WriteString("patterns:\n")
        for _, glob := range rule.Globs {
            frontmatter.WriteString(fmt.Sprintf("  - %s\n", glob))
        }
    }
    
    // Handle always apply
    if rule.AlwaysApply {
        frontmatter.WriteString("autoLoad: true\n")
    }
    
    frontmatter.WriteString("---\n\n")
    
    // Combine frontmatter with content
    content := frontmatter.String() + rule.Content
    filename := rule.Name + ".md"
    
    return filename, content, nil
}
```

### Registering Custom Adapters

To make your custom adapter available to devgen, register it with the adapter registry:

```go
package main

import (
    "github.com/tlipoca9/devgen/genkit"
)

func init() {
    // Get the global adapter registry
    registry := genkit.GetAdapterRegistry()
    
    // Register your custom adapter
    registry.Register(&MyAIAdapter{})
}
```

### Using Custom Adapters in Plugins

If you're developing a plugin, you can register adapters when your plugin loads:

```go
package main

import (
    "github.com/tlipoca9/devgen/genkit"
)

type MyGenerator struct{}

func (m *MyGenerator) Name() string { return "mygen" }

func (m *MyGenerator) Run(gen *genkit.Generator, log *genkit.Logger) error {
    // Register custom adapter
    registry := genkit.GetAdapterRegistry()
    registry.Register(&MyAIAdapter{})
    
    // ... rest of your generator logic
    return nil
}

var Tool genkit.Tool = &MyGenerator{}

func main() {}
```

## Adapter Best Practices

### 1. Preserve Content Integrity

Always preserve the original rule content without modification:

```go
func (a *MyAdapter) Transform(rule genkit.Rule) (string, string, error) {
    frontmatter := buildFrontmatter(rule)
    
    // ✅ Good: Preserve original content
    content := frontmatter + rule.Content
    
    // ❌ Bad: Modifying content
    // content := frontmatter + strings.ToUpper(rule.Content)
    
    return rule.Name + ".md", content, nil
}
```

### 2. Handle Empty Fields Gracefully

Not all rules will have all fields populated:

```go
func (a *MyAdapter) Transform(rule genkit.Rule) (string, string, error) {
    var frontmatter strings.Builder
    frontmatter.WriteString("---\n")
    
    // ✅ Good: Check before using
    if rule.Description != "" {
        frontmatter.WriteString(fmt.Sprintf("description: %s\n", rule.Description))
    }
    
    if len(rule.Globs) > 0 {
        // Handle globs
    }
    
    frontmatter.WriteString("---\n\n")
    return rule.Name + ".md", frontmatter.String() + rule.Content, nil
}
```

### 3. Use Consistent Filename Patterns

Generate predictable filenames:

```go
func (a *MyAdapter) Transform(rule genkit.Rule) (string, string, error) {
    // ✅ Good: Simple, predictable
    filename := rule.Name + ".md"
    
    // ❌ Bad: Complex, unpredictable
    // filename := fmt.Sprintf("%s_%d.markdown", rule.Name, time.Now().Unix())
    
    content := buildContent(rule)
    return filename, content, nil
}
```

### 4. Validate YAML Frontmatter

Ensure your frontmatter is valid YAML:

```go
import "gopkg.in/yaml.v3"

func (a *MyAdapter) Transform(rule genkit.Rule) (string, string, error) {
    // Build frontmatter as a struct
    fm := map[string]interface{}{
        "title":       rule.Name,
        "description": rule.Description,
        "patterns":    rule.Globs,
    }
    
    // Marshal to YAML
    yamlBytes, err := yaml.Marshal(fm)
    if err != nil {
        return "", "", fmt.Errorf("marshal frontmatter: %w", err)
    }
    
    // Build final content
    content := "---\n" + string(yamlBytes) + "---\n\n" + rule.Content
    return rule.Name + ".md", content, nil
}
```

### 5. Document Your Adapter

Provide clear documentation for users of your adapter:

```go
// MyAIAdapter transforms rules for MyAI assistant.
//
// Frontmatter format:
//   title: Rule name
//   description: Brief description
//   patterns: List of file patterns
//   autoLoad: Whether to always load (optional)
//
// Output directory: .myai/docs/
type MyAIAdapter struct{}
```

## Testing Custom Adapters

### Unit Test Example

```go
package main

import (
    "strings"
    "testing"
    
    "github.com/tlipoca9/devgen/genkit"
)

func TestMyAIAdapter_Transform(t *testing.T) {
    adapter := &MyAIAdapter{}
    
    rule := genkit.Rule{
        Name:        "test-rule",
        Description: "Test rule description",
        Globs:       []string{"**/*.go", "**/*.md"},
        AlwaysApply: true,
        Content:     "# Test Rule\n\nThis is test content.",
    }
    
    filename, content, err := adapter.Transform(rule)
    
    // Check no error
    if err != nil {
        t.Fatalf("Transform failed: %v", err)
    }
    
    // Check filename
    if filename != "test-rule.md" {
        t.Errorf("Expected filename 'test-rule.md', got '%s'", filename)
    }
    
    // Check frontmatter
    if !strings.Contains(content, "title: test-rule") {
        t.Error("Frontmatter missing title")
    }
    
    if !strings.Contains(content, "description: Test rule description") {
        t.Error("Frontmatter missing description")
    }
    
    if !strings.Contains(content, "autoLoad: true") {
        t.Error("Frontmatter missing autoLoad")
    }
    
    // Check content preservation
    if !strings.Contains(content, "# Test Rule") {
        t.Error("Original content not preserved")
    }
}

func TestMyAIAdapter_Name(t *testing.T) {
    adapter := &MyAIAdapter{}
    if adapter.Name() != "myai" {
        t.Errorf("Expected name 'myai', got '%s'", adapter.Name())
    }
}

func TestMyAIAdapter_OutputDir(t *testing.T) {
    adapter := &MyAIAdapter{}
    if adapter.OutputDir() != ".myai/docs" {
        t.Errorf("Expected output dir '.myai/docs', got '%s'", adapter.OutputDir())
    }
}
```

### Integration Test Example

```go
func TestMyAIAdapter_Integration(t *testing.T) {
    // Create temporary directory
    tmpDir := t.TempDir()
    
    // Create adapter
    adapter := &MyAIAdapter{}
    
    // Create test rule
    rule := genkit.Rule{
        Name:    "integration-test",
        Content: "# Integration Test\n\nTest content.",
    }
    
    // Transform
    filename, content, err := adapter.Transform(rule)
    if err != nil {
        t.Fatalf("Transform failed: %v", err)
    }
    
    // Write file
    filepath := filepath.Join(tmpDir, adapter.OutputDir(), filename)
    if err := os.MkdirAll(filepath.Dir(filepath), 0755); err != nil {
        t.Fatalf("Create directory failed: %v", err)
    }
    
    if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
        t.Fatalf("Write file failed: %v", err)
    }
    
    // Verify file exists and is readable
    readContent, err := os.ReadFile(filepath)
    if err != nil {
        t.Fatalf("Read file failed: %v", err)
    }
    
    if string(readContent) != content {
        t.Error("File content doesn't match")
    }
}
```

## Advanced Topics

### Dynamic Frontmatter Fields

Some adapters may need to add fields based on rule content:

```go
func (a *MyAdapter) Transform(rule genkit.Rule) (string, string, error) {
    fm := map[string]interface{}{
        "title": rule.Name,
    }
    
    // Add category based on rule name
    if strings.HasPrefix(rule.Name, "enum") {
        fm["category"] = "code-generation"
    } else if strings.HasPrefix(rule.Name, "validate") {
        fm["category"] = "validation"
    }
    
    // Add tags based on content
    if strings.Contains(rule.Content, "JSON") {
        fm["tags"] = []string{"json", "serialization"}
    }
    
    yamlBytes, _ := yaml.Marshal(fm)
    content := "---\n" + string(yamlBytes) + "---\n\n" + rule.Content
    
    return rule.Name + ".md", content, nil
}
```

### Content Transformation

While generally discouraged, some adapters may need to transform content:

```go
func (a *MyAdapter) Transform(rule genkit.Rule) (string, string, error) {
    // Build frontmatter
    frontmatter := buildFrontmatter(rule)
    
    // Transform content (use sparingly!)
    transformedContent := rule.Content
    
    // Example: Add agent-specific notes
    if strings.Contains(rule.Content, "## Quick Start") {
        note := "\n> **MyAI Note**: This feature requires MyAI v2.0+\n\n"
        transformedContent = strings.Replace(
            transformedContent,
            "## Quick Start",
            "## Quick Start"+note,
            1,
        )
    }
    
    content := frontmatter + transformedContent
    return rule.Name + ".md", content, nil
}
```

### Multi-File Output

Some adapters may generate multiple files per rule:

```go
type MultiFileAdapter struct{}

func (a *MultiFileAdapter) Transform(rule genkit.Rule) (string, string, error) {
    // This interface only supports single file output
    // For multi-file, you'd need to extend the interface
    // or handle it in a post-processing step
    
    // For now, return the main file
    return rule.Name + ".md", buildContent(rule), nil
}
```

## Troubleshooting

### Adapter Not Found

**Problem**: `devgen rules --agent myai -w` returns "unknown agent: myai"

**Solution**: Ensure your adapter is registered before the rules command runs:
1. Check that `Register()` is called in an `init()` function
2. Verify the adapter is imported (use blank import if needed)
3. Check that the adapter name matches exactly

### Invalid YAML Frontmatter

**Problem**: Generated rules have syntax errors in frontmatter

**Solution**:
1. Use a YAML library to generate frontmatter
2. Test frontmatter with a YAML validator
3. Escape special characters in string values
4. Use proper indentation for nested structures

### Content Not Preserved

**Problem**: Original rule content is missing or corrupted

**Solution**:
1. Always append rule.Content without modification
2. Don't use string replacement on content
3. Preserve newlines and formatting
4. Test with rules containing special characters

## Reference Implementations

Study the built-in adapters for reference:

- [Kiro Adapter](../genkit/adapter_kiro.go) - Complex frontmatter with arrays
- [CodeBuddy Adapter](../genkit/adapter_codebuddy.go) - Simple frontmatter
- [Cursor Adapter](../genkit/adapter_cursor.go) - Same as CodeBuddy

## Next Steps

- Learn about [RuleTool interface](plugin_EN.md#ai-rules-integration-optional) for providing rules
- Study [AI Rules System](../cmd/devgen/rules/devgen-rules.md) for rule content guidelines
- Explore [built-in adapters](../genkit/) for implementation examples
