package settings

import "testing"

func TestIssueCreationPolicyIsIndependentAndAcquired(t *testing.T) {
	policy := "COLLABORATORS_ONLY"
	desired := RepositoryDeclaration{IssueCreationPolicy: &policy}
	if desired.Empty() || desired.RepositoryAccessEmpty() || desired.HasIssues != nil || !desired.PRAccessEmpty() {
		t.Fatal(desired)
	}
	before := NewRepositoryObservation(nil, nil, nil)
	observed := before.WithIssueCreationPolicy(IssueCreationPolicyState{Policy: "ALL"})
	if len(before.Fields()) != 0 || len(observed.Fields()) != 1 {
		t.Fatal("observation modified or companions invented")
	}
	got, err := CompareRepositoryObservation(observed, desired)
	if err != nil || got.Managed != 1 || len(got.Changes) != 1 || got.Changes[0].Field().Path() != "repository.issue_creation_policy" {
		t.Fatal(got, err)
	}
	if _, err := CompareRepositoryObservation(before, desired); err == nil {
		t.Fatal("unread policy acquired a default")
	}
	if _, err := observed.Declaration(); err == nil {
		t.Fatal("partial observation exported")
	}
	for _, value := range []string{"ALL", "COLLABORATORS_ONLY"} {
		if !ValidIssueCreationPolicy(value) {
			t.Fatal(value)
		}
	}
	for _, value := range []string{"", "all", "UNKNOWN", " ALL"} {
		if ValidIssueCreationPolicy(value) {
			t.Fatal(value)
		}
	}
}
