package settings

import (
	"reflect"
	"testing"
)

func TestRepositoryObservationDeclaration(t *testing.T) {
	empty, description, homepage := "", "Example", "https://example.com"
	enabled, disabled := true, false
	branch := "main"
	for _, test := range []struct {
		name  string
		state RepositoryState
		want  RepositoryDeclaration
	}{
		{
			name:  "empty metadata and false flags",
			state: RepositoryState{DefaultBranch: branch},
			want: RepositoryDeclaration{
				Description: &empty, Homepage: &empty, DefaultBranch: &branch,
				HasIssues: &disabled, HasProjects: &disabled, HasWiki: &disabled, HasDiscussions: &disabled, WebCommitSignoffRequired: &disabled, IsTemplate: &disabled,
			},
		},
		{
			name: "mixed values",
			state: RepositoryState{
				Description: description, Homepage: homepage, DefaultBranch: branch,
				HasIssues: true, HasProjects: false, HasWiki: true, HasDiscussions: true,
			},
			want: RepositoryDeclaration{
				Description: &description, Homepage: &homepage, DefaultBranch: &branch,
				HasIssues: &enabled, HasProjects: &disabled, HasWiki: &enabled, HasDiscussions: &enabled, WebCommitSignoffRequired: &disabled, IsTemplate: &disabled,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.state.Declaration()
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("declaration=%+v, want %+v", got, test.want)
			}
		})
	}
}

func TestCombinedRepositoryObservationDeclaration(t *testing.T) {
	base := RepositoryState{DefaultBranch: "main", Description: "Example", Homepage: "https://example.com", HasIssues: true, HasWiki: true}
	for _, test := range []struct {
		name  string
		state MergeMethodsState
	}{
		{
			name: "false booleans and PR bodies",
			state: MergeMethodsState{
				SquashMergeCommitTitle: "PR_TITLE", SquashMergeCommitMessage: "PR_BODY",
				MergeCommitTitle: "PR_TITLE", MergeCommitMessage: "PR_BODY",
			},
		},
		{
			name: "true booleans and commit defaults",
			state: MergeMethodsState{
				AllowMergeCommit: true, AllowSquashMerge: true, AllowRebaseMerge: true,
				DeleteBranchOnMerge: true, AllowAutoMerge: true, AllowUpdateBranch: true,
				SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES",
				MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE",
			},
		},
		{
			name: "mixed booleans and blank messages",
			state: MergeMethodsState{
				AllowMergeCommit: true, AllowRebaseMerge: true, AllowAutoMerge: true,
				SquashMergeCommitTitle: "PR_TITLE", SquashMergeCommitMessage: "BLANK",
				MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "BLANK",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := base.Declaration()
			got := before.WithMergeMethods(test.state)
			want := RepositoryDeclaration{
				Description: &base.Description, Homepage: &base.Homepage, DefaultBranch: &base.DefaultBranch,
				HasIssues: &base.HasIssues, HasProjects: &base.HasProjects, HasWiki: &base.HasWiki, HasDiscussions: &base.HasDiscussions, WebCommitSignoffRequired: &base.WebCommitSignoffRequired, IsTemplate: &base.IsTemplate,
				AllowMergeCommit: &test.state.AllowMergeCommit, AllowSquashMerge: &test.state.AllowSquashMerge, AllowRebaseMerge: &test.state.AllowRebaseMerge,
				DeleteBranchOnMerge: &test.state.DeleteBranchOnMerge, AllowAutoMerge: &test.state.AllowAutoMerge, AllowUpdateBranch: &test.state.AllowUpdateBranch,
				SquashMergeCommitTitle: &test.state.SquashMergeCommitTitle, SquashMergeCommitMessage: &test.state.SquashMergeCommitMessage,
				MergeCommitTitle: &test.state.MergeCommitTitle, MergeCommitMessage: &test.state.MergeCommitMessage,
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("declaration=%+v, want %+v", got, want)
			}
			access := PRAccessState{PullRequestCreationPolicy: "ALL"}
			complete, err := NewRepositoryObservation(&base, &test.state, &access).WithIssueCreationPolicy(IssueCreationPolicyState{Policy: "ALL"}).WithSponsorships(SponsorshipsState{Enabled: false}).Declaration()
			completeWant := want.WithPRAccess(access)
			issuePolicy := "ALL"
			completeWant.IssueCreationPolicy = &issuePolicy
			noSponsorships := false
			completeWant.HasSponsorships = &noSponsorships
			if !test.state.AllowMergeCommit {
				completeWant.MergeCommitTitle = nil
				completeWant.MergeCommitMessage = nil
			}
			if err != nil || !reflect.DeepEqual(complete, completeWant) {
				t.Fatalf("complete declaration=%+v error=%v, want %+v", complete, err, want)
			}
			if !reflect.DeepEqual(before, base.Declaration()) {
				t.Fatal("combining observations changed the input declaration")
			}
		})
	}
}
