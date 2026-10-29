package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPlanRequestsMergeMethodFields(t *testing.T) {
	disabled, enabled := false, true
	target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	calls := 0
	result, err := Plan(func() (settings.Declaration, error) {
		return settings.Declaration{Repository: settings.RepositoryDeclaration{AllowMergeCommit: &disabled, AllowSquashMerge: &enabled}}, nil
	}, func() (repository.Repository, error) { return target, nil }, Reader{ObserveRepository: func(repo repository.Repository, fields []settings.FieldID) (settings.RepositoryObservation, error) {
		calls++
		var paths []string
		for _, field := range fields {
			paths = append(paths, field.Path())
		}
		if repo != target || !reflect.DeepEqual(paths, []string{"repository.allow_merge_commit", "repository.allow_squash_merge"}) {
			t.Fatalf("target=%+v fields=%v", repo, paths)
		}
		return settings.NewRepositoryObservation(nil, &settings.MergeMethodsState{AllowMergeCommit: true}, nil), nil
	}})
	want := []settings.Change{comparisonChange(t, "repository.allow_merge_commit", true, false), comparisonChange(t, "repository.allow_squash_merge", false, true)}
	if err != nil || calls != 1 || result.Comparison.Managed != 2 || !reflect.DeepEqual(result.Comparison.Changes, want) {
		t.Fatalf("result=%+v calls=%d error=%v", result, calls, err)
	}
}

func TestPlanMixedRepositoryObservationFailureDiscardsComparison(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(map[bool]string{false: "later read failure", true: "missing observation"}[missing], func(t *testing.T) {
			disabled, names := false, []string{}
			target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			cause := errors.New("GraphQL partial response denied")
			calls := 0
			result, err := Plan(func() (settings.Declaration, error) {
				return settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &disabled, AllowMergeCommit: &disabled}, Topics: settings.TopicsDeclaration{Names: &names}}, nil
			}, func() (repository.Repository, error) { return target, nil }, Reader{ObserveRepository: func(repo repository.Repository, fields []settings.FieldID) (settings.RepositoryObservation, error) {
				calls++
				var paths []string
				for _, field := range fields {
					paths = append(paths, field.Path())
				}
				if repo != target || !reflect.DeepEqual(paths, []string{"repository.has_issues", "repository.allow_merge_commit"}) {
					t.Fatalf("target=%+v fields=%v", repo, paths)
				}
				observed := settings.NewRepositoryObservation(&settings.RepositoryState{HasIssues: true}, nil, nil)
				if missing {
					return observed, nil
				}
				return observed, cause
			}})
			var stage *PlanError
			if !errors.As(err, &stage) || stage.Stage != "read-failure" || stage.Target != target || (!missing && !errors.Is(err, cause)) || !reflect.DeepEqual(result, PlanResult{}) || calls != 1 {
				t.Fatalf("result=%+v calls=%d error=%v", result, calls, err)
			}
		})
	}
}

func TestMergeMethodsApplyDirectAndInvalidLocal(t *testing.T) {
	appCheckout(t, "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {pull_requests: {allow_rebase_merging: false}}}}\n", true)
	calls := 0
	result, err := Apply(checkoutDeclaration, checkoutRepository, Writer{Repository: func(repo repository.Repository, d settings.RepositoryDeclaration) error {
		calls++
		if repo.Owner != "owner" || d.AllowRebaseMerge == nil || *d.AllowRebaseMerge || !d.BaseEmpty() {
			t.Fatalf("target=%+v desired=%+v", repo, d)
		}
		return nil
	}})
	if err != nil || !result.Completed() || calls != 1 {
		t.Fatalf("result=%+v calls=%d err=%v", result, calls, err)
	}
	for _, input := range []string{"null", "'false'", "[]"} {
		appCheckout(t, "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {pull_requests: {allow_merge_commits: "+input+"}}}}\n", false)
		if _, err := Plan(checkoutDeclaration, checkoutRepository, Reader{}); err == nil {
			t.Fatal("invalid plan reached remote")
		}
		if _, err := Apply(checkoutDeclaration, checkoutRepository, Writer{}); err == nil {
			t.Fatal("invalid apply reached remote")
		}
	}
}

func TestDeleteBranchOnMergeUsesDeferredReadAndDirectWrite(t *testing.T) {
	appCheckout(t, "version: daiksud/gh-settings@v0\nrepository: {settings: {general: {pull_requests: {automatically_delete_head_branches: false}}}}\n", true)
	calls := 0
	result, err := Plan(checkoutDeclaration, checkoutRepository, Reader{ObserveRepository: observeMergeFixture(func(repo repository.Repository) (settings.MergeMethodsState, error) {
		calls++
		return settings.MergeMethodsState{DeleteBranchOnMerge: true}, nil
	})})
	if err != nil || calls != 1 || result.Comparison.Managed != 1 || len(result.Comparison.Changes) != 1 || comparisonAfterBool(t, result.Comparison.Changes[0]) != false {
		t.Fatalf("plan=%+v err=%v calls=%d", result, err, calls)
	}
	applied, err := Apply(checkoutDeclaration, checkoutRepository, Writer{Repository: func(repo repository.Repository, d settings.RepositoryDeclaration) error {
		calls++
		if d.DeleteBranchOnMerge == nil || *d.DeleteBranchOnMerge || !d.BaseEmpty() {
			t.Fatalf("desired=%+v", d)
		}
		return nil
	}})
	if err != nil || !applied.Completed() || calls != 2 {
		t.Fatalf("apply=%+v err=%v calls=%d", applied, err, calls)
	}
}
