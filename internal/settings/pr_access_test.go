package settings

import "testing"

func TestPRAccessPresenceComparisonAndCompleteObservation(t *testing.T) {
	no := false
	limited := "COLLABORATORS_ONLY"
	desired := RepositoryDeclaration{HasPullRequests: &no, PullRequestCreationPolicy: &limited}
	if desired.Empty() || desired.PRAccessEmpty() {
		t.Fatal("explicit false lost")
	}
	state := PRAccessState{HasPullRequests: true, PullRequestCreationPolicy: "ALL"}
	observed := NewRepositoryObservation(nil, nil, &state)
	result, err := CompareRepositoryObservation(observed, desired)
	if err != nil || result.Managed != 2 || len(result.Changes) != 2 || result.Changes[0].Field().Path() != "repository.has_pull_requests" {
		t.Fatal(result, err)
	}
	if _, err := CompareRepositoryObservation(NewRepositoryObservation(nil, nil, nil), desired); err == nil {
		t.Fatal("unread false became observed false")
	}
	if _, err := observed.Declaration(); err == nil {
		t.Fatal("partial observation exported")
	}
	if result, err := CompareRepositoryObservation(observed, RepositoryDeclaration{HasPullRequests: &state.HasPullRequests}); err != nil || result.Managed != 1 || len(result.Changes) != 0 {
		t.Fatal(result, err)
	}
	for _, value := range []string{"ALL", "COLLABORATORS_ONLY"} {
		if !ValidPullRequestCreationPolicy(value) {
			t.Fatal(value)
		}
	}
	for _, value := range []string{"", "all", "UNKNOWN", " ALL"} {
		if ValidPullRequestCreationPolicy(value) {
			t.Fatal(value)
		}
	}
}
