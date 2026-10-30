package settings

import (
	"reflect"
	"testing"
)

func TestEnvironmentReconciliationScopesRefTypeEvidenceToManagedPatterns(t *testing.T) {
	for _, collision := range []bool{false, true} {
		state := normalizedPatternEnvironment(t, collision)
		unlisted := normalizedPatternEnvironment(t, false)
		unlisted.ID, unlisted.Policy.Name = 9, "UNMANAGED"
		observed, err := NewEnvironmentObservation([]EnvironmentState{unlisted, state})
		if err != nil {
			t.Fatal(err)
		}
		for _, partial := range []bool{false, true} {
			for _, patterns := range [][]EnvironmentPattern{{{Type: "branch", Name: "main"}}, {{Type: "tag", Name: "main"}}, {}} {
				after := cloneEnvironment(state.Policy)
				if partial {
					after = EnvironmentDeclaration{Name: "PRODUCTION"}
				}
				after.DeploymentBranchPatterns = &patterns
				desired, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{after})
				if err != nil {
					t.Fatal(err)
				}
				changes, err := ReconcileEnvironments(observed, &desired)
				want := desired.Values()[0]
				want.Name = "production"
				if err != nil || len(changes) != 1 || !reflect.DeepEqual(changes[0].PatternObservations(), state.Patterns) || !reflect.DeepEqual(changes[0].Before(), &state.Policy) || !reflect.DeepEqual(changes[0].After(), &want) || changes[0].DeploymentModeNormalization() != (EnvironmentDeploymentNormalization{}) {
					t.Fatal("pattern intent/evidence or partial payload lost", changes, err)
				}
			}
		}
		for _, partial := range []bool{false, true} {
			after := cloneEnvironment(state.Policy)
			after.DeploymentBranchPatterns = nil
			if partial {
				after = EnvironmentDeclaration{Name: "production", WaitTimer: paramInt(5)}
			}
			desired, _ := NewEnvironmentDeclarations([]EnvironmentDeclaration{after})
			changes, err := ReconcileEnvironments(observed, &desired)
			if err != nil || !partial && len(changes) != 0 || partial && (len(changes) != 1 || changes[0].PatternObservations() != nil || changes[0].After().DeploymentBranchPatterns != nil) {
				t.Fatal("unmanaged patterns became owned evidence", changes, err)
			}
		}
		if changes, err := ReconcileEnvironments(observed, nil); err != nil || len(changes) != 0 {
			t.Fatal("omitted environments gained ownership", changes, err)
		}
	}
}

func TestEnvironmentReconciliationKeepsLatentRefTypesOutOfPatternEvidenceAndCanonicalNoop(t *testing.T) {
	latent := normalizedEnvironmentModeState(t, 7, "production", true)
	latent.Patterns = normalizedPatternEnvironment(t, true).Patterns
	observed, _ := NewEnvironmentObservation([]EnvironmentState{latent})
	patterns := []EnvironmentPattern{{Type: "branch", Name: "main"}}
	after := EnvironmentDeclaration{Name: "production", DeploymentBranchPolicy: &EnvironmentBranchPolicy{CustomBranchPolicies: true}, DeploymentBranchPatterns: &patterns}
	desired, _ := NewEnvironmentDeclarations([]EnvironmentDeclaration{after})
	changes, err := ReconcileEnvironments(observed, &desired)
	if err != nil || len(changes) != 1 || changes[0].PatternObservations() != nil || changes[0].DeploymentModeNormalization() != latent.DeploymentModeNormalization {
		t.Fatal("latent ref types became active plan evidence", changes, err)
	}
	canonical := normalizedPatternEnvironment(t, false)
	canonical.Patterns[0].RefTypeNormalization = ObservationNormalization{}
	observed, _ = NewEnvironmentObservation([]EnvironmentState{canonical})
	desired, _ = NewEnvironmentDeclarations([]EnvironmentDeclaration{canonical.Policy})
	if changes, err := ReconcileEnvironments(observed, &desired); err != nil || len(changes) != 0 {
		t.Fatal("canonical repeat stopped being noop", changes, err)
	}
}
