package main

import (
	"os"

	"github.com/tlipoca9/devgen/cmd/convertgen/generator"
	"github.com/tlipoca9/devgen/genkit"
)

func main() {
	gen := genkit.New()
	if err := gen.Load("./..."); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	tool := generator.New()
	log := genkit.NewLogger()

	if err := tool.Run(gen, log); err != nil {
		log.Error("run failed: %s", err.Error())
		os.Exit(1)
	}

	if err := gen.Write(); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}
}
