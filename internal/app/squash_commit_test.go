package app

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func TestSquashCommitSelectiveReadDirectWriteAndLocalValidation(t *testing.T) {
	for _, input := range []string{"pull_request_title", "pull_request_title_and_description"} {
		appCheckout(t, "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {pull_requests: {squash_merge: {default_commit_message: "+input+"}}}}}", true)
		calls := 0
		got, err := Plan(checkoutDeclaration, checkoutRepository, Reader{ObserveRepository: observeMergeFixture(func(repository.Repository) (settings.MergeMethodsState, error) {
			calls++
			return settings.MergeMethodsState{SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES", MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE"}, nil
		})})
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
		if err != nil || !result.Completed() || calls != 2 {
			t.Fatalf("apply=%+v err=%v", result, err)
		}
	}
	for _, input := range []string{"squash_merge_commit_message: BLANK", "squash_merge: {default_commit_message: null}", "squash_merge: {default_commit_message: ''}"} {
		appCheckout(t, "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {pull_requests: {"+input+"}}}}", false)
		if _, err := Plan(checkoutDeclaration, checkoutRepository, Reader{}); err == nil {
			t.Fatal("invalid plan reached target/read")
		}
		if _, err := Apply(checkoutDeclaration, checkoutRepository, Writer{}); err == nil {
			t.Fatal("invalid apply reached target/write")
		}
	}
}
