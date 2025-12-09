package genkit

import (
	"fmt"
	"strings"
)

// KiroAdapter transforms rules for Kiro AI assistant.
// Kiro uses YAML frontmatter with 'inclusion' and 'fileMatchPattern' fields.
//
// Frontmatter format:
//
//	---
//	inclusion: fileMatch | always
//	fileMatchPattern: ['**/*.go', '**/devgen.toml']
//	---
//
// Inclusion types:
//   - "always": Rule is always included in context
//   - "fileMatch": Rule is included when files match fileMatchPattern
type KiroAdapter struct{}

// Name returns "kiro".
func (k *KiroAdapter) Name() string {
	return "kiro"
}

// OutputDir returns ".kiro/steering".
func (k *KiroAdapter) OutputDir() string {
	return ".kiro/steering"
}

// Transform converts a Rule to Kiro format with YAML frontmatter.
// It determines the inclusion type based on AlwaysApply and formats
// the Globs as a YAML array for fileMatchPattern.
func (k *KiroAdapter) Transform(rule Rule) (string, string, error) {
	// Determine inclusion type
	inclusion := "fileMatch"
	if rule.AlwaysApply {
		inclusion = "always"
	}

	// Use rule globs or default to Go files
	patterns := rule.Globs
	if len(patterns) == 0 {
		patterns = []string{"**/*.go"}
	}

	// Format patterns as YAML array
	patternStr := formatPatternsYAML(patterns)

	// Build YAML frontmatter
	frontmatter := fmt.Sprintf(`---
inclusion: %s
fileMatchPattern: %s
---

`, inclusion, patternStr)

	// Combine frontmatter with content
	content := frontmatter + rule.Content

	// Generate filename
	filename := rule.Name + ".md"

	return filename, content, nil
}

// formatPatternsYAML formats a slice of patterns as a YAML array.
// Examples:
//   - Single pattern: ['**/*.go']
//   - Multiple patterns: ['**/*.go', '**/devgen.toml']
func formatPatternsYAML(patterns []string) string {
	if len(patterns) == 0 {
		return "[]"
	}

	// Quote each pattern and join with ", "
	quoted := make([]string, len(patterns))
	for i, p := range patterns {
		quoted[i] = fmt.Sprintf("'%s'", p)
	}

	return "[" + strings.Join(quoted, ", ") + "]"
}
