package settings

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestClassicDeploymentProjectionOwnershipAndPrune(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		raw := []string{"staging", "production"}
		state := classicModelState(7, "main")
		state.Companions.RequiresDeployments = enabled
		state.Companions.Deployments.Value = &raw
		group, err := ClassicDeploymentProjection(enabled, state.Companions.Deployments)
		if err != nil {
			t.Fatal(err)
		}
		state.Rule.RequireDeploymentsToSucceedBeforeMerging = group
		observed := ClassicObservation{RepositoryID: "R", Rules: []ClassicState{state}}
		exported, err := ExportClassicRules(observed)
		if err != nil {
			t.Fatal(err)
		}
		if !ClassicMatches(state.Rule, exported[0]) {
			t.Fatal("roundtrip changed group")
		}
		if enabled {
			if !reflect.DeepEqual(*exported[0].RequireDeploymentsToSucceedBeforeMerging.Environments, []string{"production", "staging"}) {
				t.Fatal(exported)
			}
			(*exported[0].RequireDeploymentsToSucceedBeforeMerging.Environments)[0] = "other"
			if (*group.Environments)[0] != "production" || raw[0] != "staging" {
				t.Fatal("raw/owned/export alias")
			}
		} else if group.Environments != nil {
			t.Fatal("inactive names leaked")
		}
		prepared, err := PrepareClassicPrune(observed, []ClassicRule{})
		if err != nil {
			t.Fatal(err)
		}
		changed := cloneClassicState(state)
		*changed.Companions.Deployments.Value = []string{"production", "staging"}
		if prepared.Recheck(ClassicObservation{RepositoryID: "R", Rules: []ClassicState{changed}}) == nil {
			t.Fatal("prune ignored raw order")
		}
		omitted := ClassicRule{ID: state.Rule.ID, BranchNamePattern: "main"}
		if VerifyClassicPreservation(state, changed, omitted) == nil {
			t.Fatal("omission ignored raw order")
		}
		selected := CloneClassicRule(state.Rule)
		if err := VerifyClassicPreservation(state, changed, selected); err != nil {
			t.Fatal("selected set unexpectedly owns raw order", err)
		}
	}
}

func TestClassicDeploymentStrictActiveUnion(t *testing.T) {
	yes, no := true, false
	empty, names, dup := []string{}, []string{" staging ", "production"}, []string{"production", "production"}
	for _, tc := range []struct {
		group *ClassicDeployments
		bad   bool
	}{
		{nil, false}, {&ClassicDeployments{}, false}, {&ClassicDeployments{Enabled: &no}, false},
		{&ClassicDeployments{Enabled: &yes, Environments: &empty}, false}, {&ClassicDeployments{Enabled: &yes, Environments: &names}, false},
		{&ClassicDeployments{Environments: &empty}, true}, {&ClassicDeployments{Enabled: &yes}, true},
		{&ClassicDeployments{Enabled: &no, Environments: &empty}, true}, {&ClassicDeployments{Enabled: &yes, Environments: &dup}, true},
	} {
		if (ValidateClassicDeployments(tc.group) != nil) != tc.bad {
			t.Fatal(tc)
		}
	}
	for _, raw := range []ClassicNullable[[]string]{{}, {Present: true}, {Present: true, Value: &dup}} {
		if _, err := ClassicDeploymentProjection(true, raw); err == nil {
			t.Fatal("unavailable active state accepted", raw)
		}
	}
}

func TestClassicDeploymentReportRejectsNullArrayMembers(t *testing.T) {
	var change ClassicChange
	if err := json.Unmarshal([]byte(`{"after":{"branch_name_pattern":"main","require_deployments_to_succeed_before_merging":{"enabled":true,"environments":[null]}}}`), &change); err == nil {
		t.Fatal("null report list member accepted")
	}
}
