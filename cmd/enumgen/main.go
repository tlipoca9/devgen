// Command enumgen generates enum helper methods.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

	"github.com/tlipoca9/devgen/cmd/enumgen/generator"
	"github.com/tlipoca9/devgen/genkit"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

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
		Use:     "enumgen [packages]",
		Short:   "Generate enum helper methods",
		Long:    `enumgen generates enum helper methods for Go types annotated with enumgen:@enum.`,
		Version: fmt.Sprintf("%s (%s) %s", ver, commit, date),
		Example: `  enumgen ./...              # all packages
  enumgen ./pkg/status       # specific package`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return run(cmd, args)
		},
	}
	cmd.SetVersionTemplate(fmt.Sprintf("enumgen %s (%s) %s\n", ver, commit, date))

	return cmd
}

func run(_ *cobra.Command, args []string) error {
	log := genkit.NewLogger()

	gen := genkit.New(genkit.Options{
		IgnoreGeneratedFiles: true,
	})
	if err := gen.Load(args...); err != nil {
		return fmt.Errorf("load: %w", err)
	}

	log.Load("Loaded %v package(s)", len(gen.Packages))
	for _, pkg := range gen.Packages {
		log.Item("%v", pkg.GoImportPath())
	}

	tool := generator.New()
	if err := tool.Run(gen, log); err != nil {
		return err
	}

	files, err := gen.DryRun()
	if err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	if len(files) == 0 {
		log.Warn("No enums found")
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
