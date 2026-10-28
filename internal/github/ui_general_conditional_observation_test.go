package github

import (
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"testing"
)

func TestUIGeneralConditionalObservationVisibilityAndRaw(t *testing.T) {
	for _, gate := range []*bool{nil, pointer(true), pointer(false)} {
		source := settings.RepositoryDeclaration{HasIssues: gate, HasPullRequests: gate, AllowMergeCommit: gate, AllowSquashMerge: gate, IssueCreationPolicy: pointer("ALL"), PullRequestCreationPolicy: pointer("COLLABORATORS_ONLY"), MergeCommitTitle: pointer("PR_TITLE"), MergeCommitMessage: pointer("PR_BODY"), SquashMergeCommitTitle: pointer("PR_TITLE"), SquashMergeCommitMessage: pointer("BLANK")}
		view, err := GeneralObservation(source)
		if err != nil {
			t.Fatal(err)
		}
		hidden := gate != nil && !*gate
		if (view.Features.IssuePermissions.CreationAllowedBy == nil) != hidden || (view.Features.PullRequestPermissions.CreationAllowedBy == nil) != hidden || (view.PullRequests.MergeCommit.DefaultCommitMessage == nil) != hidden || (view.PullRequests.SquashMerge.DefaultCommitMessage == nil) != hidden {
			t.Fatal("conditional visibility incorrect", view)
		}
		if !hidden && (string(*view.Features.IssuePermissions.CreationAllowedBy) != "all_users" || string(*view.Features.PullRequestPermissions.CreationAllowedBy) != "collaborators_only" || string(*view.PullRequests.MergeCommit.DefaultCommitMessage) != "pull_request_title_and_description" || string(*view.PullRequests.SquashMerge.DefaultCommitMessage) != "pull_request_title") {
			t.Fatal(view)
		}
		raw := view.RawConditionalValues
		expected := GeneralConditionalValues{IssueCreationPolicy: pointer("ALL"), PullRequestCreationPolicy: pointer("COLLABORATORS_ONLY"), MergeCommitTitle: pointer("PR_TITLE"), MergeCommitMessage: pointer("PR_BODY"), SquashMergeCommitTitle: pointer("PR_TITLE"), SquashMergeCommitMessage: pointer("BLANK")}
		if !reflect.DeepEqual(raw, expected) {
			t.Fatal("latent or unacquired state lost", raw)
		}
		for _, value := range []*string{source.IssueCreationPolicy, source.PullRequestCreationPolicy, source.MergeCommitTitle, source.MergeCommitMessage, source.SquashMergeCommitTitle, source.SquashMergeCommitMessage} {
			*value = "changed"
		}
		if !reflect.DeepEqual(view.RawConditionalValues, expected) {
			t.Fatal("raw metadata aliases source")
		}
	}

	mixed, err := GeneralObservation(settings.RepositoryDeclaration{HasIssues: pointer(false), HasPullRequests: pointer(true), AllowMergeCommit: pointer(false), AllowSquashMerge: pointer(true), IssueCreationPolicy: pointer("ALL"), PullRequestCreationPolicy: pointer("COLLABORATORS_ONLY"), MergeCommitTitle: pointer("PR_TITLE"), MergeCommitMessage: pointer("PR_BODY"), SquashMergeCommitTitle: pointer("PR_TITLE"), SquashMergeCommitMessage: pointer("BLANK")})
	if err != nil || mixed.Features.IssuePermissions.CreationAllowedBy != nil || mixed.Features.PullRequestPermissions.CreationAllowedBy == nil || mixed.PullRequests.MergeCommit.DefaultCommitMessage != nil || mixed.PullRequests.SquashMerge.DefaultCommitMessage == nil {
		t.Fatal("independent conditional gates crossed", mixed, err)
	}
	empty, err := GeneralObservation(settings.RepositoryDeclaration{})
	if err != nil || !reflect.DeepEqual(empty.RawConditionalValues, GeneralConditionalValues{}) {
		t.Fatal("unacquired values defaulted", empty, err)
	}
	for _, source := range []settings.RepositoryDeclaration{{HasIssues: pointer(false), IssueCreationPolicy: pointer("UNKNOWN")}, {AllowMergeCommit: pointer(false), MergeCommitTitle: pointer("UNKNOWN"), MergeCommitMessage: pointer("BLANK")}} {
		if _, err := GeneralObservation(source); err == nil {
			t.Fatal("hidden raw unknown choice bypassed validation")
		}
	}
}
