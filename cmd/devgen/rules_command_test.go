package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tlipoca9/devgen/genkit"
)

// TestRulesCommand_ListAgents tests listing available agents
func TestRulesCommand_ListAgents(t *testing.T) {
	log := genkit.NewLogger()
	cmd := NewRulesCommand(log)

	agents := cmd.ListAgents()

	// Should have at least the built-in agents
	expected := []string{"codebuddy", "cursor", "kiro"}
	if len(agents) < len(expected) {
		t.Errorf("ListAgents() returned %d agents, want at least %d", len(agents), len(expected))
	}

	// Check all expected agents are present
	agentMap := make(map[string]bool)
	for _, agent := range agents {
		agentMap[agent] = true
	}

	for _, exp := range expected {
		if !agentMap[exp] {
			t.Errorf("ListAgents() missing expected agent %q", exp)
		}
	}

	// Check list is sorted
	for i := 1; i < len(agents); i++ {
		if agents[i-1] >= agents[i] {
			t.Errorf("ListAgents() not sorted: %q >= %q", agents[i-1], agents[i])
		}
	}
}

// TestRulesCommand_Execute_UnknownAgent tests error handling for unknown agent
func TestRulesCommand_Execute_UnknownAgent(t *testing.T) {
	log := genkit.NewLogger()
	cmd := NewRulesCommand(log)
	ctx := context.Background()

	err := cmd.Execute(ctx, "unknown-agent", false)
	if err == nil {
		t.Error("Execute() with unknown agent should return error")
	}

	if !strings.Contains(err.Error(), "unknown agent") {
		t.Errorf("Execute() error = %v, want error containing 'unknown agent'", err)
	}

	if !strings.Contains(err.Error(), "available agents") {
		t.Errorf("Execute() error = %v, want error containing 'available agents'", err)
	}
}

// TestRulesCommand_Execute_Preview tests preview mode (stdout)
func TestRulesCommand_Execute_Preview(t *testing.T) {
	// Create temporary directory for test to avoid plugin loading issues
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	log := genkit.NewLogger()
	cmd := NewRulesCommand(log)
	ctx := context.Background()

	// Preview mode should not create files
	err = cmd.Execute(ctx, "kiro", false)
	if err != nil {
		t.Fatalf("Execute() preview mode error = %v", err)
	}

	// No files should be created in preview mode
	// (This is a basic check - in a real test we'd capture stdout)
}

// TestRulesCommand_Execute_Write tests write mode
func TestRulesCommand_Execute_Write(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	log := genkit.NewLogger()
	cmd := NewRulesCommand(log)
	ctx := context.Background()

	// Test writing Kiro rules
	err = cmd.Execute(ctx, "kiro", true)
	if err != nil {
		t.Fatalf("Execute() write mode error = %v", err)
	}

	// Check that files were created
	outputDir := ".kiro/steering"
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("ReadDir(%s) error = %v", outputDir, err)
	}

	if len(entries) == 0 {
		t.Error("Execute() write mode created no files")
	}

	// Check that devgen rules are present
	expectedFiles := []string{
		"devgen.md",
		"devgen-plugin.md",
		"devgen-genkit.md",
		"devgen-rules.md",
	}

	fileMap := make(map[string]bool)
	for _, entry := range entries {
		fileMap[entry.Name()] = true
	}

	for _, expected := range expectedFiles {
		if !fileMap[expected] {
			t.Errorf("Execute() write mode missing expected file %q", expected)
		}
	}

	// Verify file content has correct frontmatter
	testFile := filepath.Join(outputDir, "devgen.md")
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", testFile, err)
	}

	contentStr := string(content)
	if !strings.HasPrefix(contentStr, "---\n") {
		t.Error("Generated file missing YAML frontmatter")
	}

	if !strings.Contains(contentStr, "inclusion:") {
		t.Error("Generated file missing 'inclusion' field")
	}

	if !strings.Contains(contentStr, "fileMatchPattern:") {
		t.Error("Generated file missing 'fileMatchPattern' field")
	}
}

