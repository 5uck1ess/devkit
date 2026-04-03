package main

import (
	"os"

	"github.com/5uck1ess/devkit/cmd"
)

var version = "dev"

func init() {
	cmd.Version = version
}

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
