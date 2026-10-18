package app

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func TestMergeCommitSelectiveReadDirectWriteAndLocalValidation(t *testing.T) {
	for _, input := range []string{"merge_commit_title: PR_TITLE", "merge_commit_title: PR_TITLE, merge_commit_message: BLANK"} {
		appCheckout(t, "schemaVersion: 1\nsettings: {repository: {"+input+"}}", true)
		calls := 0
		got, err := Plan(Reader{MergeMethods: func(repository.Repository) (settings.MergeMethodsState, error) {
			calls++
			return settings.MergeMethodsState{MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE"}, nil
		}})
		if err != nil || calls != 1 || got.Comparison.Managed < 1 {
			t.Fatalf("plan=%+v calls=%d err=%v", got, calls, err)
		}
		result, err := Apply(Writer{Repository: func(_ repository.Repository, d settings.RepositoryDeclaration) error {
			calls++
			if d.MergeCommitTitle == nil || *d.MergeCommitTitle != "PR_TITLE" || !d.BaseEmpty() {
				t.Fatalf("desired=%+v", d)
			}
			return nil
		}})
		if err != nil || !result.Applied || calls != 2 {
			t.Fatalf("apply=%+v err=%v", result, err)
		}
	}
	for _, input := range []string{"merge_commit_message: BLANK", "merge_commit_title: null", "merge_commit_title: ''"} {
		appCheckout(t, "schemaVersion: 1\nsettings: {repository: {"+input+"}}", false)
		if _, err := Plan(Reader{}); err == nil {
			t.Fatal("invalid plan reached target/read")
		}
		if _, err := Apply(Writer{}); err == nil {
			t.Fatal("invalid apply reached target/write")
		}
	}
}
