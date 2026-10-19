package app

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"testing"
)

func TestPROptionsDeferredReadAndDirectWrite(t *testing.T) {
	for _, field := range []string{"allow_auto_merge", "allow_update_branch"} {
		appCheckout(t, "schemaVersion: 1\nsettings: {repository: {"+field+": false}}", true)
		calls := 0
		got, err := Plan(checkoutDeclaration, checkoutRepository, Reader{MergeMethods: func(repository.Repository) (settings.MergeMethodsState, error) {
			calls++
			return settings.MergeMethodsState{AllowAutoMerge: true, AllowUpdateBranch: true}, nil
		}})
		if err != nil || calls != 1 || got.Comparison.Managed != 1 || len(got.Comparison.Changes) != 1 || got.Comparison.Changes[0].Field().Path() != "repository."+field || comparisonAfterBool(t, got.Comparison.Changes[0]) != false {
			t.Fatalf("plan=%+v calls=%d err=%v", got, calls, err)
		}
		result, err := Apply(checkoutDeclaration, Writer{Repository: func(_ repository.Repository, d settings.RepositoryDeclaration) error {
			calls++
			v := d.AllowAutoMerge
			if field == "allow_update_branch" {
				v = d.AllowUpdateBranch
			}
			if v == nil || *v || !d.BaseEmpty() {
				t.Fatalf("desired=%+v", d)
			}
			return nil
		}})
		if err != nil || !result.Applied || calls != 2 {
			t.Fatalf("apply=%+v err=%v", result, err)
		}
		appCheckout(t, "schemaVersion: 1\nsettings: {repository: {"+field+": null}}", false)
		if _, err := Plan(checkoutDeclaration, checkoutRepository, Reader{}); err == nil {
			t.Fatal("invalid plan reached read")
		}
		if _, err := Apply(checkoutDeclaration, Writer{}); err == nil {
			t.Fatal("invalid apply reached write")
		}
	}
}
