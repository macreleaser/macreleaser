package main

import (
	"os"

	"github.com/macreleaser/macreleaser/pkg/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
