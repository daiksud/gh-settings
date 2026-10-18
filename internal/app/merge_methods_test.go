package app

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestPlanMergeMethodsSelectsOnlyNeededReads(t *testing.T) {
	input := "schemaVersion: 1\nsettings: {repository: {allow_merge_commit: false, allow_squash_merge: true}}\n"
	path := appCheckout(t, input, true)
	calls := 0
	result, err := Plan(Reader{MergeMethods: func(repo repository.Repository) (settings.MergeMethodsState, error) {
		calls++
		if repo.Owner != "owner" || repo.Name != "current" {
			t.Fatalf("target=%+v", repo)
		}
		return settings.MergeMethodsState{AllowMergeCommit: true}, nil
	}})
	want := []settings.Change{{Field: "repository.allow_merge_commit", Before: true, After: false}, {Field: "repository.allow_squash_merge", Before: false, After: true}}
	data, _ := os.ReadFile(path)
	if err != nil || calls != 1 || result.Comparison.Managed != 2 || !reflect.DeepEqual(result.Comparison.Changes, want) || string(data) != input {
		t.Fatalf("result=%+v calls=%d err=%v", result, calls, err)
	}
}

func TestPlanMixedRepositoryMergeReadFailureDiscardsComparison(t *testing.T) {
	appCheckout(t, "schemaVersion: 1\nsettings: {repository: {has_issues: false, allow_merge_commit: false}}\n", true)
	failure := errors.New("GraphQL partial response denied")
	var calls []string
	result, err := Plan(Reader{
		Repository: func(repository.Repository) (settings.RepositoryState, error) {
			calls = append(calls, "rest")
			return settings.RepositoryState{HasIssues: true}, nil
		},
		MergeMethods: func(repository.Repository) (settings.MergeMethodsState, error) {
			calls = append(calls, "graphql")
			return settings.MergeMethodsState{AllowMergeCommit: true}, failure
		},
	})
	var stage *PlanError
	if !errors.As(err, &stage) || stage.Stage != "read-failure" || !errors.Is(err, failure) || result.Comparison.Managed != 0 || len(result.Comparison.Changes) != 0 || !reflect.DeepEqual(calls, []string{"rest", "graphql"}) {
		t.Fatalf("result=%+v calls=%v error=%v", result, calls, err)
	}
}

func TestMergeMethodsApplyDirectAndInvalidLocal(t *testing.T) {
	appCheckout(t, "schemaVersion: 1\nsettings: {repository: {allow_rebase_merge: false}}\n", true)
	calls := 0
	result, err := Apply(Writer{Repository: func(repo repository.Repository, d settings.RepositoryDeclaration) error {
		calls++
		if repo.Owner != "owner" || d.AllowRebaseMerge == nil || *d.AllowRebaseMerge || !d.BaseEmpty() {
			t.Fatalf("target=%+v desired=%+v", repo, d)
		}
		return nil
	}})
	if err != nil || !result.Applied || calls != 1 {
		t.Fatalf("result=%+v calls=%d err=%v", result, calls, err)
	}
	for _, input := range []string{"null", "'false'", "[]"} {
		appCheckout(t, "schemaVersion: 1\nsettings: {repository: {allow_merge_commit: "+input+"}}\n", false)
		if _, err := Plan(Reader{}); err == nil {
			t.Fatal("invalid plan reached remote")
		}
		if _, err := Apply(Writer{}); err == nil {
			t.Fatal("invalid apply reached remote")
		}
	}
}

func TestMergeMethodsExportReadFailurePreservesFile(t *testing.T) {
	path := appCheckout(t, "previous configuration\n", true)
	failure := errors.New("GraphQL permission denied")
	err := Export(Reader{Repository: func(repository.Repository) (settings.RepositoryState, error) { return settings.RepositoryState{}, nil }, MergeMethods: func(repository.Repository) (settings.MergeMethodsState, error) {
		return settings.MergeMethodsState{}, failure
	}})
	data, readErr := os.ReadFile(path)
	if !errors.Is(err, failure) || readErr != nil || string(data) != "previous configuration\n" {
		t.Fatalf("error=%v file=%q read=%v", err, data, readErr)
	}
}

func TestDeleteBranchOnMergeUsesDeferredReadAndDirectWrite(t *testing.T) {
	appCheckout(t, "schemaVersion: 1\nsettings: {repository: {delete_branch_on_merge: false}}\n", true)
	calls := 0
	result, err := Plan(Reader{MergeMethods: func(repo repository.Repository) (settings.MergeMethodsState, error) {
		calls++
		return settings.MergeMethodsState{DeleteBranchOnMerge: true}, nil
	}})
	if err != nil || calls != 1 || result.Comparison.Managed != 1 || len(result.Comparison.Changes) != 1 || result.Comparison.Changes[0].After != false {
		t.Fatalf("plan=%+v err=%v calls=%d", result, err, calls)
	}
	applied, err := Apply(Writer{Repository: func(repo repository.Repository, d settings.RepositoryDeclaration) error {
		calls++
		if d.DeleteBranchOnMerge == nil || *d.DeleteBranchOnMerge || !d.BaseEmpty() {
			t.Fatalf("desired=%+v", d)
		}
		return nil
	}})
	if err != nil || !applied.Applied || calls != 2 {
		t.Fatalf("apply=%+v err=%v calls=%d", applied, err, calls)
	}
}
