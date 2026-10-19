package app

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func TestSquashCommitSelectiveReadDirectWriteAndLocalValidation(t *testing.T) {
	for _, input := range []string{"squash_merge_commit_title: PR_TITLE", "squash_merge_commit_title: PR_TITLE, squash_merge_commit_message: BLANK"} {
		appCheckout(t, "schemaVersion: 1\nsettings: {repository: {"+input+"}}", true)
		calls := 0
		got, err := Plan(checkoutDeclaration, checkoutRepository, Reader{MergeMethods: func(repository.Repository) (settings.MergeMethodsState, error) {
			calls++
			return settings.MergeMethodsState{SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES", MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE"}, nil
		}})
		if err != nil || calls != 1 || got.Comparison.Managed < 1 {
			t.Fatalf("plan=%+v calls=%d err=%v", got, calls, err)
		}
		result, err := Apply(checkoutDeclaration, checkoutRepository, Writer{Repository: func(_ repository.Repository, d settings.RepositoryDeclaration) error {
			calls++
			if d.SquashMergeCommitTitle == nil || *d.SquashMergeCommitTitle != "PR_TITLE" || !d.BaseEmpty() {
				t.Fatalf("desired=%+v", d)
			}
			return nil
		}})
		if err != nil || !result.Applied || calls != 2 {
			t.Fatalf("apply=%+v err=%v", result, err)
		}
	}
	for _, input := range []string{"squash_merge_commit_message: BLANK", "squash_merge_commit_title: null", "squash_merge_commit_title: ''"} {
		appCheckout(t, "schemaVersion: 1\nsettings: {repository: {"+input+"}}", false)
		if _, err := Plan(checkoutDeclaration, checkoutRepository, Reader{}); err == nil {
			t.Fatal("invalid plan reached target/read")
		}
		if _, err := Apply(checkoutDeclaration, checkoutRepository, Writer{}); err == nil {
			t.Fatal("invalid apply reached target/write")
		}
	}
}