// TestRulesCommand_Execute_Write_CodeBuddy tests CodeBuddy format
func TestRulesCommand_Execute_Write_CodeBuddy(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	log := genkit.NewLogger()
	cmd := NewRulesCommand(log)
	ctx := context.Background()

	// Test writing CodeBuddy rules
	err = cmd.Execute(ctx, "codebuddy", true)
	if err != nil {
		t.Fatalf("Execute() write mode error = %v", err)
	}

	// Check that files were created in correct directory
	outputDir := ".codebuddy/rules"
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("ReadDir(%s) error = %v", outputDir, err)
	}

	if len(entries) == 0 {
		t.Error("Execute() write mode created no files")
	}

	// Verify file content has CodeBuddy frontmatter format
	testFile := filepath.Join(outputDir, "devgen.mdc")
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", testFile, err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "description:") {
		t.Error("Generated file missing 'description' field")
	}

	if !strings.Contains(contentStr, "globs:") {
		t.Error("Generated file missing 'globs' field")
	}

	if !strings.Contains(contentStr, "alwaysApply:") {
		t.Error("Generated file missing 'alwaysApply' field")
	}
}

// TestRulesCommand_CollectRules tests rule collection from multiple tools
func TestRulesCommand_CollectRules(t *testing.T) {
	// Create temporary directory for test to avoid plugin loading issues
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	log := genkit.NewLogger()
	cmd := NewRulesCommand(log)
	ctx := context.Background()

	rules, err := cmd.collectRules(ctx)
	if err != nil {
		t.Fatalf("collectRules() error = %v", err)
	}

	if len(rules) == 0 {
		t.Error("collectRules() returned no rules")
	}

	// Check that devgen's own rules are included
	devgenRuleNames := []string{"devgen", "devgen-plugin", "devgen-genkit", "devgen-rules"}
	ruleMap := make(map[string]bool)
	for _, rule := range rules {
		ruleMap[rule.Name] = true
	}

	for _, name := range devgenRuleNames {
		if !ruleMap[name] {
			t.Errorf("collectRules() missing devgen rule %q", name)
		}
	}

	// Check that tool rules are included (enumgen, validategen)
	toolRuleNames := []string{"devgen-tool-enumgen", "devgen-tool-validategen"}
	for _, name := range toolRuleNames {
		if !ruleMap[name] {
			t.Errorf("collectRules() missing tool rule %q", name)
		}
	}

	// Verify rule structure
	for _, rule := range rules {
		if rule.Name == "" {
			t.Error("collectRules() returned rule with empty Name")
		}
		if rule.Description == "" {
			t.Error("collectRules() returned rule with empty Description")
		}
		if rule.Content == "" {
			t.Error("collectRules() returned rule with empty Content")
		}
	}
}

// TestRulesCommand_WriteRules_CreateDirectory tests directory creation
func TestRulesCommand_WriteRules_CreateDirectory(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	defer func() {
		if err := os.Chdir(originalWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	log := genkit.NewLogger()
	cmd := NewRulesCommand(log)

	// Create test rules
	testRules := []genkit.Rule{
		{
			Name:        "test-rule",
			Description: "Test rule",
			Globs:       []string{"**/*.go"},
			AlwaysApply: false,
			Content:     "# Test\n\nTest content.",
		},
	}

	// Get Kiro adapter
	adapter, ok := cmd.registry.Get("kiro")
	if !ok {
		t.Fatal("Failed to get kiro adapter")
	}

	// Directory should not exist yet
	outputDir := adapter.OutputDir()
	if _, err := os.Stat(outputDir); err == nil {
		t.Fatalf("Directory %s already exists", outputDir)
	}

	// Write rules should create directory
	err = cmd.writeRules(adapter, testRules)
	if err != nil {
		t.Fatalf("writeRules() error = %v", err)
	}

	// Directory should now exist
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Errorf("writeRules() did not create directory %s", outputDir)
	}

	// File should exist
	testFile := filepath.Join(outputDir, "test-rule.md")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Errorf("writeRules() did not create file %s", testFile)
	}
}
