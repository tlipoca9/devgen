package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	devgenrules "github.com/tlipoca9/devgen/cmd/devgen/rules"
	"github.com/tlipoca9/devgen/genkit"
)

// RulesCommand handles the 'devgen rules' subcommand.
// It collects rules from all tools and generates agent-specific rule files.
type RulesCommand struct {
	registry *genkit.AdapterRegistry
	log      *genkit.Logger
}

// NewRulesCommand creates a new RulesCommand with the adapter registry.
func NewRulesCommand(log *genkit.Logger) *RulesCommand {
	return &RulesCommand{
		registry: genkit.NewAdapterRegistry(),
		log:      log,
	}
}

// Execute runs the rules command with the specified agent and write mode.
// If write is false, rules are printed to stdout (preview mode).
// If write is true, rules are written to agent-specific directory.
func (c *RulesCommand) Execute(ctx context.Context, agent string, write bool) error {
	// Get adapter
	adapter, ok := c.registry.Get(agent)
	if !ok {
		available := c.registry.List()
		return fmt.Errorf("unknown agent %q, available agents: %s", agent, strings.Join(available, ", "))
	}

	// Collect rules from all tools
	rules, err := c.collectRules(ctx)
	if err != nil {
		return fmt.Errorf("collect rules: %w", err)
	}

	if len(rules) == 0 {
		c.log.Warn("No rules found (no tools implement RuleTool interface)")
		return nil
	}

	// Generate output
	if write {
		return c.writeRules(adapter, rules)
	}
	return c.preview(adapter, rules)
}

// ExecuteAll runs the rules command for all supported agents.
// Only supports write mode (preview mode for all agents would be too verbose).
func (c *RulesCommand) ExecuteAll(ctx context.Context, write bool) error {
	if !write {
		return fmt.Errorf("--agent all requires -w/--write flag")
	}

	// Collect rules once
	rules, err := c.collectRules(ctx)
	if err != nil {
		return fmt.Errorf("collect rules: %w", err)
	}

	if len(rules) == 0 {
		c.log.Warn("No rules found (no tools implement RuleTool interface)")
		return nil
	}

	// Write rules for all agents
	agents := c.registry.List()
	for _, agentName := range agents {
		adapter, ok := c.registry.Get(agentName)
		if !ok {
			continue
		}
		if err := c.writeRules(adapter, rules); err != nil {
			return fmt.Errorf("write rules for %s: %w", agentName, err)
		}
	}

	return nil
}

// ListAgents returns all available agent names.
func (c *RulesCommand) ListAgents() []string {
	return c.registry.List()
}

// collectRules gathers rules from all sources:
// 1. Project-level rules from .devgen/rules/ directory
// 2. Built-in devgen rules (if enabled)
// 3. Plugin rules (from tools implementing RuleTool)
func (c *RulesCommand) collectRules(ctx context.Context) ([]genkit.Rule, error) {
	var allRules []genkit.Rule

	// Load config
	configSearchDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get working directory: %w", err)
	}

	cfg, err := genkit.LoadConfig(configSearchDir)
	if err != nil {
		cfg = &genkit.Config{}
	}

	// 1. Load project-level rules from source directory (only if explicitly configured)
	if cfg.Rules.HasSourceDir() {
		sourceDir := cfg.Rules.GetSourceDir()
		projectRules, err := genkit.LoadRulesFromDir(sourceDir)
		if err != nil {
			return nil, fmt.Errorf("load project rules from %s: %w", sourceDir, err)
		}
		if len(projectRules) > 0 {
			c.log.Info("Loaded %v project rule(s) from %s", len(projectRules), sourceDir)
			allRules = append(allRules, projectRules...)
		}
	}

	// 2. Add devgen's own rules (if enabled)
	if cfg.Rules.ShouldIncludeBuiltin() {
		allRules = append(allRules, devgenRules()...)
	}

	// 3. Collect rules from plugins
	tools := make([]genkit.Tool, 0, len(builtinTools)+len(cfg.Plugins))
	toolNames := make(map[string]bool)

	// Load external plugins first
	if len(cfg.Plugins) > 0 {
		loader := genkit.NewPluginLoader("")
		pluginTools, err := loader.LoadPlugins(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("load plugins: %w", err)
		}
		for _, tool := range pluginTools {
			tools = append(tools, tool)
			toolNames[tool.Name()] = true
		}
	}

	// Add built-in tools
	for _, tool := range builtinTools {
		if !toolNames[tool.Name()] {
			tools = append(tools, tool)
		}
	}

	// Collect rules from tools that implement RuleTool
	for _, tool := range tools {
		if rt, ok := tool.(genkit.RuleTool); ok {
			rules := rt.Rules()
			allRules = append(allRules, rules...)
		}
	}

	return allRules, nil
}

// preview prints rules to stdout without writing files.
func (c *RulesCommand) preview(adapter genkit.AgentAdapter, rules []genkit.Rule) error {
	for i, rule := range rules {
		if i > 0 {
			fmt.Println("\n" + strings.Repeat("=", 80) + "\n")
		}

		filename, content, err := adapter.Transform(rule)
		if err != nil {
			return fmt.Errorf("transform rule %s: %w", rule.Name, err)
		}

		fmt.Printf("# File: %s/%s\n\n", adapter.OutputDir(), filename)
		fmt.Print(content)
	}
	return nil
}

// writeRules writes rules to agent-specific directory.
func (c *RulesCommand) writeRules(adapter genkit.AgentAdapter, rules []genkit.Rule) error {
	outputDir := adapter.OutputDir()

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", outputDir, err)
	}

	c.log.Info("Generating rules for %s...", adapter.Name())

	// Transform and write each rule
	for _, rule := range rules {
		filename, content, err := adapter.Transform(rule)
		if err != nil {
			return fmt.Errorf("transform rule %s: %w", rule.Name, err)
		}

		filepath := filepath.Join(outputDir, filename)
		if err := os.WriteFile(filepath, []byte(content), 0644); err != nil {
			return fmt.Errorf("write file %s: %w", filepath, err)
		}
	}

	c.log.Done("Generated %v rule file(s) in %s", len(rules), outputDir)
	for _, rule := range rules {
		filename, _, err := adapter.Transform(rule)
		if err != nil {
			continue
		}
		filepath := filepath.Join(outputDir, filename)
		c.log.Item("%s", filepath)
	}

	return nil
}

// devgenRules returns the rules for devgen itself.
func devgenRules() []genkit.Rule {
	return []genkit.Rule{
		{
			Name:        "devgen",
			Description: "devgen code generation toolkit usage guide. Includes installation, CLI usage, configuration, and troubleshooting.",
			Globs:       []string{"**/devgen.toml", "**/*.go"},
			AlwaysApply: false,
			Content:     devgenrules.DevgenRule,
		},
		{
			Name:        "devgen-plugin",
			Description: "devgen plugin development guide. How to use genkit framework to develop custom code generation plugins.",
			Globs:       []string{"**/*.go"},
			AlwaysApply: false,
			Content:     devgenrules.DevgenPluginRule,
		},
		{
			Name:        "devgen-genkit",
			Description: "genkit API reference. Includes Generator, GeneratedFile, Package, Type data structures and annotation parsing functions.",
			Globs:       []string{"**/*.go"},
			AlwaysApply: false,
			Content:     devgenrules.DevgenGenkitRule,
		},
		{
			Name:        "devgen-rules",
			Description: "devgen AI Rules system documentation. How to view, generate, and write AI rules.",
			Globs:       []string{"**/*.go"},
			AlwaysApply: true,
			Content:     devgenrules.DevgenRulesRule,
		},
	}
}
