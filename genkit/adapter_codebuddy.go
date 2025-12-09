package genkit

import (
	"fmt"
	"strings"
)

// CodeBuddyAdapter transforms rules for CodeBuddy AI assistant.
// CodeBuddy uses YAML frontmatter with 'description', 'globs', and 'alwaysApply' fields.
//
// Frontmatter format:
//
//	---
//	description: Brief description of the rule
//	globs: **/*.go, **/devgen.toml
//	alwaysApply: false
//	---
type CodeBuddyAdapter struct{}

// Name returns "codebuddy".
func (c *CodeBuddyAdapter) Name() string {
	return "codebuddy"
}

// OutputDir returns ".codebuddy/rules".
func (c *CodeBuddyAdapter) OutputDir() string {
	return ".codebuddy/rules"
}

// Transform converts a Rule to CodeBuddy format with YAML frontmatter.
// It maps Rule fields directly to CodeBuddy's frontmatter format.
func (c *CodeBuddyAdapter) Transform(rule Rule) (string, string, error) {
	// Format globs as comma-separated string
	globsStr := formatGlobsComma(rule.Globs)

	// Build YAML frontmatter
	frontmatter := fmt.Sprintf(`---
description: %s
globs: %s
alwaysApply: %t
---

`, rule.Description, globsStr, rule.AlwaysApply)

	// Combine frontmatter with content
	content := frontmatter + rule.Content

	// Generate filename with .mdc extension for CodeBuddy
	filename := rule.Name + ".mdc"

	return filename, content, nil
}

// formatGlobsComma formats a slice of globs as a comma-separated string.
// Examples:
//   - Single glob: "**/*.go"
//   - Multiple globs: "**/*.go, **/devgen.toml"
//   - Empty: "**/*.go" (default)
func formatGlobsComma(globs []string) string {
	if len(globs) == 0 {
		return "**/*.go"
	}
	return strings.Join(globs, ", ")
}
