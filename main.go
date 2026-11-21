package main

import (
	"os"

	"github.com/daiksud/gh-settings/internal/command"
)

// version is set with -ldflags at build time.
var version = "dev"

func main() {
	os.Exit(command.Run(os.Args[1:], version))
}
