package genkit

import (
	"strings"
	"testing"
)

// Test data
var testRule = Rule{
	Name:        "test-rule",
	Description: "Test rule for unit testing",
	Globs:       []string{"**/*.go", "**/test.toml"},
	AlwaysApply: false,
	Content:     "# Test Rule\n\nThis is test content.",
}

var testRuleAlways = Rule{
	Name:        "always-rule",
	Description: "Always apply test rule",
	Globs:       []string{"**/*.go"},
	AlwaysApply: true,
	Content:     "# Always Rule\n\nThis rule always applies.",
}

var testRuleNoGlobs = Rule{
	Name:        "no-globs",
	Description: "Rule without globs",
	Globs:       []string{},
	AlwaysApply: false,
	Content:     "# No Globs Rule\n\nThis rule has no globs.",
}

// TestKiroAdapter tests the Kiro adapter
func TestKiroAdapter(t *testing.T) {
	adapter := &KiroAdapter{}

	t.Run("Name", func(t *testing.T) {
		if got := adapter.Name(); got != "kiro" {
			t.Errorf("Name() = %q, want %q", got, "kiro")
		}
	})

	t.Run("OutputDir", func(t *testing.T) {
		if got := adapter.OutputDir(); got != ".kiro/steering" {
			t.Errorf("OutputDir() = %q, want %q", got, ".kiro/steering")
		}
	})

	t.Run("Transform_FileMatch", func(t *testing.T) {
		filename, content, err := adapter.Transform(testRule)
		if err != nil {
			t.Fatalf("Transform() error = %v", err)
		}

		if filename != "test-rule.md" {
			t.Errorf("filename = %q, want %q", filename, "test-rule.md")
		}

		// Check frontmatter
		if !strings.Contains(content, "inclusion: fileMatch") {
			t.Error("content missing 'inclusion: fileMatch'")
		}
		if !strings.Contains(content, "fileMatchPattern: ['**/*.go', '**/test.toml']") {
			t.Error("content missing correct fileMatchPattern")
		}

		// Check content preservation
		if !strings.Contains(content, "# Test Rule") {
			t.Error("content missing original markdown")
		}
		if !strings.Contains(content, "This is test content.") {
			t.Error("content missing original text")
		}
	})

	t.Run("Transform_Always", func(t *testing.T) {
		filename, content, err := adapter.Transform(testRuleAlways)
		if err != nil {
			t.Fatalf("Transform() error = %v", err)
		}

		if filename != "always-rule.md" {
			t.Errorf("filename = %q, want %q", filename, "always-rule.md")
		}

		if !strings.Contains(content, "inclusion: always") {
			t.Error("content missing 'inclusion: always'")
		}
	})

	t.Run("Transform_NoGlobs", func(t *testing.T) {
		_, content, err := adapter.Transform(testRuleNoGlobs)
		if err != nil {
			t.Fatalf("Transform() error = %v", err)
		}

		// Should default to **/*.go
		if !strings.Contains(content, "fileMatchPattern: ['**/*.go']") {
			t.Error("content missing default fileMatchPattern")
		}
	})
}

// TestCodeBuddyAdapter tests the CodeBuddy adapter
func TestCodeBuddyAdapter(t *testing.T) {
	adapter := &CodeBuddyAdapter{}

	t.Run("Name", func(t *testing.T) {
		if got := adapter.Name(); got != "codebuddy" {
			t.Errorf("Name() = %q, want %q", got, "codebuddy")
		}
	})

	t.Run("OutputDir", func(t *testing.T) {
		if got := adapter.OutputDir(); got != ".codebuddy/rules" {
			t.Errorf("OutputDir() = %q, want %q", got, ".codebuddy/rules")
		}
	})

	t.Run("Transform", func(t *testing.T) {
		filename, content, err := adapter.Transform(testRule)
		if err != nil {
			t.Fatalf("Transform() error = %v", err)
		}

		// CodeBuddy uses .mdc extension
		if filename != "test-rule.mdc" {
			t.Errorf("filename = %q, want %q", filename, "test-rule.mdc")
		}

		// Check frontmatter
		if !strings.Contains(content, "description: Test rule for unit testing") {
			t.Error("content missing description")
		}
		if !strings.Contains(content, "globs: **/*.go, **/test.toml") {
			t.Error("content missing correct globs")
		}
		if !strings.Contains(content, "alwaysApply: false") {
			t.Error("content missing alwaysApply")
		}

		// Check content preservation
		if !strings.Contains(content, "# Test Rule") {
			t.Error("content missing original markdown")
		}
	})

	t.Run("Transform_AlwaysApply", func(t *testing.T) {
		_, content, err := adapter.Transform(testRuleAlways)
		if err != nil {
			t.Fatalf("Transform() error = %v", err)
		}

		if !strings.Contains(content, "alwaysApply: true") {
			t.Error("content missing 'alwaysApply: true'")
		}
	})

	t.Run("Transform_NoGlobs", func(t *testing.T) {
		_, content, err := adapter.Transform(testRuleNoGlobs)
		if err != nil {
			t.Fatalf("Transform() error = %v", err)
		}

		// Should default to **/*.go
		if !strings.Contains(content, "globs: **/*.go") {
			t.Error("content missing default globs")
		}
	})
}

