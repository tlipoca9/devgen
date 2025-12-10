package genkit

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRulesFromDir(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create test rule files
	rule1 := `---
description: Test rule 1
globs:
  - "**/*.go"
  - "**/*.ts"
alwaysApply: false
---

# Test Rule 1

This is test rule 1 content.
`
	rule2 := `---
description: Test rule 2
globs:
  - "**/*.md"
alwaysApply: true
---

# Test Rule 2

This is test rule 2 content.
`

	if err := os.WriteFile(filepath.Join(tmpDir, "rule1.md"), []byte(rule1), 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "rule2.md"), []byte(rule2), 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	// Create a non-md file that should be ignored
	if err := os.WriteFile(filepath.Join(tmpDir, "ignore.txt"), []byte("ignored"), 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	// Load rules
	rules, err := LoadRulesFromDir(tmpDir)
	if err != nil {
		t.Fatalf("LoadRulesFromDir error: %v", err)
	}

	if len(rules) != 2 {
		t.Errorf("LoadRulesFromDir returned %d rules, want 2", len(rules))
	}

	// Check rules content
	ruleMap := make(map[string]Rule)
	for _, r := range rules {
		ruleMap[r.Name] = r
	}

	// Check rule1
	r1, ok := ruleMap["rule1"]
	if !ok {
		t.Error("Missing rule1")
	} else {
		if r1.Description != "Test rule 1" {
			t.Errorf("rule1.Description = %q, want %q", r1.Description, "Test rule 1")
		}
		if len(r1.Globs) != 2 {
			t.Errorf("rule1.Globs length = %d, want 2", len(r1.Globs))
		}
		if r1.AlwaysApply != false {
			t.Errorf("rule1.AlwaysApply = %v, want false", r1.AlwaysApply)
		}
		if r1.Content != "# Test Rule 1\n\nThis is test rule 1 content." {
			t.Errorf("rule1.Content = %q", r1.Content)
		}
	}

	// Check rule2
	r2, ok := ruleMap["rule2"]
	if !ok {
		t.Error("Missing rule2")
	} else {
		if r2.Description != "Test rule 2" {
			t.Errorf("rule2.Description = %q, want %q", r2.Description, "Test rule 2")
		}
		if r2.AlwaysApply != true {
			t.Errorf("rule2.AlwaysApply = %v, want true", r2.AlwaysApply)
		}
	}
}

func TestLoadRulesFromDir_NonExistent(t *testing.T) {
	rules, err := LoadRulesFromDir("/nonexistent/path")
	if err != nil {
		t.Errorf("LoadRulesFromDir should not error for non-existent dir, got: %v", err)
	}
	if rules != nil {
		t.Errorf("LoadRulesFromDir should return nil for non-existent dir, got: %v", rules)
	}
}

func TestLoadRulesFromDir_Empty(t *testing.T) {
	tmpDir := t.TempDir()

	rules, err := LoadRulesFromDir(tmpDir)
	if err != nil {
		t.Errorf("LoadRulesFromDir error: %v", err)
	}
	if len(rules) != 0 {
		t.Errorf("LoadRulesFromDir should return empty for empty dir, got %d rules", len(rules))
	}
}

func TestLoadRuleFromFile_NoFrontmatter(t *testing.T) {
	tmpDir := t.TempDir()
	content := `# Simple Rule

This rule has no frontmatter.
`
	filePath := filepath.Join(tmpDir, "simple.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	rule, err := LoadRuleFromFile(filePath)
	if err != nil {
		t.Fatalf("LoadRuleFromFile error: %v", err)
	}

	if rule.Name != "simple" {
		t.Errorf("rule.Name = %q, want %q", rule.Name, "simple")
	}
	if rule.Description != "" {
		t.Errorf("rule.Description = %q, want empty", rule.Description)
	}
	if rule.Content != "# Simple Rule\n\nThis rule has no frontmatter." {
		t.Errorf("rule.Content = %q", rule.Content)
	}
}

func TestLoadRuleFromFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "empty.md")
	if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	rule, err := LoadRuleFromFile(filePath)
	if err != nil {
		t.Fatalf("LoadRuleFromFile error: %v", err)
	}

	if rule.Name != "empty" {
		t.Errorf("rule.Name = %q, want %q", rule.Name, "empty")
	}
	if rule.Content != "" {
		t.Errorf("rule.Content = %q, want empty", rule.Content)
	}
}

func TestLoadRuleFromFile_GlobsAsString(t *testing.T) {
	tmpDir := t.TempDir()
	// Test with globs as a single string (some users might write it this way)
	content := `---
description: Single glob test
globs:
  - "**/*.go"
alwaysApply: false
---

# Content
`
	filePath := filepath.Join(tmpDir, "single-glob.md")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	rule, err := LoadRuleFromFile(filePath)
	if err != nil {
		t.Fatalf("LoadRuleFromFile error: %v", err)
	}

	if len(rule.Globs) != 1 || rule.Globs[0] != "**/*.go" {
		t.Errorf("rule.Globs = %v, want [**/*.go]", rule.Globs)
	}
}
