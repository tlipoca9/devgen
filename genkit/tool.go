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