// TestCursorAdapter tests the Cursor adapter
func TestCursorAdapter(t *testing.T) {
	adapter := &CursorAdapter{}

	t.Run("Name", func(t *testing.T) {
		if got := adapter.Name(); got != "cursor" {
			t.Errorf("Name() = %q, want %q", got, "cursor")
		}
	})

	t.Run("OutputDir", func(t *testing.T) {
		if got := adapter.OutputDir(); got != ".cursor/rules" {
			t.Errorf("OutputDir() = %q, want %q", got, ".cursor/rules")
		}
	})

	t.Run("Transform", func(t *testing.T) {
		filename, content, err := adapter.Transform(testRule)
		if err != nil {
			t.Fatalf("Transform() error = %v", err)
		}

		// Cursor uses .mdc extension
		if filename != "test-rule.mdc" {
			t.Errorf("filename = %q, want %q", filename, "test-rule.mdc")
		}

		// Check frontmatter (same as CodeBuddy)
		if !strings.Contains(content, "description: Test rule for unit testing") {
			t.Error("content missing description")
		}
		if !strings.Contains(content, "globs: **/*.go, **/test.toml") {
			t.Error("content missing correct globs")
		}
		if !strings.Contains(content, "alwaysApply: false") {
			t.Error("content missing alwaysApply")
		}

		// Check content preservation
		if !strings.Contains(content, "# Test Rule") {
			t.Error("content missing original markdown")
		}
	})
}

// TestAdapterRegistry tests the adapter registry
func TestAdapterRegistry(t *testing.T) {
	t.Run("NewAdapterRegistry", func(t *testing.T) {
		registry := NewAdapterRegistry()

		// Check built-in adapters are registered
		names := registry.List()
		expected := []string{"codebuddy", "cursor", "kiro"}

		if len(names) != len(expected) {
			t.Errorf("List() returned %d adapters, want %d", len(names), len(expected))
		}

		for i, name := range expected {
			if names[i] != name {
				t.Errorf("List()[%d] = %q, want %q", i, names[i], name)
			}
		}
	})

	t.Run("Get", func(t *testing.T) {
		registry := NewAdapterRegistry()

		adapter, ok := registry.Get("kiro")
		if !ok {
			t.Error("Get(kiro) returned false, want true")
		}
		if adapter.Name() != "kiro" {
			t.Errorf("adapter.Name() = %q, want %q", adapter.Name(), "kiro")
		}

		_, ok = registry.Get("unknown")
		if ok {
			t.Error("Get(unknown) returned true, want false")
		}
	})

	t.Run("Register", func(t *testing.T) {
		registry := NewAdapterRegistry()

		// Register custom adapter
		custom := &testAdapter{name: "custom"}
		registry.Register(custom)

		adapter, ok := registry.Get("custom")
		if !ok {
			t.Error("Get(custom) returned false after Register")
		}
		if adapter.Name() != "custom" {
			t.Errorf("adapter.Name() = %q, want %q", adapter.Name(), "custom")
		}

		// Check it appears in list
		names := registry.List()
		found := false
		for _, name := range names {
			if name == "custom" {
				found = true
				break
			}
		}
		if !found {
			t.Error("custom adapter not found in List()")
		}
	})

	t.Run("List_Sorted", func(t *testing.T) {
		registry := NewAdapterRegistry()

		names := registry.List()
		for i := 1; i < len(names); i++ {
			if names[i-1] >= names[i] {
				t.Errorf("List() not sorted: %q >= %q", names[i-1], names[i])
			}
		}
	})
}

// testAdapter is a mock adapter for testing
type testAdapter struct {
	name string
}

func (t *testAdapter) Name() string {
	return t.name
}

func (t *testAdapter) OutputDir() string {
	return ".test/rules"
}

func (t *testAdapter) Transform(rule Rule) (string, string, error) {
	return rule.Name + ".test", rule.Content, nil
}

// TestFormatPatternsYAML tests the pattern formatting helper
func TestFormatPatternsYAML(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		want     string
	}{
		{
			name:     "empty",
			patterns: []string{},
			want:     "[]",
		},
		{
			name:     "single",
			patterns: []string{"**/*.go"},
			want:     "['**/*.go']",
		},
		{
			name:     "multiple",
			patterns: []string{"**/*.go", "**/test.toml", "**/*.md"},
			want:     "['**/*.go', '**/test.toml', '**/*.md']",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatPatternsYAML(tt.patterns)
			if got != tt.want {
				t.Errorf("formatPatternsYAML() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestFormatGlobsComma tests the globs formatting helper
func TestFormatGlobsComma(t *testing.T) {
	tests := []struct {
		name  string
		globs []string
		want  string
	}{
		{
			name:  "empty",
			globs: []string{},
			want:  "**/*.go",
		},
		{
			name:  "single",
			globs: []string{"**/*.go"},
			want:  "**/*.go",
		},
		{
			name:  "multiple",
			globs: []string{"**/*.go", "**/test.toml"},
			want:  "**/*.go, **/test.toml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatGlobsComma(tt.globs)
			if got != tt.want {
				t.Errorf("formatGlobsComma() = %q, want %q", got, tt.want)
			}
		})
	}
}
