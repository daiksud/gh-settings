package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func exportEnvironmentModeState(t *testing.T, id int64, name string, raw bool) settings.EnvironmentState {
	t.Helper()
	zero, off, reviewers := 0, false, []settings.EnvironmentReviewer{}
	state := settings.EnvironmentState{ID: id, Policy: settings.EnvironmentDeclaration{Name: name, WaitTimer: &zero, PreventSelfReview: &off, CanAdminsBypass: &off, Reviewers: &reviewers, DeploymentBranchPolicyDeclared: true}, DeploymentModeNormalization: settings.EnvironmentDeploymentNormalization{Path: settings.EnvironmentDeploymentModePath, RawPolicy: settings.EnvironmentBranchPolicy{ProtectedBranches: raw, CustomBranchPolicies: raw}, Fallback: uimodel.NoRestriction}}
	if raw {
		state.Patterns = []settings.EnvironmentPatternState{{ID: 31, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "LATENT_PATTERN"}}}
	}
	return state
}

func TestEnvironmentDeploymentExportWarnsAfterCompleteAcquisitionInNameOrder(t *testing.T) {
	type warning struct {
		name   string
		record settings.EnvironmentDeploymentNormalization
	}
	for _, mode := range []string{"selected", "canonical", "unrequested", "nil sink", "read", "unacquired", "ui"} {
		z, a := exportEnvironmentModeState(t, 7, "Z", true), exportEnvironmentModeState(t, 9, "a", false)
		if mode == "canonical" {
			z.DeploymentModeNormalization, a.DeploymentModeNormalization = settings.EnvironmentDeploymentNormalization{}, settings.EnvironmentDeploymentNormalization{}
			z.Patterns = nil
		}
		canonical := exportEnvironmentModeState(t, 8, "canonical", false)
		canonical.DeploymentModeNormalization = settings.EnvironmentDeploymentNormalization{}
		canonical.Policy.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{ProtectedBranches: true}
		if mode == "ui" {
			canonical.Policy.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}
		} // Missing acquired active patterns prevents UI conversion/save.
		observed, err := settings.NewEnvironmentObservation([]settings.EnvironmentState{z, canonical, a})
		if err != nil {
			t.Fatal(err)
		}
		reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
		cause := errors.New("environment acquisition failed")
		reads := 0
		reader.Environments = func(repository.Repository, []string) (settings.EnvironmentObservation, error) {
			reads++
			if mode == "read" {
				return observed, cause
			}
			if mode == "unacquired" {
				return settings.EnvironmentObservation{}, nil
			}
			return observed, nil
		}
		saved := false
		var got []warning
		checkout := ExportCheckout{Repository: checkoutRepository, IncludeEnvironments: mode != "unrequested", PrepareSave: func() (func(settings.Declaration) error, error) {
			return func(d settings.Declaration) error {
				ui, err := github.UIFromExportDeclaration(d)
				if err != nil {
					return err
				}
				if mode != "unrequested" {
					entries := *ui.Settings.Environments
					if len(entries) != 3 || entries[0].Name != "a" || entries[1].Name != "canonical" || entries[2].Name != "Z" || *entries[0].DeploymentBranchesAndTags != uimodel.NoRestriction || *entries[1].DeploymentBranchesAndTags != uimodel.ProtectedBranchesOnly || *entries[2].DeploymentBranchesAndTags != uimodel.NoRestriction || entries[0].DeploymentBranchOrTagRules != nil || entries[2].DeploymentBranchOrTagRules != nil {
						t.Fatal("effective mode/export order/latent privacy", entries)
					}
				}
				saved = true
				return nil
			}, nil
		}}
		if mode != "nil sink" {
			checkout.EnvironmentDeploymentNormalizationWarning = func(name string, r settings.EnvironmentDeploymentNormalization) { got = append(got, warning{name, r}) }
		}
		err = Export(checkout, reader)
		var want []warning
		if mode == "selected" || mode == "ui" {
			want = []warning{{"a", settings.EnvironmentDeploymentNormalization{Path: settings.EnvironmentDeploymentModePath, RawPolicy: settings.EnvironmentBranchPolicy{}, Fallback: uimodel.NoRestriction}}, {"Z", settings.EnvironmentDeploymentNormalization{Path: settings.EnvironmentDeploymentModePath, RawPolicy: settings.EnvironmentBranchPolicy{ProtectedBranches: true, CustomBranchPolicies: true}, Fallback: uimodel.NoRestriction}}}
		}
		failed := mode == "read" || mode == "unacquired" || mode == "ui"
		wantReads := 1
		if mode == "unrequested" {
			wantReads = 0
		}
		if (err != nil) != failed || saved == failed || reads != wantReads || !reflect.DeepEqual(got, want) {
			t.Fatal(mode, saved, reads, got, want, err)
		}
		if mode == "read" && !errors.Is(err, cause) {
			t.Fatal("read cause lost", err)
		}
	}
}
