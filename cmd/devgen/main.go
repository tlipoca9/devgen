// Command devgen is a unified code generator that runs all devgen tools.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

	devgenrules "github.com/tlipoca9/devgen/cmd/devgen/rules"
	enumgen "github.com/tlipoca9/devgen/cmd/enumgen/generator"
	golangcilint "github.com/tlipoca9/devgen/cmd/golangcilint/generator"
	validategen "github.com/tlipoca9/devgen/cmd/validategen/generator"
	"github.com/tlipoca9/devgen/genkit"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// builtinTools is the list of built-in code generation tools.
var builtinTools = []genkit.Tool{
	enumgen.New(),
	validategen.New(),
	golangcilint.New(),
}

func main() {
	if err := fang.Execute(context.Background(), rootCmd()); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	var dryRun bool
	var jsonOutput bool
	var includeTests bool

	ver := version
	if ver == commit {
		ver = "dev"
	}
	cmd := &cobra.Command{
		Use:   "devgen [packages]",
		Short: "Unified code generator for Go",
		Long: `devgen is a unified code generator that runs all devgen tools:
  - enumgen: Generate enum helper methods (String, JSON, SQL, etc.)
  - validategen: Generate Validate() methods for structs

External plugins can be configured in devgen.toml:
  [[plugins]]
  name = "customgen"
  path = "./tools/customgen"
  type = "source"  # source | plugin`,
		Version: fmt.Sprintf("%s (%s) %s", ver, commit, date),
		Example: `  devgen ./...              # all packages
  devgen ./pkg/model        # specific package
  devgen ./pkg/...          # all packages under pkg/
  devgen --dry-run ./...    # validate without writing files
  devgen --dry-run --json ./...  # JSON output for IDE integration`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			if dryRun {
				return runDryRun(cmd.Context(), args, jsonOutput, includeTests)
			}
			return run(cmd.Context(), args, includeTests)
		},
	}
	cmd.SetVersionTemplate(fmt.Sprintf("devgen %s (%s) %s\n", ver, commit, date))

	// Add flags
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate and preview without writing files")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format (for IDE integration, requires --dry-run)")
	cmd.Flags().BoolVar(&includeTests, "include-tests", false, "Also generate *_test.go files")

	// Add config subcommand
	cmd.AddCommand(configCmd())

	// Add rules subcommand
	cmd.AddCommand(rulesCmd())

	return cmd
}

