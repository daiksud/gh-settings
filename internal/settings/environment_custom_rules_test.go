package settings

import (
	"fmt"
	"github.com/cli/go-gh/v2/pkg/repository"
	"reflect"
	"strings"
	"testing"
)

func customRule(id int64) EnvironmentCustomRule {
	return EnvironmentCustomRule{ID: id + 100, AppID: id, NodeID: fmt.Sprint("rule", id), AppNodeID: fmt.Sprint("app", id), Slug: fmt.Sprint("custom-", id), IntegrationURL: "https://api.github.com/apps/custom"}
}

func TestEnvironmentCustomRulesOwnershipAndCapacity(t *testing.T) {
	current := []EnvironmentCustomRule{customRule(1), customRule(2)}
	for _, tc := range []struct {
		name   string
		want   *[]int64
		add    []int64
		remove []EnvironmentCustomRule
	}{
		{"omitted", nil, nil, nil}, {"reordered", actionsTestPointer([]int64{2, 1}), []int64{}, []EnvironmentCustomRule{}},
		{"clear", actionsTestPointer([]int64{}), []int64{}, current}, {"replace", actionsTestPointer([]int64{2, 3}), []int64{3}, []EnvironmentCustomRule{current[0]}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			add, remove, err := EnvironmentCustomRuleChanges(current, tc.want)
			if err != nil || !reflect.DeepEqual(add, tc.add) || !reflect.DeepEqual(remove, tc.remove) {
				t.Fatal(add, remove, err)
			}
		})
	}
	full := []EnvironmentCustomRule{}
	for id := int64(1); id <= 6; id++ {
		full = append(full, customRule(id))
	}
	if _, _, err := EnvironmentCustomRuleChanges(full, actionsTestPointer([]int64{2, 3, 4, 5, 6, 7})); err == nil || !strings.Contains(err.Error(), "first apply an explicit reduced") {
		t.Fatal(err)
	}
	for _, ids := range [][]int64{{0}, {-1}, {1, 1}, {1, 2, 3, 4, 5, 6, 7}} {
		if _, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{{Name: "prod", CustomRules: &ids}}); err == nil {
			t.Fatal("invalid App selection", ids)
		}
	}
}

func TestEnvironmentCustomRuleAcquisitionIsExplicitAndCopyOwned(t *testing.T) {
	state := EnvironmentState{ID: 7, Policy: environmentPolicy(t, "prod")}
	if _, err := NewEnvironmentObservation([]EnvironmentState{state}); err == nil {
		t.Fatal("missing custom observation became empty")
	}
	state.Policy.CustomRules = nil
	state.CustomRules = []EnvironmentCustomRule{customRule(3), customRule(1)}
	observed, err := NewEnvironmentObservation([]EnvironmentState{state})
	if err != nil {
		t.Fatal(err)
	}
	state.CustomRules[0].ID = 99
	values, _ := observed.Values()
	if !reflect.DeepEqual(*values[0].Policy.CustomRules, []int64{1, 3}) || values[0].CustomRules[0].ID == 99 {
		t.Fatal(values)
	}
	values[0].CustomRules[0].AppID = 99
	(*values[0].Policy.CustomRules)[0] = 99
	values, _ = observed.Values()
	if values[0].CustomRules[0].AppID == 99 || (*values[0].Policy.CustomRules)[0] == 99 {
		t.Fatal("aliased evidence")
	}
	requested := EnvironmentDeclaration{Name: "prod", CustomRules: actionsTestPointer([]int64{1, 3})}
	d, _ := NewEnvironmentDeclarations([]EnvironmentDeclaration{requested})
	changes, err := ReconcileEnvironments(observed, &d)
	if err != nil || len(changes) != 0 {
		t.Fatal(changes, err)
	}
	requested.CustomRules = actionsTestPointer([]int64{1})
	d, _ = NewEnvironmentDeclarations([]EnvironmentDeclaration{requested})
	changes, err = ReconcileEnvironments(observed, &d)
	if err != nil || len(changes) != 1 {
		t.Fatal(changes, err)
	}
	if match, err := EnvironmentBuiltinsMatchRequest(values[0], requested); err != nil || !match {
		t.Fatal("App change requested parent PUT", match, err)
	}
	if !SameEnvironmentCustomRules(values[0].CustomRules, []EnvironmentCustomRule{customRule(1), customRule(3)}) {
		t.Fatal("order changed identities")
	}
	changed := append([]EnvironmentCustomRule{}, values[0].CustomRules...)
	changed[0].ID++
	if SameEnvironmentCustomRules(values[0].CustomRules, changed) {
		t.Fatal("replacement binding matched")
	}
}

func TestEnvironmentPruneRejectsKnownNonemptyCustomRulesEvenWithTrueFlag(t *testing.T) {
	state := EnvironmentState{ID: 7, Policy: environmentPolicy(t, "prod"), CustomRules: []EnvironmentCustomRule{customRule(1)}}
	vars, _ := NewVariableInventory(nil)
	if _, err := NewEnvironmentPruneCandidate(repository.Repository{Owner: "owner", Name: "current"}, ArchiveState{ID: 1, FullName: "owner/current"}, state, vars, []EnvironmentSecretMetadata{}, true); err == nil {
		t.Fatal("caller flag authorized deletion of custom protections")
	}
}
