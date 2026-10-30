package app

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/github"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func exportEnvironmentRefTypeState(t *testing.T, id int64, name string, patterns []settings.EnvironmentPatternState) settings.EnvironmentState {
	t.Helper()
	state := exportEnvironmentModeState(t, id, name, false)
	state.DeploymentModeNormalization = settings.EnvironmentDeploymentNormalization{}
	state.Policy.DeploymentBranchPolicy = &settings.EnvironmentBranchPolicy{CustomBranchPolicies: true}
	state.Patterns = patterns
	effective, err := settings.EffectiveEnvironmentPatterns(patterns)
	if err != nil {
		t.Fatal(err)
	}
	state.Policy.DeploymentBranchPatterns = &effective
	return state
}

func TestEnvironmentRefTypeExportCollisionAndAcquisitionFailureNeverSave(t *testing.T) {
	future := settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: "future", Fallback: "branch"}
	a := exportEnvironmentRefTypeState(t, 1, "a", []settings.EnvironmentPatternState{{ID: 3, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}, RefTypeNormalization: future}})
	collision := exportEnvironmentRefTypeState(t, 2, "b", []settings.EnvironmentPatternState{{ID: 9, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}, RefTypeNormalization: future}})
	collision.Patterns = append(collision.Patterns, settings.EnvironmentPatternState{ID: 10, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}})
	collision.Policy.DeploymentBranchPatterns = nil
	z := exportEnvironmentRefTypeState(t, 4, "Z", []settings.EnvironmentPatternState{{ID: 6, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}, RefTypeNormalization: future}})
	observed, err := settings.NewEnvironmentObservation([]settings.EnvironmentState{z, collision, a})
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("incomplete environment acquisition")
	for _, mode := range []string{"collision", "read", "unacquired"} {
		for _, sink := range []bool{true, false} {
			reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
			reader.Environments = func(repository.Repository, []string) (settings.EnvironmentObservation, error) {
				if mode == "read" {
					return observed, cause
				}
				if mode == "unacquired" {
					return settings.EnvironmentObservation{}, nil
				}
				return observed, nil
			}
			saved := false
			var got, want []string
			checkout := ExportCheckout{Repository: checkoutRepository, IncludeEnvironments: true, PrepareSave: func() (func(settings.Declaration) error, error) {
				return func(settings.Declaration) error { saved = true; return nil }, nil
			}}
			if sink {
				checkout.EnvironmentRefTypeNormalizationWarning = func(name string, pattern settings.EnvironmentPatternState) {
					if pattern.RefTypeNormalization != future || pattern.Pattern.Type != "branch" {
						t.Fatal("raw normalization evidence changed", pattern)
					}
					got = append(got, fmt.Sprintf("%s/%d/%s", name, pattern.ID, pattern.Pattern.Name))
				}
				if mode == "collision" {
					want = []string{"a/3/main", "b/9/main", "Z/6/main"}
				}
			}
			err := Export(checkout, reader)
			if err == nil || saved || !reflect.DeepEqual(got, want) {
				t.Fatal("failed materialization/acquisition escaped as export", mode, sink, saved, got, want, err)
			}
			if mode == "read" && !errors.Is(err, cause) {
				t.Fatal("read failure cause lost", err)
			}
		}
	}
}

func TestEnvironmentRefTypeExportWarningsAndEffectivePolicy(t *testing.T) {
	type warning struct {
		name    string
		pattern settings.EnvironmentPatternState
	}
	future := settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: "future", Fallback: "branch"}
	empty := settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: "", Fallback: "branch"}
	for _, sink := range []bool{true, false} {
		z := exportEnvironmentRefTypeState(t, 7, "Z", []settings.EnvironmentPatternState{{ID: 5, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "z"}, RefTypeNormalization: future}})
		a := exportEnvironmentRefTypeState(t, 9, "a", []settings.EnvironmentPatternState{
			{ID: 9, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}, RefTypeNormalization: future},
			{ID: 8, Pattern: settings.EnvironmentPattern{Type: "tag", Name: "release"}},
			{ID: 2, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "dev"}, RefTypeNormalization: empty},
		})
		latent := exportEnvironmentModeState(t, 11, "latent", true)
		latent.Patterns[0].RefTypeNormalization = future
		observed, err := settings.NewEnvironmentObservation([]settings.EnvironmentState{z, latent, a})
		if err != nil {
			t.Fatal(err)
		}
		reader := exportFixtureReader(t, settings.RepositoryState{DefaultBranch: "main"}, func(string, repository.Repository) error { return nil })
		reader.Environments = func(repository.Repository, []string) (settings.EnvironmentObservation, error) { return observed, nil }
		var got, want []warning
		if sink {
			want = []warning{
				{"a", settings.EnvironmentPatternState{ID: 2, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "dev"}, RefTypeNormalization: empty}},
				{"a", settings.EnvironmentPatternState{ID: 9, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}, RefTypeNormalization: future}},
				{"Z", settings.EnvironmentPatternState{ID: 5, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "z"}, RefTypeNormalization: future}},
			}
		}
		saved := false
		checkout := ExportCheckout{Repository: checkoutRepository, IncludeEnvironments: true, PrepareSave: func() (func(settings.Declaration) error, error) {
			return func(d settings.Declaration) error {
				if !reflect.DeepEqual(got, want) {
					t.Fatal("save preceded scoped warnings", got, want)
				}
				ui, err := github.UIFromExportDeclaration(d)
				if err != nil {
					return err
				}
				entries := *ui.Settings.Environments
				if len(entries) != 3 || entries[0].Name != "a" || entries[1].Name != "latent" || entries[2].Name != "Z" || !reflect.DeepEqual(*entries[0].DeploymentBranchOrTagRules, []uimodel.EnvironmentDeploymentRule{{RefType: "branch", NamePattern: "dev"}, {RefType: "branch", NamePattern: "main"}, {RefType: "tag", NamePattern: "release"}}) || entries[1].DeploymentBranchOrTagRules != nil || *entries[1].DeploymentBranchesAndTags != uimodel.NoRestriction || !reflect.DeepEqual(*entries[2].DeploymentBranchOrTagRules, []uimodel.EnvironmentDeploymentRule{{RefType: "branch", NamePattern: "z"}}) {
					t.Fatal("effective export or latent privacy", entries)
				}
				saved = true
				return nil
			}, nil
		}}
		if sink {
			checkout.EnvironmentRefTypeNormalizationWarning = func(name string, pattern settings.EnvironmentPatternState) {
				got = append(got, warning{name, pattern})
			}
		}
		err = Export(checkout, reader)
		if err != nil || !saved || !reflect.DeepEqual(got, want) {
			t.Fatal("export projection or scoped warning order", saved, got, want, err)
		}
	}
}