func configCmd() *cobra.Command {
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Output tools configuration",
		Long: `Output the configuration of all tools (built-in and plugins).

This command displays all available annotations and their metadata for each tool.
The VSCode extension uses this to provide autocomplete and validation.

WHAT THIS COMMAND SHOWS:
  • All available tools (enumgen, validategen, golangcilint, and plugins)
  • Annotations supported by each tool
  • Parameter types and allowed values for each annotation
  • Documentation for each annotation

OUTPUT FORMAT:
  TOML (default): Human-readable format for understanding the configuration
  JSON (--json):  Machine-readable format for IDE/tool integration

CONFIGURATION FILES:
  devgen looks for devgen.toml in the current directory (and parent directories)
  to load plugin configurations. Example devgen.toml:

    [[plugins]]
    name = "customgen"
    path = "./tools/customgen"
    type = "source"

UNDERSTANDING THE OUTPUT:
  Each tool section shows:
    • output_suffix: Generated file suffix (e.g., "_enum.go")
    • annotations: List of supported annotations with:
        - name: Annotation name (e.g., "enum", "required")
        - type: Where to use ("type" for types, "field" for struct fields)
        - doc: Description of what the annotation does
        - params: Parameter configuration (if the annotation takes arguments)
            - type: Expected type ("string", "number", "bool", "list", "enum")
            - values: Allowed values for enum type
            - placeholder: Hint text for the parameter`,
		Example: `  # View all tool configurations in human-readable TOML format
  devgen config

  # View configurations in JSON format (for IDE integration)
  devgen config --json

  # Pipe to less for easier reading
  devgen config | less

  # Search for specific annotation
  devgen config | grep -A5 "required"

  # Pretty print JSON output
  devgen config --json | jq .

  # List all available annotations for validategen
  devgen config --json | jq '.validategen.annotations | keys'

  # Get details about a specific annotation
  devgen config --json | jq '.validategen.annotations.email'

EXAMPLE OUTPUT (TOML):
  [tools.enumgen]
  output_suffix = "_enum.go"

  [[tools.enumgen.annotations]]
  name = "enum"
  type = "type"
  doc = "Generate enum helper methods (options: string, json, text, sql)"

  [tools.enumgen.annotations.params]
  values = ["string", "json", "text", "sql"]

EXAMPLE OUTPUT (JSON):
  {
    "enumgen": {
      "outputSuffix": "_enum.go",
      "typeAnnotations": ["enum"],
      "fieldAnnotations": ["name"],
      "annotations": {
        "enum": {
          "doc": "Generate enum helper methods",
          "paramType": "enum",
          "values": ["string", "json", "text", "sql"]
        }
      }
    }
  }

HOW TO USE ANNOTATIONS:
  Type annotations (applied to type declarations):
    // enumgen:@enum(string, json)
    type Status int

  Field annotations (applied to struct fields):
    type User struct {
        // validategen:@required
        // validategen:@email
        Email string
    }`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfig(cmd.Context(), jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format (for IDE/tool integration)")

	return cmd
}

func runConfig(ctx context.Context, jsonOutput bool) error {
	// Load config to get plugins
	configSearchDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	cfg, err := genkit.LoadConfig(configSearchDir)
	if err != nil {
		cfg = &genkit.Config{}
	}

	// Collect all tools
	tools := make([]genkit.Tool, 0, len(builtinTools)+len(cfg.Plugins))
	toolNames := make(map[string]bool)

	// Load external plugins first
	if len(cfg.Plugins) > 0 {
		loader := genkit.NewPluginLoader("")
		pluginTools, err := loader.LoadPlugins(ctx, cfg)
		if err != nil {
			return fmt.Errorf("load plugins: %w", err)
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

	// Collect configs from tools
	toolConfigs := genkit.CollectToolConfigs(tools)

	// Merge with config file (config file takes precedence)
	if cfg.Tools != nil {
		toolConfigs = genkit.MergeToolConfigs(toolConfigs, cfg.Tools)
	}

	// Output
	if jsonOutput {
		return outputConfigJSON(toolConfigs)
	}
	return outputConfigTOML(toolConfigs)
}

func outputConfigJSON(configs map[string]genkit.ToolConfig) error {
	// Convert to VSCode extension format
	result := make(map[string]any)

	for name, cfg := range configs {
		result[name] = cfg.ToVSCodeConfig()
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

func outputConfigTOML(configs map[string]genkit.ToolConfig) error {
	for name, cfg := range configs {
		fmt.Printf("[tools.%s]\n", name)
		if cfg.OutputSuffix != "" {
			fmt.Printf("output_suffix = %q\n", cfg.OutputSuffix)
		}
		fmt.Println()

		for _, ann := range cfg.Annotations {
			fmt.Printf("[[tools.%s.annotations]]\n", name)
			fmt.Printf("name = %q\n", ann.Name)
			fmt.Printf("type = %q\n", ann.Type)
			if ann.Doc != "" {
				fmt.Printf("doc = %q\n", ann.Doc)
			}
			if ann.Params != nil {
				fmt.Println()
				fmt.Printf("[tools.%s.annotations.params]\n", name)
				if ann.Params.Type != nil {
					fmt.Printf("type = %q\n", ann.Params.Type)
				}
				if len(ann.Params.Values) > 0 {
					fmt.Printf("values = %v\n", formatStringSlice(ann.Params.Values))
				}
				if ann.Params.Placeholder != "" {
					fmt.Printf("placeholder = %q\n", ann.Params.Placeholder)
				}
				if ann.Params.MaxArgs > 0 {
					fmt.Printf("maxArgs = %d\n", ann.Params.MaxArgs)
				}
			}
			fmt.Println()
		}
	}
	return nil
}

func formatStringSlice(ss []string) string {
	quoted := make([]string, len(ss))
	for i, s := range ss {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func runDryRun(ctx context.Context, args []string, jsonOutput bool, includeTests bool) error {
	// Use silent logger for JSON output to avoid polluting stdout
	var log *genkit.Logger
	if jsonOutput {
		log = genkit.NewLoggerWithWriter(io.Discard)
	} else {
		log = genkit.NewLogger()
	}
	result := &genkit.DryRunResult{
		Success: true,
		Files:   make(map[string]string),
	}

	// Determine config search directory from first argument
	configSearchDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// If first arg is a relative path, use it as the starting point for config search
	if len(args) > 0 {
		arg := args[0]
		arg = strings.TrimSuffix(arg, "/...")
		arg = strings.TrimSuffix(arg, "...")
		if arg == "." || arg == "" {
			// Use current directory
		} else if strings.HasPrefix(arg, "./") || strings.HasPrefix(arg, "../") || !strings.HasPrefix(arg, "/") {
			absPath, err := filepath.Abs(arg)
			if err == nil {
				if info, err := os.Stat(absPath); err == nil && info.IsDir() {
					configSearchDir = absPath
				}
			}
		}
	}

	cfg, err := genkit.LoadConfig(configSearchDir)
	if err != nil {
		cfg = &genkit.Config{}
	}

	// Collect all tools: built-in + plugins
	tools := make([]genkit.Tool, 0, len(builtinTools)+len(cfg.Plugins))
	toolNames := make(map[string]bool)

	// Load external plugins first
	if len(cfg.Plugins) > 0 {
		loader := genkit.NewPluginLoader("")
		pluginTools, err := loader.LoadPlugins(ctx, cfg)
		if err != nil {
			return fmt.Errorf("load plugins: %w", err)
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
			toolNames[tool.Name()] = true
		}
	}

	gen := genkit.New(genkit.Options{
		IgnoreGeneratedFiles: true,
		IncludeTests:         includeTests,
	})
	if err := gen.Load(args...); err != nil {
		return fmt.Errorf("load: %w", err)
	}
	result.Stats.PackagesLoaded = len(gen.Packages)

	// Run validation for tools that support it
	for _, tool := range tools {
		if vt, ok := tool.(genkit.ValidatableTool); ok {
			diagnostics := vt.Validate(gen, log)
			for _, d := range diagnostics {
				result.AddDiagnostic(d)
			}
		}
	}

	// If no validation errors, try to generate (dry-run)
	if result.Success {
		for _, tool := range tools {
			if err := tool.Run(gen, log); err != nil {
				// Convert run error to diagnostic if possible
				result.Success = false
				result.AddDiagnostic(genkit.Diagnostic{
					Severity: genkit.DiagnosticError,
					Message:  err.Error(),
					Tool:     tool.Name(),
				})
			}
		}
	}

	// Get generated files preview
	if result.Success {
		files, err := gen.DryRun()
		if err != nil {
			result.Success = false
			result.AddDiagnostic(genkit.Diagnostic{
				Severity: genkit.DiagnosticError,
				Message:  fmt.Sprintf("generate: %v", err),
				Tool:     "devgen",
			})
		} else {
			result.Stats.FilesGenerated = len(files)
			for path, content := range files {
				// Store first 500 bytes as preview
				preview := string(content)
				if len(preview) > 500 {
					preview = preview[:500] + "\n... (truncated)"
				}
				result.Files[path] = preview
			}
		}
	}

	// Output result
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	return printDryRunResult(result, log)
}

func printDryRunResult(result *genkit.DryRunResult, log *genkit.Logger) error {
	if result.Success {
		log.Done("Dry-run successful")
		log.Item("Packages: %v", result.Stats.PackagesLoaded)
		log.Item("Files to generate: %v", result.Stats.FilesGenerated)
		for path := range result.Files {
			log.Item("  %s", path)
		}
	} else {
		log.Warn("Dry-run found issues")
	}

	if result.Stats.ErrorCount > 0 {
		log.Warn("Errors: %v", result.Stats.ErrorCount)
	}
	if result.Stats.WarningCount > 0 {
		log.Warn("Warnings: %v", result.Stats.WarningCount)
	}

	for _, d := range result.Diagnostics {
		loc := ""
		if d.File != "" {
			loc = fmt.Sprintf("%s:%d:%d: ", d.File, d.Line, d.Column)
		}
		switch d.Severity {
		case genkit.DiagnosticError:
			log.Warn("%s[%s] %s%s", d.Tool, d.Code, loc, d.Message)
		case genkit.DiagnosticWarning:
			log.Warn("%s[%s] %s%s", d.Tool, d.Code, loc, d.Message)
		default:
			log.Item("%s[%s] %s%s", d.Tool, d.Code, loc, d.Message)
		}
	}

	if !result.Success {
		return fmt.Errorf("dry-run failed with %d error(s)", result.Stats.ErrorCount)
	}
	return nil
}

func run(ctx context.Context, args []string, includeTests bool) error {
	log := genkit.NewLogger()

	// Determine config search directory from first argument
	configSearchDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// If first arg is a relative path, use it as the starting point for config search
	if len(args) > 0 {
		arg := args[0]
		// Handle patterns like "./...", "./pkg/...", "./pkg"
		arg = strings.TrimSuffix(arg, "/...")
		arg = strings.TrimSuffix(arg, "...")
		if arg == "." || arg == "" {
			// Use current directory
		} else if strings.HasPrefix(arg, "./") || strings.HasPrefix(arg, "../") || !strings.HasPrefix(arg, "/") {
			// Relative path - resolve it
			absPath, err := filepath.Abs(arg)
			if err == nil {
				if info, err := os.Stat(absPath); err == nil && info.IsDir() {
					configSearchDir = absPath
				}
			}
		}
	}

	cfg, err := genkit.LoadConfig(configSearchDir)
	if err != nil {
		log.Warn("Failed to load devgen.toml: %v", err)
		// Continue with built-in tools only
		cfg = &genkit.Config{}
	}

	// Collect all tools: built-in + plugins
	tools := make([]genkit.Tool, 0, len(builtinTools)+len(cfg.Plugins))
	toolNames := make(map[string]bool)

	// Load external plugins first (they can override built-in tools)
	if len(cfg.Plugins) > 0 {
		loader := genkit.NewPluginLoader("")
		pluginTools, err := loader.LoadPlugins(ctx, cfg)
		if err != nil {
			return fmt.Errorf("load plugins: %w", err)
		}
		for _, tool := range pluginTools {
			tools = append(tools, tool)
			toolNames[tool.Name()] = true
			log.Item("Loaded plugin: %s", tool.Name())
		}
	}

	// Add built-in tools (skip if overridden by plugin)
	for _, tool := range builtinTools {
		if !toolNames[tool.Name()] {
			tools = append(tools, tool)
			toolNames[tool.Name()] = true
		}
	}

	gen := genkit.New(genkit.Options{
		IgnoreGeneratedFiles: true,
		IncludeTests:         includeTests,
	})
	if err := gen.Load(args...); err != nil {
		return fmt.Errorf("load: %w", err)
	}

	log.Load("Loaded %v package(s)", len(gen.Packages))
	for _, pkg := range gen.Packages {
		log.Item("%v", pkg.GoImportPath())
	}

	// Run all tools
	for _, tool := range tools {
		if err := tool.Run(gen, log); err != nil {
			return fmt.Errorf("%s: %w", tool.Name(), err)
		}
	}

	files, err := gen.DryRun()
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	if len(files) == 0 {
		log.Warn("No annotations found")
		return nil
	}

	if err := gen.Write(); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	log.Done("Generated %v file(s)", len(files))
	for path := range files {
		log.Item("%v", path)
	}

	return nil
}

// AgentConfig defines the configuration for an AI agent's rules format.
type AgentConfig struct {
	// Name is the agent name (e.g., "codebuddy", "cursor").
	Name string
	// RulesDir is the directory where rules are stored.
	RulesDir string
	// FileExt is the file extension for rules files.
	FileExt string
	// FrontmatterFormat is the format for rule frontmatter.
	// Supported: "yaml" (CodeBuddy, Cursor)
	FrontmatterFormat string
}

// supportedAgents defines all supported AI agents.
var supportedAgents = map[string]AgentConfig{
	"codebuddy": {
		Name:              "codebuddy",
		RulesDir:          ".codebuddy/rules",
		FileExt:           ".md",
		FrontmatterFormat: "yaml",
	},
	// Future agents can be added here:
	// "cursor": {
	// 	Name:              "cursor",
	// 	RulesDir:          ".cursor/rules",
	// 	FileExt:           ".mdc",
	// 	FrontmatterFormat: "yaml",
	// },
}

func rulesCmd() *cobra.Command {
	var agentName string
	var writeFiles bool
	var listAgents bool
	var noColor bool

	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Generate AI rules for coding assistants",
		Long: `Generate AI-friendly rules/documentation for AI coding assistants.

This command generates rules that help AI assistants (like CodeBuddy, Cursor, etc.)
understand how to use devgen tools correctly. Each tool provides detailed,
step-by-step documentation with examples.

SUPPORTED AGENTS:
  codebuddy    Tencent CodeBuddy (.codebuddy/rules/*.md)

WHAT GETS GENERATED:
  Each tool that implements the RuleTool interface will generate a rule file
  containing:
    • Tool overview and when to use it
    • Step-by-step usage guide
    • Annotation reference with examples
    • Common mistakes and how to avoid them
    • Complete working examples

OUTPUT:
  By default, rules are printed to stdout.
  Use -w/--write to write files to the appropriate directory.`,
		Example: `  # List supported agents
  devgen rules --list-agents

  # Preview rules for CodeBuddy (stdout)
  devgen rules --agent codebuddy

  # Generate and write rules for CodeBuddy
  devgen rules --agent codebuddy -w`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if listAgents {
				return listSupportedAgents()
			}
			if agentName == "" {
				return fmt.Errorf("--agent is required. Use --list-agents to see supported agents")
			}
			return runRules(cmd.Context(), agentName, writeFiles, noColor)
		},
	}

	cmd.Flags().StringVar(&agentName, "agent", "", "Target AI agent (e.g., codebuddy)")
	cmd.Flags().BoolVarP(&writeFiles, "write", "w", false, "Write rules to files instead of stdout")
	cmd.Flags().BoolVar(&listAgents, "list-agents", false, "List supported AI agents")
	cmd.Flags().BoolVar(&noColor, "no-color", false, "Disable colored output")

	return cmd
}

func listSupportedAgents() error {
	fmt.Println("Supported AI agents:")
	fmt.Println()
	for name, cfg := range supportedAgents {
		fmt.Printf("  %-12s  %s/*%s\n", name, cfg.RulesDir, cfg.FileExt)
	}
	fmt.Println()
	fmt.Println("Usage: devgen rules --agent <name> [-w]")
	return nil
}

func runRules(ctx context.Context, agentName string, writeFiles bool, noColor bool) error {
	log := genkit.NewLogger().SetNoColor(noColor)

	// Validate agent
	agent, ok := supportedAgents[strings.ToLower(agentName)]
	if !ok {
		return fmt.Errorf("unsupported agent %q. Use --list to see supported agents", agentName)
	}

	// Load config to get plugins
	configSearchDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	cfg, err := genkit.LoadConfig(configSearchDir)
	if err != nil {
		cfg = &genkit.Config{}
	}

	// Collect all tools
	tools := make([]genkit.Tool, 0, len(builtinTools)+len(cfg.Plugins))
	toolNames := make(map[string]bool)

	// Load external plugins first
	if len(cfg.Plugins) > 0 {
		loader := genkit.NewPluginLoader("")
		pluginTools, err := loader.LoadPlugins(ctx, cfg)
		if err != nil {
			return fmt.Errorf("load plugins: %w", err)
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

	// Collect rules from tools
	var allRules []genkit.Rule

	// Add devgen's own rules first
	allRules = append(allRules, devgenRules()...)

	// Add rules from tools
	for _, tool := range tools {
		if rt, ok := tool.(genkit.RuleTool); ok {
			rules := rt.Rules()
			allRules = append(allRules, rules...)
		}
	}

	if len(allRules) == 0 {
		log.Warn("No rules found (no tools implement RuleTool interface)")
		return nil
	}

	// Generate output
	if writeFiles {
		return writeRuleFiles(agent, allRules, log)
	}
	return printRules(agent, allRules)
}

func printRules(agent AgentConfig, rules []genkit.Rule) error {
	for i, rule := range rules {
		if i > 0 {
			fmt.Println("\n" + strings.Repeat("=", 80) + "\n")
		}
		fmt.Printf("# File: %s/%s%s\n\n", agent.RulesDir, rule.Name, agent.FileExt)
		fmt.Print(formatRuleContent(agent, rule))
	}
	return nil
}

func writeRuleFiles(agent AgentConfig, rules []genkit.Rule, log *genkit.Logger) error {
	// Create rules directory
	if err := os.MkdirAll(agent.RulesDir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", agent.RulesDir, err)
	}

	log.Info("Generating rules for %s...", agent.Name)

	for _, rule := range rules {
		filename := filepath.Join(agent.RulesDir, rule.Name+agent.FileExt)
		content := formatRuleContent(agent, rule)

		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", filename, err)
		}
	}

	log.Done("Generated %v rule file(s) in %s", len(rules), agent.RulesDir)
	for _, rule := range rules {
		filename := filepath.Join(agent.RulesDir, rule.Name+agent.FileExt)
		log.Item("%s", filename)
	}
	return nil
}

func formatRuleContent(agent AgentConfig, rule genkit.Rule) string {
	var sb strings.Builder

	// Write frontmatter
	switch agent.FrontmatterFormat {
	case "yaml":
		sb.WriteString("---\n")
		sb.WriteString(fmt.Sprintf("description: %s\n", rule.Description))
		if len(rule.Globs) > 0 {
			sb.WriteString(fmt.Sprintf("globs: %s\n", strings.Join(rule.Globs, ", ")))
		}
		sb.WriteString(fmt.Sprintf("alwaysApply: %t\n", rule.AlwaysApply))
		sb.WriteString("---\n\n")
	}

	// Write content
	sb.WriteString(rule.Content)

	return sb.String()
}

// devgenRules returns the rules for devgen itself.
func devgenRules() []genkit.Rule {
	return []genkit.Rule{
		{
			Name:        "devgen",
			Description: "devgen 代码生成工具集使用指南。包含安装、命令行用法、配置文件、故障排查等。",
			Globs:       []string{"**/devgen.toml", "**/*.go"},
			AlwaysApply: false,
			Content:     devgenrules.DevgenRule,
		},
		{
			Name:        "devgen-plugin",
			Description: "devgen 插件开发指南。介绍如何使用 genkit 框架开发自定义代码生成插件。",
			Globs:       []string{"**/*.go"},
			AlwaysApply: false,
			Content:     devgenrules.DevgenPluginRule,
		},
		{
			Name:        "devgen-genkit",
			Description: "genkit API 参考。包含 Generator、GeneratedFile、Package、Type 等数据结构和注解解析函数。",
			Globs:       []string{"**/*.go"},
			AlwaysApply: false,
			Content:     devgenrules.DevgenGenkitRule,
		},
		{
			Name:        "devgen-rules",
			Description: "devgen AI Rules 系统说明。介绍如何查看、生成和编写 AI rules。",
			Globs:       []string{"**/*.go"},
			AlwaysApply: true,
			Content:     devgenrules.DevgenRulesRule,
		},
	}
}
