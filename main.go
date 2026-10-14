package main

import (
	"os"
)

// version is set with -ldflags at release build time.
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, commandOperations{
		export:   exportCurrent,
		validate: validateCurrent,
		plan:     planCurrent,
	}))
}
