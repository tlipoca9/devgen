// Package generator provides golangci-lint integration for devgen.
// It runs golangci-lint and converts its output to devgen diagnostics.
package generator

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/tlipoca9/devgen/genkit"
)

// ToolName is the name of this tool.
const ToolName = "golangcilint"

// golangci-lint config file names.
var configFiles = []string{
	".golangci.yml",
	".golangci.yaml",
	".golangci.toml",
	".golangci.json",
}

// Generator integrates golangci-lint with devgen.
type Generator struct{}

// New creates a new Generator.
func New() *Generator {
	return &Generator{}
}

// Name returns the tool name.
func (g *Generator) Name() string {
	return ToolName
}

// Run is a no-op for this tool since it only provides validation.
func (g *Generator) Run(_ *genkit.Generator, _ *genkit.Logger) error {
	return nil
}

// Config returns the tool configuration.
// This tool has no annotations since it only runs golangci-lint.
func (g *Generator) Config() genkit.ToolConfig {
	return genkit.ToolConfig{
		// No output suffix since we don't generate files
		// No annotations since we don't use devgen annotation syntax
	}
}

// Validate implements genkit.ValidatableTool.
// It runs golangci-lint and returns diagnostics.
func (g *Generator) Validate(gen *genkit.Generator, log *genkit.Logger) []genkit.Diagnostic {
	if len(gen.Packages) == 0 {
		return nil
	}

	// Find the root directory from the first package
	rootDir := findRootDir(gen)
	if rootDir == "" {
		return nil
	}

	// Check if golangci-lint config exists
	if !hasConfigFile(rootDir) {
		return nil
	}

	// Check if golangci-lint is installed
	if !isInstalled() {
		return nil
	}

	// Run golangci-lint
	return runLint(rootDir, log)
}

// findRootDir finds the project root directory from loaded packages.
func findRootDir(gen *genkit.Generator) string {
	if len(gen.Packages) == 0 {
		return ""
	}
	// Use the first package's directory and search upward for config
	pkg := gen.Packages[0]
	dir := pkg.Dir
	for {
		if hasConfigFile(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	// Return the first package's directory if no config found
	return gen.Packages[0].Dir
}

// hasConfigFile checks if any golangci-lint config file exists in the directory.
func hasConfigFile(dir string) bool {
	for _, name := range configFiles {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// isInstalled checks if golangci-lint is installed.
func isInstalled() bool {
	_, err := exec.LookPath("golangci-lint")
	return err == nil
}

// golangciLintOutput represents the JSON output from golangci-lint.
type golangciLintOutput struct {
	Issues []golangciLintIssue `json:"Issues"`
}

type golangciLintIssue struct {
	FromLinter  string                 `json:"FromLinter"`
	Text        string                 `json:"Text"`
	Severity    string                 `json:"Severity"`
	SourceLines []string               `json:"SourceLines"`
	Pos         golangciLintPosition   `json:"Pos"`
	LineRange   *golangciLintLineRange `json:"LineRange"`
}

type golangciLintPosition struct {
	Filename string `json:"Filename"`
	Line     int    `json:"Line"`
	Column   int    `json:"Column"`
}

type golangciLintLineRange struct {
	From int `json:"From"`
	To   int `json:"To"`
}

// runLint runs golangci-lint and returns diagnostics.
func runLint(dir string, log *genkit.Logger) []genkit.Diagnostic {
	// Try v2 format first (--output.json.path), fall back to v1 format (--out-format)
	cmd := exec.Command("golangci-lint", "run", "--output.json.path", "stdout", "./...")
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		// Check if it's a flag error (v1 golangci-lint)
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "unknown flag") || strings.Contains(stderr, "--output.json.path") {
				// Try v1 format
				return runLintV1(dir, log)
			}
			// golangci-lint returns non-zero exit code when issues are found
			// Use stderr if no stdout
			if len(output) == 0 {
				output = exitErr.Stderr
			}
		}
		if len(output) == 0 {
			log.Warn("golangci-lint failed: %v", err)
			return nil
		}
	}

	return parseOutput(output, log)
}

// runLintV1 runs golangci-lint with v1 format flags.
func runLintV1(dir string, log *genkit.Logger) []genkit.Diagnostic {
	cmd := exec.Command("golangci-lint", "run", "--out-format", "json", "./...")
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if len(output) == 0 {
				output = exitErr.Stderr
			}
		}
		if len(output) == 0 {
			log.Warn("golangci-lint failed: %v", err)
			return nil
		}
	}

	return parseOutput(output, log)
}

// parseOutput parses golangci-lint JSON output and returns diagnostics.
func parseOutput(output []byte, log *genkit.Logger) []genkit.Diagnostic {
	// golangci-lint v2 may append text after JSON (e.g., "3 issues:\n* gci: 2")
	// Find the end of JSON object and truncate
	jsonEnd := findJSONEnd(output)
	if jsonEnd > 0 {
		output = output[:jsonEnd]
	}

	var result golangciLintOutput
	if err := json.Unmarshal(output, &result); err != nil {
		log.Warn("failed to parse golangci-lint output: %v", err)
		return nil
	}

	var diagnostics []genkit.Diagnostic
	for _, issue := range result.Issues {
		severity := genkit.DiagnosticWarning
		if issue.Severity == "error" {
			severity = genkit.DiagnosticError
		}

		d := genkit.Diagnostic{
			Severity: severity,
			Message:  issue.Text,
			File:     issue.Pos.Filename,
			Line:     issue.Pos.Line,
			Column:   issue.Pos.Column,
			Tool:     ToolName,
			Code:     issue.FromLinter,
		}

		if issue.LineRange != nil {
			d.EndLine = issue.LineRange.To
		}

		diagnostics = append(diagnostics, d)
	}

	return diagnostics
}

// findJSONEnd finds the end of a JSON object in the output.
// Returns the position after the closing brace, or 0 if not found.
func findJSONEnd(data []byte) int {
	depth := 0
	inString := false
	escape := false

	for i, b := range data {
		if escape {
			escape = false
			continue
		}
		if inString {
			switch b {
			case '\\':
				escape = true
			case '"':
				inString = false
			}
			continue
		}
		switch b {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return 0
}
