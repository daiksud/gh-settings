package main

import (
	"fmt"
	"io"
)

func run(args []string, stdout, stderr io.Writer, export func() ([]byte, error)) int {
	if len(args) == 1 && (args[0] == "--version" || args[0] == "-v") {
		if _, err := fmt.Fprintln(stdout, "gh-settings "+version); err != nil {
			fmt.Fprintf(stderr, "gh settings: write version: %v\n", err)
			return 1
		}
		return 0
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		if _, err := fmt.Fprintln(stdout, "Usage: gh settings export\n       gh settings validate\nExport the current repository's five basic settings as YAML to standard output.\nValidate .github/settings.yml locally."); err != nil {
			fmt.Fprintf(stderr, "gh settings: write help: %v\n", err)
			return 1
		}
		return 0
	}
	if len(args) != 1 || (args[0] != "export" && args[0] != "validate") {
		fmt.Fprintln(stderr, "gh settings: unsupported command or arguments; usage: gh settings <export|validate>")
		return 1
	}
	if args[0] == "validate" {
		if err := validateCurrent(); err != nil {
			fmt.Fprintf(stderr, "gh settings: %v\n", err)
			return 1
		}
		return 0
	}
	data, err := export()
	if err != nil {
		fmt.Fprintf(stderr, "gh settings: %v\n", err)
		return 1
	}
	if _, err := stdout.Write(data); err != nil {
		fmt.Fprintf(stderr, "gh settings: write output: %v\n", err)
		return 1
	}
	return 0
}

