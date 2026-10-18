package app

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestLabelsPlanAndApplyEmptyListDeletesAll(t *testing.T) {
	appCheckout(t, "schemaVersion: 1\nsettings: {labels: []}\n", true)
	target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
	state := []settings.LabelState{{Name: "assigned", Color: "abcdef", Description: "old"}}
	reads := 0
	read := func(repo repository.Repository) ([]settings.LabelState, error) {
		if repo != target {
			t.Fatalf("target=%+v", repo)
		}
		reads++
		return state, nil
	}
	plan, err := Plan(Reader{Labels: read})
	if err != nil || plan.Comparison.Managed != 1 || len(plan.Comparison.Labels) != 1 || plan.Comparison.Labels[0].Operation != "delete" {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	calls := 0
	applied, err := Apply(Writer{ReadLabels: read, Labels: func(repo repository.Repository, changes []settings.LabelChange) error {
		calls++
		if repo != target || !reflect.DeepEqual(changes, plan.Comparison.Labels) {
			t.Fatalf("changes=%+v", changes)
		}
		return nil
	}})
	if err != nil || !applied.Applied || calls != 1 || reads != 2 {
		t.Fatalf("result=%+v reads=%d writes=%d err=%v", applied, reads, calls, err)
	}
}

func TestLabelsApplyReadsFreshStateAndSkipsUnchangedWrites(t *testing.T) {
	input := "schemaVersion: 1\nsettings: {labels: [{name: bug}]}\n"
	path := appCheckout(t, input, true)
	result, err := Apply(Writer{ReadLabels: func(repository.Repository) ([]settings.LabelState, error) {
		return []settings.LabelState{{Name: "BUG", Color: "abcdef", Description: "external edit"}}, nil
	}})
	if err != nil || !result.Applied {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != input {
		t.Fatal("configuration changed")
	}
}

func TestInvalidLabelsBeforeTargetAndRemoteAccess(t *testing.T) {
	for _, input := range []string{"null", "[{name: bug}, {name: BUG}]", "[{name: bug, color: false}]"} {
		t.Run(input, func(t *testing.T) {
			appCheckout(t, "schemaVersion: 1\nsettings: {labels: "+input+"}", false)
			if _, err := Plan(Reader{}); err == nil {
				t.Fatal("plan accepted invalid labels")
			}
			if _, err := Apply(Writer{}); err == nil {
				t.Fatal("apply accepted invalid labels")
			}
		})
	}
	appCheckout(t, "schemaVersion: 1\nsettings: {labels: []}", false)
	if _, err := Apply(Writer{}); err == nil || !strings.Contains(err.Error(), "origin") {
		t.Fatalf("empty list lost managed presence: %v", err)
	}
}

func TestLabelsMixedApplyFailureOrder(t *testing.T) {
	for _, failureStage := range []string{"", "repository", "topics", "labels-read", "labels-write"} {
		t.Run(failureStage, func(t *testing.T) {
			appCheckout(t, "schemaVersion: 1\nsettings:\n  repository: {has_issues: false}\n  topics: {names: []}\n  labels: []\n", true)
			failure := errors.New("fixture failure")
			var calls []string
			step := func(name string) error {
				calls = append(calls, name)
				if failureStage == name {
					return failure
				}
				return nil
			}
			result, err := Apply(Writer{
				Repository: func(repository.Repository, settings.RepositoryDeclaration) error { return step("repository") },
				Topics:     func(repository.Repository, settings.TopicsDeclaration) error { return step("topics") },
				ReadLabels: func(repository.Repository) ([]settings.LabelState, error) {
					err := step("labels-read")
					return []settings.LabelState{{Name: "old", Color: "abcdef"}}, err
				},
				Labels: func(repository.Repository, []settings.LabelChange) error { return step("labels-write") },
			})
			want := []string{"repository", "topics", "labels-read", "labels-write"}
			if failureStage != "" {
				for i, name := range want {
					if name == failureStage {
						want = want[:i+1]
						break
					}
				}
			}
			if !reflect.DeepEqual(calls, want) || (failureStage != "" && !errors.Is(err, failure)) || (failureStage == "" && (err != nil || !result.Applied)) || (failureStage != "" && result.Applied) {
				t.Fatalf("calls=%v result=%+v err=%v", calls, result, err)
			}
		})
	}
}

func TestLabelsReadFailurePreservesExport(t *testing.T) {
	before := "original configuration"
	path := appCheckout(t, before, true)
	failure := errors.New("later labels page unavailable")
	err := Export(Reader{MergeMethods: func(repository.Repository) (settings.MergeMethodsState, error) {
		return settings.MergeMethodsState{SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES", MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE"}, nil
	}, Repository: func(repository.Repository) (settings.RepositoryState, error) { return settings.RepositoryState{}, nil }, Topics: func(repository.Repository) ([]string, error) { return []string{}, nil }, Labels: func(repository.Repository) ([]settings.LabelState, error) { return nil, failure }})
	data, readErr := os.ReadFile(path)
	if !errors.Is(err, failure) || readErr != nil || string(data) != before {
		t.Fatalf("error=%v data=%q read=%v", err, data, readErr)
	}
}

func TestLabelsPlanFailureDoesNotExposeEarlierComparison(t *testing.T) {
	appCheckout(t, "schemaVersion: 1\nsettings:\n  repository: {has_issues: false}\n  labels: []\n", true)
	for _, ambiguous := range []bool{false, true} {
		result, err := Plan(Reader{
			Repository: func(repository.Repository) (settings.RepositoryState, error) {
				return settings.RepositoryState{HasIssues: true}, nil
			},
			Labels: func(repository.Repository) ([]settings.LabelState, error) {
				if !ambiguous {
					return nil, errors.New("later page failed")
				}
				return []settings.LabelState{{Name: "bug", Color: "abcdef"}, {Name: "BUG", Color: "abcdef"}}, nil
			},
		})
		var failure *PlanError
		if !errors.As(err, &failure) || failure.Stage != "read-failure" || failure.Target.Owner != "owner" || result.Comparison.Managed != 0 || len(result.Comparison.Changes) != 0 || len(result.Comparison.Labels) != 0 {
			t.Fatalf("result=%+v error=%v", result, err)
		}
	}
}
