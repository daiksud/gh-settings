package github

import (
	"reflect"
	"testing"

	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUIEnvironmentNormalizedRefTypeOwnsProjectionAndRawMetadata(t *testing.T) {
	state := observedEnvironmentRefTypeState(t, "future", false)
	observation, err := settings.NewEnvironmentObservation([]settings.EnvironmentState{state})
	if err != nil {
		t.Fatal(err)
	}
	view, err := EnvironmentsObservation(observation)
	if err != nil || len(view) != 1 {
		t.Fatal("complete normalized patterns rejected", view, err)
	}
	if !reflect.DeepEqual(*view[0].Environment.DeploymentBranchOrTagRules, []uimodel.EnvironmentDeploymentRule{{RefType: "branch", NamePattern: "main"}}) {
		t.Fatal("normalized branch UI projection missing", view)
	}
	if !reflect.DeepEqual(view[0].Patterns, state.Patterns) {
		t.Fatal("raw identity metadata changed", view[0].Patterns)
	}
	raw, err := view[0].Patterns[0].RawPattern()
	if err != nil || raw != (settings.EnvironmentPattern{Type: "future", Name: "main"}) || view[0].Patterns[0].ID != 7 {
		t.Fatal("raw identity replaced by fallback", raw, err)
	}
	view[0].Patterns[0].ID = 99
	view[0].Patterns[0].Pattern.Name = "changed"
	view[0].Patterns[0].RefTypeNormalization.RawValue = "changed"
	(*view[0].Environment.DeploymentBranchOrTagRules)[0].NamePattern = "changed"
	unchanged, err := observation.Values()
	if err != nil || len(unchanged) != 1 || !reflect.DeepEqual(unchanged[0], state) {
		t.Fatal("adapter output changed acquired state", unchanged, err)
	}
	fresh, err := EnvironmentsObservation(observation)
	if err != nil || len(fresh) != 1 || !reflect.DeepEqual(fresh[0].Patterns, state.Patterns) || !reflect.DeepEqual(*fresh[0].Environment.DeploymentBranchOrTagRules, []uimodel.EnvironmentDeploymentRule{{RefType: "branch", NamePattern: "main"}}) {
		t.Fatal("adapter output was shared across calls", fresh, err)
	}
}

func TestUIEnvironmentNormalizedRefTypeCollisionReturnsNoPartialCollection(t *testing.T) {
	state := observedEnvironmentRefTypeState(t, "future", true)
	observation, err := settings.NewEnvironmentObservation([]settings.EnvironmentState{{ID: 11, Policy: uiEnvironmentObservedPolicy("alpha")}, state})
	if err != nil {
		t.Fatal(err)
	}
	view, err := EnvironmentsObservation(observation)
	if err == nil || view != nil {
		t.Fatal("colliding projection returned a partial collection", view, err)
	}
}
