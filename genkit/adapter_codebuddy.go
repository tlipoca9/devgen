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
	// Build YAML frontmatter
	var frontmatter string
	if len(rule.Globs) > 0 {
		frontmatter = fmt.Sprintf(`---
description: %s
globs: %s
alwaysApply: %t
---

`, rule.Description, formatGlobsComma(rule.Globs), rule.AlwaysApply)
	} else {
		frontmatter = fmt.Sprintf(`---
description: %s
alwaysApply: %t
---

`, rule.Description, rule.AlwaysApply)
	}

	// Combine frontmatter with content
	content := frontmatter + rule.Content

	// Generate filename with .mdc extension for CodeBuddy
	filename := rule.Name + ".mdc"

	return filename, content, nil
}

// formatGlobsComma formats a slice of globs as a comma-separated string.
// Returns empty string if no globs provided.
func formatGlobsComma(globs []string) string {
	if len(globs) == 0 {
		return ""
	}
	return strings.Join(globs, ", ")
}
