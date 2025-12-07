// Command devgen is a unified code generator that runs all devgen tools.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

	enumgen "github.com/tlipoca9/devgen/cmd/enumgen/generator"
	validategen "github.com/tlipoca9/devgen/cmd/validategen/generator"
	"github.com/tlipoca9/devgen/genkit"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// tools is the list of all available code generation tools.
var tools = []genkit.Tool{
	enumgen.New(),
	validategen.New(),
}

func main() {
	if err := fang.Execute(context.Background(), rootCmd()); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	ver := version
	if ver == commit {
		ver = "dev"
	}
	cmd := &cobra.Command{
		Use:   "devgen [packages]",
		Short: "Unified code generator for Go",
		Long: `devgen is a unified code generator that runs all devgen tools:
  - enumgen: Generate enum helper methods (String, JSON, SQL, etc.)
  - validategen: Generate Validate() methods for structs`,
		Version: fmt.Sprintf("%s (%s) %s", ver, commit, date),
		Example: `  devgen ./...              # all packages
  devgen ./pkg/model        # specific package
  devgen ./pkg/...          # all packages under pkg/`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return run(cmd, args)
		},
	}
	cmd.SetVersionTemplate(fmt.Sprintf("devgen %s (%s) %s\n", ver, commit, date))

	return cmd
}

func run(_ *cobra.Command, args []string) error {
	log := genkit.NewLogger()

	gen := genkit.New()
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
