package settings

import "testing"

func TestCommitMessageDependencies(t *testing.T) {
	squashTitle, mergeTitle, message, disabled, enabled := "COMMIT_OR_PR_TITLE", "MERGE_MESSAGE", "BLANK", false, true
	for _, test := range []struct {
		name        string
		declaration RepositoryDeclaration
		wantError   string
	}{
		{name: "unmanaged"},
		{name: "squash title only", declaration: RepositoryDeclaration{SquashMergeCommitTitle: &squashTitle}},
		{name: "merge title only with unmanaged method", declaration: RepositoryDeclaration{MergeCommitTitle: &mergeTitle}},
		{name: "merge title with enabled method", declaration: RepositoryDeclaration{AllowMergeCommit: &enabled, MergeCommitTitle: &mergeTitle}},
		{name: "squash pair", declaration: RepositoryDeclaration{SquashMergeCommitTitle: &squashTitle, SquashMergeCommitMessage: &message}},
		{name: "merge pair", declaration: RepositoryDeclaration{MergeCommitTitle: &mergeTitle, MergeCommitMessage: &message}},
		{name: "both pairs", declaration: RepositoryDeclaration{SquashMergeCommitTitle: &squashTitle, SquashMergeCommitMessage: &message, MergeCommitTitle: &mergeTitle, MergeCommitMessage: &message}},
		{
			name: "squash message only", declaration: RepositoryDeclaration{SquashMergeCommitMessage: &message},
			wantError: "squash_merge_commit_message requires squash_merge_commit_title",
		},
		{
			name: "merge message only", declaration: RepositoryDeclaration{MergeCommitMessage: &message},
			wantError: "merge_commit_message requires merge_commit_title",
		},
		{
			name: "merge title cannot satisfy squash message", declaration: RepositoryDeclaration{MergeCommitTitle: &mergeTitle, SquashMergeCommitMessage: &message},
			wantError: "squash_merge_commit_message requires squash_merge_commit_title",
		},
		{
			name: "squash title cannot satisfy merge message", declaration: RepositoryDeclaration{SquashMergeCommitTitle: &squashTitle, MergeCommitMessage: &message},
			wantError: "merge_commit_message requires merge_commit_title",
		},
		{
			name: "squash dependency fails first", declaration: RepositoryDeclaration{SquashMergeCommitMessage: &message, MergeCommitMessage: &message},
			wantError: "squash_merge_commit_message requires squash_merge_commit_title",
		},
		{
			name: "disabled merge title", declaration: RepositoryDeclaration{AllowMergeCommit: &disabled, MergeCommitTitle: &mergeTitle},
			wantError: "repository.settings.general.pull_requests.merge_commit.default_commit_message requires repository.settings.general.pull_requests.allow_merge_commits: true",
		},
		{
			name: "disabled merge pair", declaration: RepositoryDeclaration{AllowMergeCommit: &disabled, MergeCommitTitle: &mergeTitle, MergeCommitMessage: &message},
			wantError: "repository.settings.general.pull_requests.merge_commit.default_commit_message requires repository.settings.general.pull_requests.allow_merge_commits: true",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := ""
			if err := test.declaration.ValidateCommitDependencies(); err != nil {
				got = err.Error()
			}
			if got != test.wantError {
				t.Fatalf("dependency error=%q, want %q", got, test.wantError)
			}
		})
	}
}
