// Package command wires the CLI and Action entry points to the current checkout.
package command

import (
	"os"

	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/cli"
	"github.com/daiksud/gh-settings/internal/command/bindings"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

// Run dispatches the command-line arguments and returns the process exit code.
func Run(args []string, version string) int {
	checkout := workspace.NewCheckout(".")
	loadDeclaration := func() (settings.Declaration, error) { return loadCheckoutDeclaration(checkout) }
	reader := bindings.Reader()
	if len(args) > 0 && args[0] == "action" {
		return actions.Run(args[1:], os.Stdout, os.Stderr, version, func() (app.PlanResult, error) {
			var prune settings.PruneScopes
			if raw := os.Getenv("SETTINGS_PRUNE"); raw != "" {
				var err error
				prune, err = settings.ParsePruneScopes(raw)
				if err != nil {
					return app.PlanResult{}, &app.PlanError{Stage: "invalid-config", Err: err}
				}
			}
			return app.Plan(loadDeclaration, checkout.Repository, reader, prune)
		})
	}
	return cli.Run(args, os.Stdout, os.Stderr, version, cli.Operations{
		Export:   func() error { return exportCheckout(checkout, reader, os.Stderr) },
		Validate: func() (settings.Declaration, error) { return app.Validate(loadDeclaration) },
		Plan: func(options cli.PlanOptions) (app.PlanResult, error) {
			return app.Plan(loadDeclaration, checkout.Repository, reader, options.Prune)
		},
		Apply: func(options cli.ApplyOptions) (app.ApplyResult, error) {
			desired, record, err := prepareCheckoutApply(checkout)
			if err != nil {
				return app.ApplyResult{Outcome: app.ApplyFailed}, err
			}
			if options.NoWriteback || os.Getenv("GITHUB_ACTIONS") == "true" {
				record = nil
			}
			return app.Apply(func() (settings.Declaration, error) { return desired, nil }, checkout.Repository, bindings.Writer(reader, desired, record), options.Prune)
		},
	})
}
