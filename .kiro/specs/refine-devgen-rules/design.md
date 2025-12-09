# Design Document

## Overview

This design implements a flexible adapter system for devgen AI rules that allows a single source of truth (English markdown files in `cmd/*/rules/`) to be transformed into agent-specific formats (Kiro, CodeBuddy, Cursor, etc.) with appropriate frontmatter and formatting.

The design follows a "source → adapter → output" pattern where:
1. Source rules are maintained in English in `cmd/*/rules/` directories
2. Adapter functions transform rules based on agent requirements
3. Output files are generated in agent-specific locations with proper formatting

## Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     devgen rules Command                     │
│                                                              │
│  ┌────────────┐      ┌──────────────┐      ┌─────────────┐ │
│  │   Parser   │─────▶│   Adapter    │─────▶│   Writer    │ │
│  │            │      │   Registry   │      │             │ │
│  └────────────┘      └──────────────┘      └─────────────┘ │
│        │                     │                     │        │
│        ▼                     ▼                     ▼        │
│  ┌────────────┐      ┌──────────────┐      ┌─────────────┐ │
│  │   Source   │      │ Kiro Adapter │      │ .kiro/      │ │
│  │   Rules    │      │CodeBuddy Adp │      │ .codebuddy/ │ │
│  │ (cmd/*/    │      │ Cursor Adp   │      │ .cursor/    │ │
│  │  rules/)   │      └──────────────┘      └─────────────┘ │
│  └────────────┘                                             │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

1. User runs `devgen rules --agent kiro -w`
2. Command collects all RuleTool implementations from loaded tools
3. For each rule, the appropriate adapter transforms it
4. Writer creates files in agent-specific directory with proper formatting

## Components and Interfaces

### 1. Agent Adapter Interface

```go
// AgentAdapter transforms rules for a specific AI assistant
type AgentAdapter interface {
    // Name returns the agent name (e.g., "kiro", "codebuddy")
    Name() string
    
    // OutputDir returns the directory where rules should be written
    OutputDir() string
    
    // Transform converts a genkit.Rule to agent-specific format
    Transform(rule genkit.Rule) (filename string, content string, err error)
}
```

### 2. Kiro Adapter Implementation

```go
type KiroAdapter struct{}

func (k *KiroAdapter) Name() string {
    return "kiro"
}

func (k *KiroAdapter) OutputDir() string {
    return ".kiro/steering"
}

func (k *KiroAdapter) Transform(rule genkit.Rule) (string, string, error) {
    // Determine inclusion type based on AlwaysApply
    inclusion := "fileMatch"
    if rule.AlwaysApply {
        inclusion = "always"
    }
    
    // Convert Globs to fileMatchPattern
    patterns := rule.Globs
    if len(patterns) == 0 {
        patterns = []string{"**/*.go"}
    }
    
    // Build YAML frontmatter
    frontmatter := fmt.Sprintf(`---
inclusion: %s
fileMatchPattern: %s
---

`, inclusion, formatPatterns(patterns))
    
    // Combine frontmatter with content
    content := frontmatter + rule.Content
    filename := rule.Name + ".md"
    
    return filename, content, nil
}
```

### 3. Adapter Registry

```go
// AdapterRegistry manages available agent adapters
type AdapterRegistry struct {
    adapters map[string]AgentAdapter
}

func NewAdapterRegistry() *AdapterRegistry {
    registry := &AdapterRegistry{
        adapters: make(map[string]AgentAdapter),
    }
    
    // Register built-in adapters
    registry.Register(&KiroAdapter{})
    registry.Register(&CodeBuddyAdapter{})
    registry.Register(&CursorAdapter{})
    
    return registry
}

func (r *AdapterRegistry) Register(adapter AgentAdapter) {
    r.adapters[adapter.Name()] = adapter
}

func (r *AdapterRegistry) Get(name string) (AgentAdapter, bool) {
    adapter, ok := r.adapters[name]
    return adapter, ok
}

func (r *AdapterRegistry) List() []string {
    names := make([]string, 0, len(r.adapters))
    for name := range r.adapters {
        names = append(names, name)
    }
    sort.Strings(names)
    return names
}
```

### 4. Rules Command Structure

```go
// RulesCommand handles the 'devgen rules' subcommand
type RulesCommand struct {
    registry *AdapterRegistry
    gen      *genkit.Generator
    log      *genkit.Logger
}

func (c *RulesCommand) Execute(agent string, write bool) error {
    // Get adapter
    adapter, ok := c.registry.Get(agent)
    if !ok {
        return fmt.Errorf("unknown agent: %s", agent)
    }
    
    // Collect rules from all tools
    rules := c.collectRules()
    
    if !write {
        // Preview mode: print to stdout
        return c.preview(adapter, rules)
    }
    
    // Write mode: create files
    return c.writeRules(adapter, rules)
}

func (c *RulesCommand) collectRules() []genkit.Rule {
    var allRules []genkit.Rule
    
    // Iterate through all loaded tools
    for _, tool := range c.gen.Tools {
        if ruleTool, ok := tool.(genkit.RuleTool); ok {
            rules := ruleTool.Rules()
            allRules = append(allRules, rules...)
        }
    }
    
    return allRules
}

