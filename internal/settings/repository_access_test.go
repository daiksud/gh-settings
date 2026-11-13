package settings

import (
	"reflect"
	"testing"
)

func TestRepositoryAccessChangesUsesOnlySelectedRawValues(t *testing.T) {
	yes, no, all, limited, unknown := true, false, "ALL", "COLLABORATORS_ONLY", "FUTURE"
	before := RepositoryDeclaration{HasPullRequests: &yes, PullRequestCreationPolicy: &all, IssueCreationPolicy: &unknown, HasSponsorships: &no}
	desired := RepositoryDeclaration{HasPullRequests: &yes, PullRequestCreationPolicy: &limited, IssueCreationPolicy: &all, HasSponsorships: &yes}
	want := RepositoryDeclaration{PullRequestCreationPolicy: &limited, IssueCreationPolicy: &all, HasSponsorships: &yes}
	got, err := RepositoryAccessChanges(before, desired)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal("selected delta lost raw policy or matching scalar", got, err)
	}
	*got.PullRequestCreationPolicy = "caller mutation"
	if *desired.PullRequestCreationPolicy != "COLLABORATORS_ONLY" {
		t.Fatal("delta aliases desired state")
	}
	for _, value := range []string{"ALL", "FUTURE", "", " \t\n "} {
		for _, control := range []string{"PR", "Issue"} {
			var observed, intent RepositoryDeclaration
			if control == "PR" {
				observed.PullRequestCreationPolicy, intent.PullRequestCreationPolicy = &value, &all
			} else {
				observed.IssueCreationPolicy, intent.IssueCreationPolicy = &value, &all
			}
			change, err := RepositoryAccessChanges(observed, intent)
			if err != nil || change.RepositoryAccessEmpty() != (value == "ALL") {
				t.Fatal("raw fallback equality became a match", control, value, change, err)
			}
		}
	}
	if got, err := RepositoryAccessChanges(RepositoryDeclaration{}, RepositoryDeclaration{HasIssues: &yes}); err != nil || !got.Empty() {
		t.Fatal("unrelated REST intent acquired access", got, err)
	}
}

func TestRepositoryAccessChangesRequiresEveryDeclaredObservation(t *testing.T) {
	yes, no, all := true, false, "ALL"
	for _, desired := range []RepositoryDeclaration{{HasPullRequests: &no}, {HasSponsorships: &no}, {PullRequestCreationPolicy: &all}, {IssueCreationPolicy: &all}} {
		if got, err := RepositoryAccessChanges(RepositoryDeclaration{}, desired); err == nil || !got.Empty() {
			t.Fatal("missing managed observation accepted", got, err)
		}
	}
	for _, desired := range []RepositoryDeclaration{{HasPullRequests: &no}, {HasSponsorships: &no}} {
		changed, err := RepositoryAccessChanges(RepositoryDeclaration{HasPullRequests: &yes, HasSponsorships: &yes}, desired)
		if err != nil || !reflect.DeepEqual(changed, desired) {
			t.Fatal("literal false lost", changed, err)
		}
		matched, err := RepositoryAccessChanges(desired, desired)
		if err != nil || !matched.Empty() {
			t.Fatal("false did not match", matched, err)
		}
	}
}
