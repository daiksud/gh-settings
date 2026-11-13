package settings

import "testing"

func TestEnvironmentMatchingPartialAndNameOnlySkipUpdates(t *testing.T) {
	before := environmentPolicy(t, "production")
	observation, err := NewEnvironmentObservation([]EnvironmentState{{ID: 1, Policy: before}})
	if err != nil {
		t.Fatal(err)
	}
	for _, requested := range []EnvironmentDeclaration{{Name: "production"}, {Name: "production", WaitTimer: paramInt(0)}} {
		desired, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{requested})
		if err != nil {
			t.Fatal(err)
		}
		changes, err := ReconcileEnvironments(observation, &desired)
		if err != nil || len(changes) != 0 {
			t.Fatalf("matching partial policy requires mutation: changes=%v err=%v", changes, err)
		}
	}
}

func TestEnvironmentUpdateCompanionsRemainSeparateFromIntent(t *testing.T) {
	before := environmentPolicy(t, "production")
	before.WaitTimer = paramInt(30)
	before.PreventSelfReview = paramBool(true)
	before.CanAdminsBypass = paramBool(true)
	before.Reviewers = &[]EnvironmentReviewer{{Type: "User", ID: 7}}
	requested := EnvironmentDeclaration{Name: "PRODUCTION", WaitTimer: paramInt(0)}
	state := EnvironmentState{ID: 1, Policy: before}
	observation, _ := NewEnvironmentObservation([]EnvironmentState{state})
	desired, _ := NewEnvironmentDeclarations([]EnvironmentDeclaration{requested})
	changes, err := ReconcileEnvironments(observation, &desired)
	if err != nil || len(changes) != 1 {
		t.Fatal(changes, err)
	}
	after := changes[0].After()
	if after.PreventSelfReview != nil || after.CanAdminsBypass != nil || after.Reviewers != nil || after.DeploymentBranchPolicyDeclared {
		t.Fatal("companions became owned intent", after)
	}
	expected, err := PrepareEnvironmentUpdate(state, *after)
	if err != nil || *expected.WaitTimer != 0 || !*expected.PreventSelfReview || !*expected.CanAdminsBypass || len(*expected.Reviewers) != 1 || !expected.DeploymentBranchPolicyDeclared || expected.DeploymentBranchPolicy != nil {
		t.Fatal(expected, err)
	}
	*expected.WaitTimer = 99
	(*expected.Reviewers)[0].ID = 99
	if *after.WaitTimer != 0 || (*before.Reviewers)[0].ID != 7 {
		t.Fatal("snapshot aliases caller state", after, before)
	}
}

func TestEnvironmentPartialDependencyConflictStopsReconciliation(t *testing.T) {
	for _, clearReviewers := range []bool{false, true} {
		before := environmentPolicy(t, "production")
		requested := EnvironmentDeclaration{Name: "production", PreventSelfReview: paramBool(true)}
		if clearReviewers {
			before.PreventSelfReview = paramBool(true)
			before.Reviewers = &[]EnvironmentReviewer{{Type: "User", ID: 7}}
			requested = EnvironmentDeclaration{Name: "production", Reviewers: &[]EnvironmentReviewer{}}
		}
		observed, _ := NewEnvironmentObservation([]EnvironmentState{{ID: 1, Policy: before}})
		desired, err := NewEnvironmentDeclarations([]EnvironmentDeclaration{requested})
		if err != nil {
			t.Fatal("partial dependency cannot be resolved until acquisition", err)
		}
		if changes, err := ReconcileEnvironments(observed, &desired); err == nil || len(changes) != 0 {
			t.Fatal("silently changed unowned dependency", changes, err)
		}
	}
}
