package main

import (
	"os"

	"github.com/toneclone/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		// Execute already rendered the error (structured in --json mode).
		os.Exit(1)
	}
}
