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

	// Build YAML frontmatter
	var frontmatter string
	if rule.AlwaysApply {
		// Always mode: no fileMatchPattern needed
		if len(rule.Globs) > 0 {
			frontmatter = fmt.Sprintf(`---
description: %s
inclusion: %s
fileMatchPattern: %s
---

`, rule.Description, inclusion, formatPatternsYAML(rule.Globs))
		} else {
			frontmatter = fmt.Sprintf(`---
description: %s
inclusion: %s
---

`, rule.Description, inclusion)
		}
	} else {
		// FileMatch mode: need fileMatchPattern, default to match all files
		patterns := rule.Globs
		if len(patterns) == 0 {
			patterns = []string{"**/*"}
		}
		frontmatter = fmt.Sprintf(`---
description: %s
inclusion: %s
fileMatchPattern: %s
---

`, rule.Description, inclusion, formatPatternsYAML(patterns))
	}

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