func (c *RulesCommand) writeRules(adapter AgentAdapter, rules []genkit.Rule) error {
    outputDir := adapter.OutputDir()
    
    // Create output directory
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return err
    }
    
    // Transform and write each rule
    for _, rule := range rules {
        filename, content, err := adapter.Transform(rule)
        if err != nil {
            return fmt.Errorf("transform rule %s: %w", rule.Name, err)
        }
        
        filepath := filepath.Join(outputDir, filename)
        if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
            return fmt.Errorf("write file %s: %w", filepath, err)
        }
        
        c.log.Done("Generated %s", filepath)
    }
    
    return nil
}
```

## Data Models

### Rule Structure (from genkit)

```go
type Rule struct {
    Name        string   // Rule filename without extension
    Description string   // Brief description for AI context loading
    Globs       []string // File patterns that trigger auto-loading
    AlwaysApply bool     // Whether to always include in context
    Content     string   // Full markdown documentation
}
```

### Kiro Frontmatter Format

```yaml
---
inclusion: fileMatch | always
fileMatchPattern: ['**/*.go', '**/devgen.toml']
---
```

### CodeBuddy Frontmatter Format

```yaml
---
description: Brief description
globs: **/*.go
alwaysApply: false
---
```

### Cursor Frontmatter Format

```yaml
---
description: Brief description
globs: **/*.go
alwaysApply: false
---
```

## Error Handling

### Error Scenarios

1. **Unknown Agent**: Return error with list of available agents
2. **Directory Creation Failure**: Return error with permission details
3. **File Write Failure**: Return error with file path and reason
4. **Transform Failure**: Return error with rule name and transformation issue
5. **No Rules Found**: Log warning but don't error (valid for projects without RuleTool implementations)

### Error Messages

```go
// Unknown agent
fmt.Errorf("unknown agent %q, available agents: %s", agent, strings.Join(available, ", "))

// Directory creation
fmt.Errorf("failed to create directory %s: %w", dir, err)

// File write
fmt.Errorf("failed to write rule file %s: %w", filepath, err)

// Transform
fmt.Errorf("failed to transform rule %s for agent %s: %w", rule.Name, agent, err)
```

## Testing Strategy

### Unit Tests

1. **Adapter Tests**
   - Test Kiro adapter frontmatter generation
   - Test pattern formatting (single vs multiple patterns)
   - Test inclusion type selection (always vs fileMatch)
   - Test content preservation

2. **Registry Tests**
   - Test adapter registration
   - Test adapter retrieval
   - Test listing all adapters
   - Test duplicate registration handling

3. **Command Tests**
   - Test rule collection from tools
   - Test preview mode output
   - Test write mode file creation
   - Test error handling for unknown agents

### Integration Tests

1. **End-to-End Rule Generation**
   - Load test project with RuleTool implementations
   - Generate rules for each agent
   - Verify file creation and content
   - Verify frontmatter correctness

2. **Multi-Tool Scenarios**
   - Test with multiple tools implementing RuleTool
   - Verify all rules are collected
   - Verify no duplicate rule names

### Test Data

Create test fixtures in `testdata/rules/`:
- Sample RuleTool implementations
- Expected output for each agent
- Edge cases (empty globs, special characters in content)

## Implementation Notes

### File Organization

```
cmd/devgen/
├── main.go
├── rules/
│   ├── devgen.md              # Source rules (English)
│   ├── devgen-plugin.md
│   ├── devgen-genkit.md
│   ├── devgen-rules.md
│   └── embed.go
└── rules_command.go           # New file for rules command

genkit/
├── tool.go                    # Existing RuleTool interface
├── adapter.go                 # New file for adapter interface
├── adapter_kiro.go            # New file for Kiro adapter
├── adapter_codebuddy.go       # New file for CodeBuddy adapter
├── adapter_cursor.go          # New file for Cursor adapter
└── adapter_registry.go        # New file for adapter registry
```

### Backward Compatibility

- Existing `devgen rules` command behavior is preserved
- CodeBuddy and Cursor adapters maintain current frontmatter format
- Source rules in `cmd/*/rules/` remain unchanged in structure
- RuleTool interface remains unchanged

### Performance Considerations

- Rule collection is done once per command execution
- File writes are sequential (acceptable for small number of rules)
- No caching needed (rules generation is infrequent)

### Future Extensibility

1. **Custom Adapters**: Users can register custom adapters for proprietary AI assistants
2. **Rule Validation**: Add validation step before transformation
3. **Template System**: Allow adapters to use templates for complex transformations
4. **Localization**: Support multiple languages by adding language parameter to adapters
5. **Rule Merging**: Support merging rules from multiple sources

## Migration Plan

### Phase 1: Refine Source Rules
1. Update all rules in `cmd/*/rules/` to English
2. Remove broad `go generate` recommendations
3. Ensure consistent structure across all rules
4. Add comprehensive examples and error sections

### Phase 2: Implement Adapter System
1. Create adapter interface and registry
2. Implement Kiro adapter
3. Implement CodeBuddy adapter (maintain current format)
4. Implement Cursor adapter (maintain current format)

### Phase 3: Update Rules Command
1. Integrate adapter registry into rules command
2. Update `--list-agents` to use registry
3. Update rule generation to use adapters
4. Add tests for all adapters

### Phase 4: Generate and Validate
1. Run `devgen rules --agent kiro -w` to generate Kiro rules
2. Run `devgen rules --agent codebuddy -w` to verify CodeBuddy rules
3. Run `devgen rules --agent cursor -w` to verify Cursor rules
4. Validate all generated files have correct frontmatter
5. Test with actual AI assistants

## Open Questions

1. Should we support rule versioning to track changes over time?
2. Should adapters support custom frontmatter fields via configuration?
3. Should we add a validation command to check rule quality?
4. Should we support rule inheritance or composition?

## Dependencies

- No new external dependencies required
- Uses existing genkit package
- Uses standard library (os, path/filepath, fmt, strings, sort)
