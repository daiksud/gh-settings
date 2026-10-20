package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

var errUnsupportedCommand = errors.New("unsupported command or arguments; usage: gh settings <export|validate|plan|apply>")

// Operations binds the four shared application operations to their real dependencies.
type Operations struct {
	Export         func() error
	ExportRulesets func() error
	Validate       func() (settings.Declaration, error)
	Plan           func() (app.PlanResult, error)
	Apply          func() (app.ApplyResult, error)
}

// Run adapts arguments, typed results and errors to terminal output and exit status.
func Run(args []string, stdout, stderr io.Writer, version string, operations Operations) int {
	if err := executeCommand(args, stdout, version, operations); err != nil {
		fmt.Fprintf(stderr, "gh settings: %v\n", err)
		return 1
	}
	return 0
}

func executeCommand(args []string, stdout io.Writer, version string, operations Operations) error {
	if len(args) == 2 && args[0] == "export" && args[1] == "--rulesets" && operations.ExportRulesets != nil {
		return operations.ExportRulesets()
	}
	if len(args) != 1 {
		return errUnsupportedCommand
	}
	var data []byte
	switch args[0] {
	case "--version", "-v":
		if _, err := fmt.Fprintln(stdout, "gh-settings "+version); err != nil {
			return fmt.Errorf("write version: %w", err)
		}
		return nil
	case "--help", "-h":
		if _, err := fmt.Fprintln(stdout, "Usage: gh settings export [--rulesets]\n       gh settings validate\n       gh settings plan\n       gh settings apply\nExport the current repository's basic settings, topics and labels as YAML to .github/settings.yml at the checkout root.\nReplace an existing regular settings file; success is silent.\nUse --rulesets to include supported repository branch/tag Rulesets; complete bypass visibility is required.\nValidate .github/settings.yml locally.\nPlan differences for declared repository settings.\nApply declared repository settings directly."); err != nil {
			return fmt.Errorf("write help: %w", err)
		}
		return nil
	case "export":
		return operations.Export()
	case "validate":
		_, err := operations.Validate()
		return err
	case "plan":
		result, err := operations.Plan()
		if err != nil {
			return err
		}
		data = RenderComparison(result.Comparison)
	case "apply":
		result, err := operations.Apply()
		if err != nil {
			return err
		}
		if result.Completed() {
			data = []byte("Applied repository settings.\n")
		} else {
			data = []byte("No managed repository settings.\n")
		}
	default:
		return errUnsupportedCommand
	}
	if _, err := stdout.Write(data); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}
