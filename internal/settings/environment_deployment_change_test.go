package settings

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestNormalizedEnvironmentDeploymentChangeOwnsModeIntentAndPartialRequest(t *testing.T) {
	for _, raw := range []bool{false, true} {
		for _, choice := range []uimodel.EnvironmentDeploymentChoice{uimodel.NoRestriction, uimodel.ProtectedBranchesOnly, uimodel.SelectedBranchesAndTags} {
			for _, partial := range []bool{false, true} {
				state := normalizedEnvironmentModeState(t, 7, "production", raw)
				before, after := cloneEnvironment(state.Policy), cloneEnvironment(state.Policy)
				if partial {
					after = EnvironmentDeclaration{Name: "production", DeploymentBranchPolicyDeclared: true}
				}
				policy, ok := EnvironmentDeploymentPolicyForChoice(choice)
				if !ok {
					t.Fatal(choice)
				}
				after.DeploymentBranchPolicy = policy
				record := state.DeploymentModeNormalization
				change, err := NewEnvironmentChangeWithDeploymentNormalization("update", &before, &after, record)
				if err != nil || change.Operation() != "update" || change.Name() != "production" || change.DeploymentModeNormalization() != record || !reflect.DeepEqual(change.After(), &after) {
					t.Fatal(change, err)
				}
				if partial && !EnvironmentHasAPIOmissions(*change.After()) {
					t.Fatal("partial request supplied API defaults")
				}
				if !partial && choice == uimodel.NoRestriction {
					if _, err := NewEnvironmentChange("update", &before, &after); err == nil {
						t.Fatal("ordinary equality accepted")
					}
				}
				wantBefore, wantAfter := cloneEnvironment(before), cloneEnvironment(after)
				before.Name = "mutated input"
				after.Name = "mutated after"
				record.RawPolicy.CustomBranchPolicies = !raw
				result := change.DeploymentModeNormalization()
				result.RawPolicy.ProtectedBranches = !raw
				returned := change.Before()
				returned.Name = "mutated result"
				if !reflect.DeepEqual(change.Before(), &wantBefore) || !reflect.DeepEqual(change.After(), &wantAfter) || change.DeploymentModeNormalization() != state.DeploymentModeNormalization {
					t.Fatal("mode change alias", change)
				}
				if change.Before().DeploymentBranchPatterns != nil {
					t.Fatal("latent patterns replayed into before policy")
				}
			}
		}
	}
}

func TestNormalizedEnvironmentDeploymentChangeRejectsUnownedAndForgedEvidence(t *testing.T) {
	for _, mutate := range []func(*string, **EnvironmentDeclaration, **EnvironmentDeclaration, *EnvironmentDeploymentNormalization){
		func(op *string, b, a **EnvironmentDeclaration, r *EnvironmentDeploymentNormalization) {
			*op = "create"
			*b = nil
		},
		func(op *string, b, a **EnvironmentDeclaration, r *EnvironmentDeploymentNormalization) { *b = nil },
		func(op *string, b, a **EnvironmentDeclaration, r *EnvironmentDeploymentNormalization) { *a = nil },
		func(op *string, b, a **EnvironmentDeclaration, r *EnvironmentDeploymentNormalization) {
			(*a).Name = "PRODUCTION"
		},
		func(op *string, b, a **EnvironmentDeclaration, r *EnvironmentDeploymentNormalization) {
			(*a).DeploymentBranchPolicyDeclared = false
			(*a).WaitTimer = paramInt(5)
		},
		func(op *string, b, a **EnvironmentDeclaration, r *EnvironmentDeploymentNormalization) {
			(*a).DeploymentBranchPolicy = &EnvironmentBranchPolicy{}
		},
		func(op *string, b, a **EnvironmentDeclaration, r *EnvironmentDeploymentNormalization) {
			(*b).WaitTimer = nil
		},
		func(op *string, b, a **EnvironmentDeclaration, r *EnvironmentDeploymentNormalization) {
			(*b).DeploymentBranchPolicy = &EnvironmentBranchPolicy{ProtectedBranches: true}
		},
		func(op *string, b, a **EnvironmentDeclaration, r *EnvironmentDeploymentNormalization) {
			(*b).DeploymentBranchPolicyDeclared = false
		},
		func(op *string, b, a **EnvironmentDeclaration, r *EnvironmentDeploymentNormalization) {
			r.Path = "other"
		},
		func(op *string, b, a **EnvironmentDeclaration, r *EnvironmentDeploymentNormalization) {
			r.Fallback = uimodel.ProtectedBranchesOnly
		},
		func(op *string, b, a **EnvironmentDeclaration, r *EnvironmentDeploymentNormalization) {
			r.RawPolicy.CustomBranchPolicies = false
		},
		func(op *string, b, a **EnvironmentDeclaration, r *EnvironmentDeploymentNormalization) {
			*r = EnvironmentDeploymentNormalization{}
		},
	} {
		state := normalizedEnvironmentModeState(t, 7, "production", true)
		before, after := cloneEnvironment(state.Policy), cloneEnvironment(state.Policy)
		b, a := &before, &after
		op, record := "update", state.DeploymentModeNormalization
		mutate(&op, &b, &a, &record)
		change, err := NewEnvironmentChangeWithDeploymentNormalization(op, b, a, record)
		if err == nil || !reflect.DeepEqual(change, EnvironmentChange{}) {
			t.Fatal("forged/unowned mode change accepted", change, err)
		}
	}
	// A real omitted-mode API-controlled update remains ordinary and carries no raw mode evidence.
	before := environmentPolicy(t, "production")
	after := EnvironmentDeclaration{Name: "production", WaitTimer: paramInt(5)}
	change, err := NewEnvironmentChange("update", &before, &after)
	if err != nil || change.DeploymentModeNormalization() != (EnvironmentDeploymentNormalization{}) || change.After().DeploymentBranchPolicyDeclared || change.After().DeploymentBranchPolicy != nil {
		t.Fatal("omission became normalization intent", change, err)
	}
}
