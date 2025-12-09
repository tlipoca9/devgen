package genkit

// AgentAdapter transforms rules for a specific AI assistant.
// Different AI assistants (Kiro, CodeBuddy, Cursor, etc.) require different
// frontmatter formats and directory structures. Adapters handle these differences
// by converting the generic Rule structure into agent-specific formats.
//
// Example usage:
//
//	adapter := &KiroAdapter{}
//	filename, content, err := adapter.Transform(rule)
//	if err != nil {
//	    return err
//	}
//	filepath := filepath.Join(adapter.OutputDir(), filename)
//	os.WriteFile(filepath, []byte(content), 0644)
type AgentAdapter interface {
	// Name returns the agent identifier (e.g., "kiro", "codebuddy", "cursor").
	// This name is used in CLI commands like `devgen rules --agent kiro`.
	Name() string

	// OutputDir returns the directory path where rules should be written.
	// The path is relative to the project root.
	// Examples:
	//   - Kiro: ".kiro/steering"
	//   - CodeBuddy: ".codebuddy/rules"
	//   - Cursor: ".cursor/rules"
	OutputDir() string

	// Transform converts a generic Rule into agent-specific format.
	// It returns:
	//   - filename: the output filename (e.g., "enumgen.md")
	//   - content: the complete file content with frontmatter and markdown
	//   - error: any transformation error
	//
	// The transformation typically involves:
	//   1. Converting Rule fields to agent-specific frontmatter
	//   2. Formatting the frontmatter as YAML
	//   3. Combining frontmatter with the rule content
	Transform(rule Rule) (filename string, content string, err error)
}
