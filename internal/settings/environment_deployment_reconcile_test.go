package settings

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestEnvironmentDeploymentReconciliationScopesEvidenceToExplicitMode(t *testing.T) {
	for _, raw := range []bool{false, true} {
		for _, choice := range []uimodel.EnvironmentDeploymentChoice{uimodel.NoRestriction, uimodel.ProtectedBranchesOnly, uimodel.SelectedBranchesAndTags} {
			for _, partial := range []bool{false, true} {
				state := normalizedEnvironmentModeState(t, 7, "production", raw)
				unlisted := normalizedEnvironmentModeState(t, 9, "UNMANAGED", !raw)
				observed, err := NewEnvironmentObservation([]EnvironmentState{unlisted, state})
				if err != nil {
					t.Fatal(err)
				}
				after := cloneEnvironment(state.Policy)
				if partial {
					after = EnvironmentDeclaration{Name: "PRODUCTION", DeploymentBranchPolicyDeclared: true}
				}
				policy, _ := EnvironmentDeploymentPolicyForChoice(choice)
				after.DeploymentBranchPolicy = policy
				desired, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{after})
				if err != nil {
					t.Fatal(err)
				}
				changes, err := ReconcileEnvironments(observed, &desired)
				if err != nil || len(changes) != 1 || changes[0].Operation() != "update" || changes[0].Name() != "production" || changes[0].DeploymentModeNormalization() != state.DeploymentModeNormalization || !reflect.DeepEqual(changes[0].Before(), &state.Policy) {
					t.Fatal("mode intent/evidence/identity lost", changes, err)
				}
				want := desired.Values()[0]
				want.Name = "production"
				if !reflect.DeepEqual(changes[0].After(), &want) || changes[0].Before().DeploymentBranchPatterns != nil {
					t.Fatal("partial request or latent privacy changed", changes)
				}
				for _, selection := range []*EnvironmentDeclarations{nil, {}} {
					if changes, err := ReconcileEnvironments(observed, selection); err != nil || len(changes) != 0 {
						t.Fatal("omission managed resources", changes, err)
					}
				}
				omitted, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{{Name: "PRODUCTION", WaitTimer: paramInt(5)}})
				if err != nil {
					t.Fatal(err)
				}
				changes, err = ReconcileEnvironments(observed, &omitted)
				if err != nil || len(changes) != 1 || changes[0].DeploymentModeNormalization() != (EnvironmentDeploymentNormalization{}) || changes[0].After().DeploymentBranchPolicyDeclared || changes[0].After().DeploymentBranchPolicy != nil || !EnvironmentHasAPIOmissions(*changes[0].After()) {
					t.Fatal("omitted mode gained evidence or payload", changes, err)
				}
			}
		}
	}
	canonical := EnvironmentState{ID: 7, Policy: environmentPolicy(t, "production")}
	observed, err := NewEnvironmentObservation([]EnvironmentState{canonical})
	if err != nil {
		t.Fatal(err)
	}
	desired, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{canonical.Policy})
	if err != nil {
		t.Fatal(err)
	}
	if changes, err := ReconcileEnvironments(observed, &desired); err != nil || len(changes) != 0 {
		t.Fatal("canonical equality changed", changes, err)
	}
	nameOnly, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{{Name: "production"}})
	if err != nil {
		t.Fatal(err)
	}
	changes, err := ReconcileEnvironments(observed, &nameOnly)
	if err != nil || len(changes) != 1 || !EnvironmentHasAPIOmissions(*changes[0].After()) || changes[0].DeploymentModeNormalization() != (EnvironmentDeploymentNormalization{}) {
		t.Fatal("name-only request became read-only assertion", changes, err)
	}
}

func TestEnvironmentDeploymentReconciliationRequiresExplicitSelectedModeForPatterns(t *testing.T) {
	for _, raw := range []bool{false, true} {
		state := normalizedEnvironmentModeState(t, 7, "production", raw)
		observed, err := NewEnvironmentObservation([]EnvironmentState{state})
		if err != nil {
			t.Fatal(err)
		}
		patterns := []EnvironmentPattern{{Type: "branch", Name: "release/*"}}
		desired, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{{Name: "production", DeploymentBranchPatterns: &patterns}})
		if err != nil {
			t.Fatal(err)
		}
		if changes, err := ReconcileEnvironments(observed, &desired); err == nil || len(changes) != 0 {
			t.Fatal("fallback or latent raw custom flag proved applicability", changes, err)
		}
		selected := EnvironmentBranchPolicy{CustomBranchPolicies: true}
		desired, err = NewEnvironmentDeclarations([]EnvironmentDeclaration{{Name: "production", DeploymentBranchPolicy: &selected, DeploymentBranchPatterns: &patterns}})
		if err != nil {
			t.Fatal(err)
		}
		changes, err := ReconcileEnvironments(observed, &desired)
		if err != nil || len(changes) != 1 || changes[0].DeploymentModeNormalization() != state.DeploymentModeNormalization || !reflect.DeepEqual(changes[0].After().DeploymentBranchPatterns, &patterns) {
			t.Fatal("explicit selected intent lost", changes, err)
		}
	}
}
