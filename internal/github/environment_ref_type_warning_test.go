package github

import (
	"reflect"
	"slices"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
)

func TestEnvironmentRefTypeWarningsRequireOwnedCompleteActiveState(t *testing.T) {
	state := observedEnvironmentRefTypeState(t, "future", false)
	emptyRecord := settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: "", Fallback: "branch"}
	state.Patterns = append(state.Patterns,
		settings.EnvironmentPatternState{ID: 5, Pattern: settings.EnvironmentPattern{Type: "tag", Name: "main"}},
		settings.EnvironmentPatternState{ID: 3, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "dev"}, RefTypeNormalization: emptyRecord},
	)
	effective, err := settings.EffectiveEnvironmentPatterns(state.Patterns)
	if err != nil {
		t.Fatal(err)
	}
	state.Policy.DeploymentBranchPatterns = &effective
	for _, mode := range []string{"complete", "collision", "omitted", "other", "latent", "canonical", "missing", "forged"} {
		for _, sink := range []bool{true, false} {
			current := state
			current.Patterns = slices.Clone(state.Patterns)
			empty := []settings.EnvironmentPattern{}
			requested := settings.EnvironmentDeclaration{Name: "PRODUCTION", DeploymentBranchPatterns: &empty}
			switch mode {
			case "collision":
				current.Patterns = append(current.Patterns, settings.EnvironmentPatternState{ID: 9, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}})
				current.Policy.DeploymentBranchPatterns = nil
			case "omitted":
				requested.DeploymentBranchPatterns = nil
			case "other":
				requested.Name = "other"
			case "latent":
				current = observedEnvironmentModeState(t, true)
				current.Patterns = slices.Clone(state.Patterns)
			case "canonical":
				current = observedEnvironmentRefTypeState(t, "branch", false)
			case "missing":
				current.Policy.WaitTimer = nil
			case "forged":
				current.Patterns[2].RefTypeNormalization.Path = "wrong"
			}
			var got, want []settings.EnvironmentPatternState
			client := &EnvironmentsClient{}
			if sink {
				client.RefTypeNormalizationWarning = func(name string, pattern settings.EnvironmentPatternState) {
					if name != "production" {
						t.Fatal("diagnostic lost acquired display name", name)
					}
					got = append(got, pattern)
				}
				if mode == "complete" || mode == "collision" {
					want = []settings.EnvironmentPatternState{
						{ID: 3, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "dev"}, RefTypeNormalization: emptyRecord},
						{ID: 7, Pattern: settings.EnvironmentPattern{Type: "branch", Name: "main"}, RefTypeNormalization: settings.ObservationNormalization{Path: settings.EnvironmentRefTypePath, RawValue: "future", Fallback: "branch"}},
					}
				}
			}
			client.warnRefTypeNormalizations(current, requested)
			if !reflect.DeepEqual(got, want) {
				t.Fatal("ref-type diagnostic scope or acquired metadata", mode, sink, got, want)
			}
		}
	}
}
