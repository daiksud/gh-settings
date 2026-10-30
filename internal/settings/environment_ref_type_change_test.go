package settings

import (
	"reflect"
	"testing"
)

func TestEnvironmentRefTypeChangeRetainsOwnedRawCollectionAndPartialIntent(t *testing.T) {
	for _, collision := range []bool{false, true} {
		for _, partial := range []bool{false, true} {
			for _, clear := range []bool{false, true} {
				state := normalizedPatternEnvironment(t, collision)
				before := cloneEnvironment(state.Policy)
				after := cloneEnvironment(before)
				if partial {
					after = EnvironmentDeclaration{Name: "production"}
				}
				patterns := []EnvironmentPattern{{Type: "branch", Name: "main"}}
				if clear {
					patterns = []EnvironmentPattern{}
				}
				after.DeploymentBranchPatterns = &patterns
				wantBefore, wantAfter := cloneEnvironment(before), cloneEnvironment(after)
				wantEvidence := append([]EnvironmentPatternState{}, state.Patterns...)
				change, err := NewEnvironmentChangeWithPatternNormalizations("update", &before, &after, state.Patterns)
				if err != nil || change.Operation() != "update" || change.Name() != "production" || !reflect.DeepEqual(change.Before(), &wantBefore) || !reflect.DeepEqual(change.After(), &wantAfter) || !reflect.DeepEqual(change.PatternObservations(), wantEvidence) || change.DeploymentModeNormalization() != (EnvironmentDeploymentNormalization{}) {
					t.Fatal("closed pattern evidence/intent lost", change, err)
				}
				if !collision && !partial && !clear {
					if _, err := NewEnvironmentChange("update", &before, &after); err == nil {
						t.Fatal("ordinary equal update accepted")
					}
				}
				state.Patterns[0].RefTypeNormalization.RawValue = "input changed"
				returned := change.PatternObservations()
				returned[0].ID = 99
				returned[0].RefTypeNormalization.RawValue = "output changed"
				before.Name, after.Name = "changed", "changed"
				if !reflect.DeepEqual(change.PatternObservations(), wantEvidence) || !reflect.DeepEqual(change.Before(), &wantBefore) || !reflect.DeepEqual(change.After(), &wantAfter) {
					t.Fatal("pattern evidence aliases input/output", change)
				}
			}
		}
	}
}

func TestEnvironmentRefTypeChangeRejectsUnownedIncompleteAndForgedEvidence(t *testing.T) {
	for _, mutate := range []func(*string, **EnvironmentDeclaration, **EnvironmentDeclaration, *[]EnvironmentPatternState){
		func(op *string, b, a **EnvironmentDeclaration, p *[]EnvironmentPatternState) {
			*op = "create"
			*b = nil
		},
		func(op *string, b, a **EnvironmentDeclaration, p *[]EnvironmentPatternState) { *b = nil },
		func(op *string, b, a **EnvironmentDeclaration, p *[]EnvironmentPatternState) { *a = nil },
		func(op *string, b, a **EnvironmentDeclaration, p *[]EnvironmentPatternState) {
			(*a).Name = "PRODUCTION"
		},
		func(op *string, b, a **EnvironmentDeclaration, p *[]EnvironmentPatternState) {
			(*a).DeploymentBranchPatterns = nil
			(*a).WaitTimer = paramInt(5)
		},
		func(op *string, b, a **EnvironmentDeclaration, p *[]EnvironmentPatternState) {
			(*b).DeploymentBranchPolicy = nil
			(*b).DeploymentBranchPatterns = nil
		},
		func(op *string, b, a **EnvironmentDeclaration, p *[]EnvironmentPatternState) { *p = nil },
		func(op *string, b, a **EnvironmentDeclaration, p *[]EnvironmentPatternState) {
			*p = []EnvironmentPatternState{}
		},
		func(op *string, b, a **EnvironmentDeclaration, p *[]EnvironmentPatternState) {
			(*p)[0].RefTypeNormalization = ObservationNormalization{}
		},
		func(op *string, b, a **EnvironmentDeclaration, p *[]EnvironmentPatternState) {
			(*p)[0].RefTypeNormalization.RawValue = "branch"
		},
		func(op *string, b, a **EnvironmentDeclaration, p *[]EnvironmentPatternState) { (*p)[0].ID = 0 },
		func(op *string, b, a **EnvironmentDeclaration, p *[]EnvironmentPatternState) {
			*p = append(*p, (*p)[0])
		},
		func(op *string, b, a **EnvironmentDeclaration, p *[]EnvironmentPatternState) {
			(*b).DeploymentBranchPatterns = nil
		},
	} {
		state := normalizedPatternEnvironment(t, false)
		op, before, after, patterns := "update", cloneEnvironment(state.Policy), cloneEnvironment(state.Policy), state.Patterns
		b, a := &before, &after
		mutate(&op, &b, &a, &patterns)
		if got, err := NewEnvironmentChangeWithPatternNormalizations(op, b, a, patterns); err == nil || !reflect.DeepEqual(got, EnvironmentChange{}) {
			t.Fatal("invalid pattern evidence accepted", got, err)
		}
	}
}
