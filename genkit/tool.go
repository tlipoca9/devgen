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
