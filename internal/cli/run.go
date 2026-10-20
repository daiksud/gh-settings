package cli

import (
	"errors"
	"fmt"
	"io"

	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/settings"
)

var errUnsupportedCommand = errors.New("unsupported command or arguments; usage: gh settings <export|validate|plan|apply>")

// ExportOptions selects independently acquired opt-in resources.
type ExportOptions struct{ Rulesets, Actions, WorkflowPermissions, SecretScanning, PrivateReporting bool }

// Operations binds shared application operations to their real dependencies.
type Operations struct {
	Export   func(ExportOptions) error
	Validate func() (settings.Declaration, error)
	Plan     func() (app.PlanResult, error)
	Apply    func() (app.ApplyResult, error)
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
	if len(args) > 0 && args[0] == "export" {
		var options ExportOptions
		for i := 1; i < len(args); i++ {
			flag := args[i]
			switch flag {
			case "--security":
				if i+1 >= len(args) {
					return errUnsupportedCommand
				}
				i++
				switch args[i] {
				case "secret-scanning":
					if options.SecretScanning {
						return errUnsupportedCommand
					}
					options.SecretScanning = true
				case "private-vulnerability-reporting":
					if options.PrivateReporting {
						return errUnsupportedCommand
					}
					options.PrivateReporting = true
				default:
					return errUnsupportedCommand
				}

			case "--rulesets":
				if options.Rulesets {
					return errUnsupportedCommand
				}
				options.Rulesets = true
			case "--actions":
				if options.Actions {
					return errUnsupportedCommand
				}
				options.Actions = true
			case "--workflow-permissions":
				if options.WorkflowPermissions {
					return errUnsupportedCommand
				}
				options.WorkflowPermissions = true
			default:
				return errUnsupportedCommand
			}
		}
		return operations.Export(options)
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
		if _, err := fmt.Fprintln(stdout, "Usage: gh settings export [--rulesets] [--actions] [--workflow-permissions] [--security <secret-scanning|private-vulnerability-reporting>]...\n       gh settings validate\n       gh settings plan\n       gh settings apply\nExport the current repository's basic settings, topics and labels as YAML to .github/settings.yml at the checkout root.\nReplace an existing regular settings file; success is silent.\nUse --rulesets to include supported repository branch/tag Rulesets; complete bypass visibility is required.\nUse --actions to include repository Actions execution policy; Administration read is required.\nUse --workflow-permissions to include default workflow token permissions; Administration read is required.\nUse --security secret-scanning to include both repository Secret scanning controls; complete security visibility is required.\nUse --security private-vulnerability-reporting for the public-repository reporting toggle; Metadata read is required.\nValidate .github/settings.yml locally.\nPlan differences for declared repository settings.\nApply declared repository settings directly."); err != nil {
			return fmt.Errorf("write help: %w", err)
		}
		return nil
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
