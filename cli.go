package main

import (
	"errors"
	"fmt"
	"io"
)

type commandOperations struct {
	export   func() ([]byte, error)
	validate func() error
	plan     func() ([]byte, error)
}

func run(args []string, stdout, stderr io.Writer, operations commandOperations) int {
	if err := executeCommand(args, stdout, operations); err != nil {
		fmt.Fprintf(stderr, "gh settings: %v\n", err)
		return 1
	}
	return 0
}

func executeCommand(args []string, stdout io.Writer, operations commandOperations) error {
	if len(args) == 1 && (args[0] == "--version" || args[0] == "-v") {
		if _, err := fmt.Fprintln(stdout, "gh-settings "+version); err != nil {
			return fmt.Errorf("write version: %w", err)
		}
		return nil
	}
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		if _, err := fmt.Fprintln(stdout, "Usage: gh settings export\n       gh settings validate\n       gh settings plan\nExport the current repository's five basic settings as YAML to standard output.\nValidate .github/settings.yml locally.\nPlan differences for declared repository settings."); err != nil {
			return fmt.Errorf("write help: %w", err)
		}
		return nil
	}
	if len(args) != 1 || (args[0] != "export" && args[0] != "validate" && args[0] != "plan") {
		return errors.New("unsupported command or arguments; usage: gh settings <export|validate|plan>")
	}
	if args[0] == "validate" {
		return operations.validate()
	}
	operation := operations.export
	if args[0] == "plan" {
		operation = operations.plan
	}
	data, err := operation()
	if err != nil {
		return err
	}
	if _, err := stdout.Write(data); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
