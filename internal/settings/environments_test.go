package settings

import (
	"encoding/json"
	"reflect"
	"testing"
)

func environmentPolicy(t *testing.T, name string) EnvironmentDeclaration {
	t.Helper()
	d, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{{Name: name}})
	if err != nil {
		t.Fatal(err)
	}
	return d.Values()[0]
}

func TestEnvironmentNamesSelectWholeObjectsWithoutDeletingOthers(t *testing.T) {
	before := environmentPolicy(t, "production")
	delay, prevent := 30, true
	before.WaitTimer, before.PreventSelfReview = &delay, &prevent
	before.Reviewers = &[]EnvironmentReviewer{{Type: "User", ID: 7}}
	before.DeploymentBranchPolicy = &EnvironmentBranchPolicy{ProtectedBranches: true}
	unlisted := environmentPolicy(t, "unlisted")
	observation, err := NewEnvironmentObservation([]EnvironmentState{{ID: 1, Policy: before}, {ID: 2, Policy: unlisted}})
	if err != nil {
		t.Fatal(err)
	}
	for _, empty := range []*EnvironmentDeclarations{nil, {}} {
		changes, err := ReconcileEnvironments(EnvironmentObservation{}, empty)
		if err != nil || len(changes) != 0 {
			t.Fatal("empty selection acquired state", changes, err)
		}
	}
	desired, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{{Name: "PRODUCTION"}})
	if err != nil {
		t.Fatal(err)
	}
	changes, err := ReconcileEnvironments(observation, &desired)
	if err != nil || len(changes) != 1 || changes[0].Operation() != "update" || changes[0].Name() != "production" {
		t.Fatal(changes, err)
	}
	after := changes[0].After()
	if *after.WaitTimer != 0 || *after.PreventSelfReview || len(*after.Reviewers) != 0 || after.DeploymentBranchPolicy != nil {
		t.Fatal("omission retained protection instead of fixed defaults", after)
	}
	if *changes[0].Before().WaitTimer != 30 || !*changes[0].Before().PreventSelfReview {
		t.Fatal("before values lost")
	}
	empty, err := NewEnvironmentObservation(nil)
	if err != nil {
		t.Fatal(err)
	}
	created, err := ReconcileEnvironments(empty, &desired)
	if err != nil || len(created) != 1 || created[0].Operation() != "create" || created[0].Before() != nil {
		t.Fatal(created, err)
	}
	createAfter := created[0].After()
	createAfter.Name = after.Name
	if !reflect.DeepEqual(createAfter, after) {
		t.Fatal("create/update defaults differ", createAfter, after)
	}
	nameChange, _ := NewEnvironmentDeclarations([]EnvironmentDeclaration{{Name: "another"}})
	next, err := ReconcileEnvironments(observation, &nameChange)
	if err != nil || len(next) != 1 || next[0].Operation() != "create" {
		t.Fatal("name change deleted/renamed an old object", next, err)
	}
	encoded, err := json.Marshal(after)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if _, exists := fields["id"]; exists {
		t.Fatal("environment metadata ID became declaration/writeback input", string(encoded))
	}
	if _, err := ReconcileEnvironments(EnvironmentObservation{}, &desired); err == nil {
		t.Fatal("missing observation became absence")
	}
}

func TestEnvironmentValidationAndCompleteObservation(t *testing.T) {
	for _, invalid := range []EnvironmentDeclaration{
		{Name: ""}, {Name: " production"}, {Name: "production\n"},
		{Name: "production", WaitTimer: paramInt(-1)}, {Name: "production", WaitTimer: paramInt(43201)},
		{Name: "production", PreventSelfReview: paramBool(true)},
		{Name: "production", Reviewers: &[]EnvironmentReviewer{{Type: "User", ID: 0}}},
		{Name: "production", Reviewers: &[]EnvironmentReviewer{{Type: "Bot", ID: 1}}},
		{Name: "production", Reviewers: &[]EnvironmentReviewer{{Type: "User", ID: 7}, {Type: "User", ID: 7}}},
		{Name: "production", DeploymentBranchPolicy: &EnvironmentBranchPolicy{}},
		{Name: "production", DeploymentBranchPolicy: &EnvironmentBranchPolicy{CustomBranchPolicies: true}},
	} {
		if _, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{invalid}); err == nil {
			t.Fatal("invalid environment accepted", invalid)
		}
	}
	if _, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{{Name: "Production"}, {Name: "production"}}); err == nil {
		t.Fatal("case duplicate accepted")
	}
	if _, err := NewEnvironmentObservation([]EnvironmentState{{ID: 1, Policy: EnvironmentDeclaration{Name: "production"}}}); err == nil {
		t.Fatal("missing protection observation became defaults")
	}
	complete := environmentPolicy(t, "production")
	if _, err := NewEnvironmentObservation([]EnvironmentState{{ID: 0, Policy: complete}}); err == nil {
		t.Fatal("missing remote identity accepted")
	}
	if _, err := NewEnvironmentObservation([]EnvironmentState{{ID: 1, Policy: complete}, {ID: 2, Policy: complete}}); err == nil {
		t.Fatal("ambiguous remote names accepted")
	}
}

func TestEnvironmentCanonicalCopiesAndReviewerOrder(t *testing.T) {
	input := EnvironmentDeclaration{Name: "production", WaitTimer: paramInt(10), Reviewers: &[]EnvironmentReviewer{{Type: "User", ID: 2}, {Type: "Team", ID: 1}}}
	desired, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{input})
	if err != nil {
		t.Fatal(err)
	}
	*input.WaitTimer = 99
	(*input.Reviewers)[0].ID = 99
	policy := desired.Values()[0]
	if *policy.WaitTimer != 10 {
		t.Fatal("external pointer retained")
	}
	observation, err := NewEnvironmentObservation([]EnvironmentState{{ID: 1, Policy: policy}})
	if err != nil {
		t.Fatal(err)
	}
	*policy.WaitTimer = 77
	(*policy.Reviewers)[0].ID = 77
	changes, err := ReconcileEnvironments(observation, &desired)
	if err != nil || len(changes) != 0 {
		t.Fatal("owned state mutated", changes, err)
	}
	reordered := desired.Values()[0]
	(*reordered.Reviewers)[0], (*reordered.Reviewers)[1] = (*reordered.Reviewers)[1], (*reordered.Reviewers)[0]
	round, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{reordered})
	if err != nil {
		t.Fatal(err)
	}
	changes, err = ReconcileEnvironments(observation, &round)
	if err != nil || len(changes) != 0 {
		t.Fatal("reviewer order became drift", changes, err)
	}
}
