package main

import (
	"os"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/actions"
	"github.com/daiksud/gh-settings/internal/app"
	"github.com/daiksud/gh-settings/internal/cli"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/workspace"
)

// version is set with -ldflags at build time.
var version = "dev"

func main() {
	checkout := workspace.NewCheckout(".")
	loadDeclaration := func() (settings.Declaration, error) { return loadCheckoutDeclaration(checkout) }
	reader := githubReader()
	if len(os.Args) > 1 && os.Args[1] == "action" {
		os.Exit(actions.Run(os.Args[2:], os.Stdout, os.Stderr, version, func() (app.PlanResult, error) {
			var prune settings.PruneScopes
			if raw := os.Getenv("SETTINGS_PRUNE"); raw != "" {
				var err error
				prune, err = settings.ParsePruneScopes(raw)
				if err != nil {
					return app.PlanResult{}, &app.PlanError{Stage: "invalid-config", Err: err}
				}
			}
			return app.Plan(loadDeclaration, checkout.Repository, reader, prune)
		}))
	}
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, version, cli.Operations{
		Export: func() error {
			warnings := exportWarnings{output: os.Stderr}
			target, err := checkout.Repository()
			if err != nil {
				return err
			}
			existing, save, err := prepareCheckoutSave(checkout)
			if err != nil {
				return err
			}
			return app.Export(app.ExportCheckout{
				Check:                              github.CheckExportAcquisition,
				Existing:                           existing,
				Info:                               exportInfo,
				Repository:                         func() (repository.Repository, error) { return target, nil },
				PrepareSave:                        func() (func(settings.Declaration) error, error) { return save, nil },
				NormalizationWarning:               warnings.warnNormalization,
				CommitMessageNormalizationWarning:  warnings.warnCommitMessageNormalization,
				SecurityStatusNormalizationWarning: warnings.warnSecurityStatusNormalization,
				RulesetNormalizationWarning:        warnings.warnRulesetNormalization,
				RulesetRequiredApprovalsNormalizationWarning:      warnings.warnRulesetRequiredApprovalsNormalization,
				RulesetMergeMethodsNormalizationWarning:           warnings.warnRulesetMergeMethodsNormalization,
				RulesetTeamApprovalsNormalizationWarning:          warnings.warnRulesetTeamApprovalsNormalization,
				RulesetScanningAlertsNormalizationWarning:         warnings.warnRulesetScanningAlertsNormalization,
				RulesetScanningSecurityAlertsNormalizationWarning: warnings.warnRulesetScanningSecurityAlertsNormalization,
				RulesetBypassModeNormalizationWarning:             warnings.warnRulesetBypassModeNormalization,
				EnvironmentDeploymentNormalizationWarning:         warnings.warnEnvironmentDeploymentNormalization,
				EnvironmentRefTypeNormalizationWarning:            warnings.warnEnvironmentRefTypeNormalization,
			}, reader)
		},
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
			return app.Apply(func() (settings.Declaration, error) { return desired, nil }, checkout.Repository, githubWriter(reader, desired, record), options.Prune)
		},
	}))
}
