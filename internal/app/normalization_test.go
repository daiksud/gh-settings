package app

import (
	"errors"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestExportWorkflowNormalizationIsScopedToAcquiredOptIn(t *testing.T) {
	for _, test := range []struct {
		name    string
		include bool
		failure bool
		raw     string
	}{
		{"requested complete observation", true, false, "FUTURE"},
		{"supported exact observation", true, false, "read"},
		{"unrequested workflow policy", false, false, "FUTURE"},
		{"partial state accompanying failure", true, true, "FUTURE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			observed := github.NormalizeWorkflowPermissions(settings.WorkflowPermissionsState{DefaultWorkflowPermissions: test.raw, CanApprovePullRequestReviews: false})
			failure := errors.New("complete acquisition denied")
			reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
			reads := 0
			reader.WorkflowPermissions = func(repository.Repository) (settings.WorkflowPermissionsState, error) {
				reads++
				if test.failure {
					return observed, failure
				}
				return observed, nil
			}
			var warnings []settings.ObservationNormalization
			saved := false
			err := Export(ExportCheckout{
				Repository:                 checkoutRepository,
				IncludeWorkflowPermissions: test.include,
				NormalizationWarning:       func(normalization settings.ObservationNormalization) { warnings = append(warnings, normalization) },
				PrepareSave: func() (func(settings.Declaration) error, error) {
					return func(declaration settings.Declaration) error {
						saved = true
						ui, err := github.UIFromExportDeclaration(declaration)
						if err != nil {
							t.Fatal(err)
						}
						if test.include && (ui.Settings == nil || ui.Settings.Actions == nil || ui.Settings.Actions.General == nil || ui.Settings.Actions.General.WorkflowPermissions == nil || *ui.Settings.Actions.General.WorkflowPermissions != "read_repository_contents_and_packages_permissions" || ui.Settings.Actions.General.AllowGitHubActionsToCreateAndApprovePullRequests == nil || *ui.Settings.Actions.General.AllowGitHubActionsToCreateAndApprovePullRequests) {
							t.Fatal("fallback or explicit false approval lost", ui)
						}
						return nil
					}, nil
				},
			}, reader)
			var wantErr error
			wantReads, wantWarnings := 0, 0
			if test.include {
				wantReads = 1
				if test.raw == "FUTURE" {
					wantWarnings = 1
				}
			}
			if test.failure {
				wantErr, wantWarnings = failure, 0
			}
			if !errors.Is(err, wantErr) || saved == test.failure || reads != wantReads || len(warnings) != wantWarnings {
				t.Fatalf("export acquisition boundary lost: saved=%v reads=%d warnings=%+v error=%v", saved, reads, warnings, err)
			}
			if wantWarnings == 1 && warnings[0] != observed.Normalization {
				t.Fatal("warning lost raw provenance", warnings)
			}
		})
	}
}
