package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestTopicsApplyOrderAndPartialFailures(t *testing.T) {
	for _, fail := range []string{"", "repository", "topics"} {
		t.Run(fail, func(t *testing.T) {
			input := "schemaVersion: 1\nsettings:\n  repository: {has_issues: false}\n  topics: {names: []}\n"
			path := appCheckout(t, input, true)
			var calls []string
			failure := errors.New("write failure")
			result, err := Apply(checkoutDeclaration, Writer{
				Repository: func(repo repository.Repository, d settings.RepositoryDeclaration) error {
					calls = append(calls, "repository")
					if d.HasIssues == nil || *d.HasIssues {
						t.Fatal("lost false")
					}
					if fail == "repository" {
						return failure
					}
					return nil
				},
				Topics: func(repo repository.Repository, d settings.TopicsDeclaration) error {
					calls = append(calls, "topics")
					if d.Names == nil || len(*d.Names) != 0 {
						t.Fatal("lost clear")
					}
					if fail == "topics" {
						return failure
					}
					return nil
				},
			})
			want := []string{"repository", "topics"}
			if fail == "repository" {
				want = want[:1]
			}
			if !reflect.DeepEqual(calls, want) || (fail != "" && !errors.Is(err, failure)) || (fail == "" && (err != nil || !result.Applied)) || (fail != "" && result.Applied) {
				t.Fatalf("calls=%v result=%+v err=%v", calls, result, err)
			}
			data, _ := os.ReadFile(path)
			if string(data) != input {
				t.Fatal("configuration changed")
			}
		})
	}
}
func TestTopicsOnlyPlanAndApply(t *testing.T) {
	for _, names := range []string{"[]", "[b, a]"} {
		t.Run(names, func(t *testing.T) {
			appCheckout(t, "schemaVersion: 1\nsettings: {topics: {names: "+names+"}}", true)
			result, err := Plan(checkoutDeclaration, checkoutRepository, Reader{Topics: func(repository.Repository) ([]string, error) { return []string{"a", "b"}, nil }})
			if err != nil || result.Comparison.Managed != 1 {
				t.Fatalf("result=%+v error=%v", result, err)
			}
			changes := 0
			if names == "[]" {
				changes = 1
			}
			if len(result.Comparison.Changes) != changes {
				t.Fatalf("comparison=%+v", result.Comparison)
			}
			calls := 0
			applied, err := Apply(checkoutDeclaration, Writer{Topics: func(repository.Repository, settings.TopicsDeclaration) error { calls++; return nil }})
			if err != nil || !applied.Applied || calls != 1 {
				t.Fatalf("result=%+v calls=%d error=%v", applied, calls, err)
			}
		})
	}
}
func TestEmptyTopicsIsLocalNoop(t *testing.T) {
	appCheckout(t, "schemaVersion: 1\nsettings: {topics: {}}", false)
	if _, err := Plan(checkoutDeclaration, checkoutRepository, Reader{}); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(checkoutDeclaration, Writer{}); err != nil {
		t.Fatal(err)
	}
}
func TestInvalidTopicsBeforeRemote(t *testing.T) {
	appCheckout(t, "schemaVersion: 1\nsettings: {topics: {names: [null]}}", false)
	if _, err := Plan(checkoutDeclaration, checkoutRepository, Reader{}); err == nil || !strings.Contains(err.Error(), "array of strings") {
		t.Fatalf("error=%v", err)
	}
	if _, err := Apply(checkoutDeclaration, Writer{}); err == nil || !strings.Contains(err.Error(), "array of strings") {
		t.Fatalf("error=%v", err)
	}
}
func TestTopicsExportFailureBeforeOverwrite(t *testing.T) {
	for _, existing := range []bool{true, false} {
		t.Run(map[bool]string{true: "existing", false: "absent"}[existing], func(t *testing.T) {
			before := "previous configuration"
			path := appCheckout(t, before, true)
			if !existing {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			}
			failure := errors.New("topics forbidden")
			var calls []string
			err := Export(Reader{MergeMethods: func(repository.Repository) (settings.MergeMethodsState, error) {
				return settings.MergeMethodsState{SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES", MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE"}, nil
			}, Repository: func(repository.Repository) (settings.RepositoryState, error) {
				calls = append(calls, "repository")
				return settings.RepositoryState{}, nil
			}, Topics: func(repository.Repository) ([]string, error) { calls = append(calls, "topics"); return nil, failure }})
			if !errors.Is(err, failure) || !reflect.DeepEqual(calls, []string{"repository", "topics"}) {
				t.Fatalf("calls=%v error=%v", calls, err)
			}
			data, err := os.ReadFile(path)
			if existing && (err != nil || string(data) != before) || !existing && !os.IsNotExist(err) {
				t.Fatalf("destination=%q error=%v", data, err)
			}
		})
	}
}
