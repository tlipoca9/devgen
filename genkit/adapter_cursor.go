package genkit

import "fmt"

// CursorAdapter transforms rules for Cursor AI assistant.
// Cursor uses YAML frontmatter with 'description', 'globs', and 'alwaysApply' fields,
// similar to CodeBuddy but with .mdc file extension.
//
// Frontmatter format:
//
//	---
//	description: Brief description of the rule
//	globs: **/*.go, **/devgen.toml
//	alwaysApply: false
//	---
type CursorAdapter struct{}

// Name returns "cursor".
func (c *CursorAdapter) Name() string {
	return "cursor"
}

// OutputDir returns ".cursor/rules".
func (c *CursorAdapter) OutputDir() string {
	return ".cursor/rules"
}

// Transform converts a Rule to Cursor format with YAML frontmatter.
// It maps Rule fields directly to Cursor's frontmatter format.
// Note: Cursor uses .mdc extension instead of .md.
func (c *CursorAdapter) Transform(rule Rule) (string, string, error) {
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

	// Generate filename with .mdc extension
	filename := rule.Name + ".mdc"

	return filename, content, nil
}
