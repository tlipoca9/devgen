package genkit

// Tool is the interface that code generation tools must implement.
// It provides a unified way to run code generators.
type Tool interface {
	// Name returns the tool name (e.g., "enumgen", "validategen").
	Name() string

	// Run processes all packages and generates code.
	// It should handle logging internally.
	Run(gen *Generator, log *Logger) error
}

// ConfigurableTool extends Tool with self-describing configuration.
// Implement this interface to provide annotation metadata for VSCode extension
// and CLI integration without requiring a separate devgen.toml file.
type ConfigurableTool interface {
	Tool

	// Config returns the tool's configuration including annotations metadata.
	// This is used by VSCode extension for syntax highlighting and auto-completion,
	// and by CLI for validation.
	Config() ToolConfig
}

// ValidatableTool extends Tool with validation capability for dry-run mode.
// Implement this interface to provide detailed diagnostics (errors/warnings)
// that can be displayed in IDEs without generating files.
type ValidatableTool interface {
	Tool

	// Validate checks for errors without generating files.
	// Returns diagnostics (errors/warnings) found during validation.
	// This is called in dry-run mode to provide IDE integration.
	Validate(gen *Generator, log *Logger) []Diagnostic
}

// RuleTool extends Tool with AI rules generation capability.
// Implement this interface to provide AI-friendly documentation
// that can be used by AI coding assistants (CodeBuddy, Cursor, Copilot, etc.)
type RuleTool interface {
	Tool

	// Rules returns the AI rules for this tool.
	// Each rule should be detailed, step-by-step documentation
	// with plenty of examples to help AI assistants understand
	// how to use this tool correctly.
	Rules() []Rule
}

// Rule represents an AI rule configuration.
// This structure is designed to be compatible with multiple AI agents:
// - CodeBuddy: .codebuddy/rules/*.mdc
// - Cursor: .cursor/rules/*.mdc
// - Kiro: .kiro/steering/*.md
// - GitHub Copilot: .github/copilot-instructions.md
type Rule struct {
	// Name is the rule file name (without extension).
	// Example: "enumgen", "validategen-basics"
	Name string

	// Description is a short description of what this rule covers.
	// Used by AI agents to decide whether to include this rule.
	// Example: "Go enum code generation with enumgen"
	Description string

	// Globs specifies file patterns that trigger this rule.
	// When files matching these patterns are referenced, the rule is auto-attached.
	// Example: []string{"*.go", "**/*_enum.go"}
	// Empty means the rule won't be auto-attached by file patterns.
	Globs []string

	// AlwaysApply indicates whether this rule should always be included.
	// If true, the rule is always in context (like Cursor's "Always" type).
	// If false, the rule is included based on Globs or manual reference.
	AlwaysApply bool

	// Content is the actual rule content in Markdown format.
	// This should be detailed, step-by-step documentation with examples.
	// Write it as if the reader knows nothing - be explicit and thorough.
	Content string
}
