// Command golangcilint integrates golangci-lint with devgen.
// It checks if golangci-lint is configured and installed, then runs it
// and converts the output to devgen diagnostics for IDE integration.
//
// This tool is validation-only and does not generate any code.
// It is automatically enabled when a golangci-lint config file exists
// (.golangci.yml, .golangci.yaml, .golangci.toml, or .golangci.json).
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

	"github.com/tlipoca9/devgen/cmd/golangcilint/generator"
	"github.com/tlipoca9/devgen/genkit"
)

func main() {
	if err := fang.Execute(context.Background(), rootCmd()); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "golangcilint [packages]",
		Short: "Run golangci-lint and output devgen diagnostics",
		Long: `golangcilint integrates golangci-lint with devgen.

It checks if golangci-lint is configured and installed, then runs it
and converts the output to devgen diagnostics for IDE integration.

This tool is validation-only and does not generate any code.`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				args = []string{"./..."}
			}
			return run(args)
		},
	}
	return cmd
}

func run(args []string) error {
	log := genkit.NewLogger()
	gen := genkit.New(genkit.Options{
		IgnoreGeneratedFiles: true,
	})

	if err := gen.Load(args...); err != nil {
		return fmt.Errorf("load: %w", err)
	}

	tool := generator.New()
	diagnostics := tool.Validate(gen, log)

	if len(diagnostics) == 0 {
		log.Done("No issues found")
		return nil
	}

	log.Warn("Found %d issue(s)", len(diagnostics))
	for _, d := range diagnostics {
		loc := ""
		if d.File != "" {
			loc = fmt.Sprintf("%s:%d:%d: ", d.File, d.Line, d.Column)
		}
		switch d.Severity {
		case genkit.DiagnosticError:
			log.Warn("[%s] %s%s", d.Code, loc, d.Message)
		case genkit.DiagnosticWarning:
			log.Warn("[%s] %s%s", d.Code, loc, d.Message)
		default:
			log.Item("[%s] %s%s", d.Code, loc, d.Message)
		}
	}

	return nil
}
