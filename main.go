package main

import (
	"os"

	"github.com/daiksud/gh-settings/internal/app"
)

// version is set with -ldflags at release build time.
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, commandOperations{
		export: exportCurrent,
		validate: func() error {
			_, err := app.Validate()
			return err
		},
		plan:  planCurrent,
		apply: applyCurrent,
	}))
}
