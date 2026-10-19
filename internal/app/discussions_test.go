package app

import (
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPlanDiscussionsOnlyReadsREST(t *testing.T) {
	appCheckout(t, "schemaVersion: 1\nsettings: {repository: {has_discussions: false}}\n", true)
	calls := 0
	got, err := Plan(checkoutDeclaration, Reader{Repository: func(repo repository.Repository) (settings.RepositoryState, error) {
		calls++
		if repo.Owner != "owner" || repo.Name != "current" {
			t.Fatalf("wrong target: %+v", repo)
		}
		return settings.RepositoryState{HasDiscussions: true}, nil
	}})
	if err != nil || calls != 1 || got.Comparison.Managed != 1 || len(got.Comparison.Changes) != 1 || got.Comparison.Changes[0].Field().Path() != "repository.has_discussions" || comparisonAfterBool(t, got.Comparison.Changes[0]) != false {
		t.Fatalf("plan=%+v calls=%d err=%v", got, calls, err)
	}
}
