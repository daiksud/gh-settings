package app

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"strings"
	"testing"
)

func TestTopicsApplyOrderAndPartialFailures(t *testing.T) {
	for _, fail := range []string{"", "repository", "topics"} {
		t.Run(fail, func(t *testing.T) {
			disabled, names := false, []string{}
			target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			var calls []string
			failure := errors.New("write failure")
			result, err := Apply(func() (settings.Declaration, error) {
				calls = append(calls, "load")
				return settings.Declaration{Repository: settings.RepositoryDeclaration{HasIssues: &disabled}, Topics: settings.TopicsDeclaration{Names: &names}}, nil
			}, func() (repository.Repository, error) {
				calls = append(calls, "target")
				return target, nil
			}, Writer{
				Repository: func(repo repository.Repository, d settings.RepositoryDeclaration) error {
					calls = append(calls, "repository")
					if repo != target || d.HasIssues == nil || *d.HasIssues {
						t.Fatal("lost false")
					}
					if fail == "repository" {
						return failure
					}
					return nil
				},
				Topics: func(repo repository.Repository, d settings.TopicsDeclaration) error {
					calls = append(calls, "topics")
					if repo != target || d.Names == nil || len(*d.Names) != 0 {
						t.Fatal("lost clear")
					}
					if fail == "topics" {
						return failure
					}
					return nil
				},
			})
			want := []string{"load", "target", "repository", "topics"}
			if fail == "repository" {
				want = want[:3]
			}
			if !reflect.DeepEqual(calls, want) || (fail != "" && !errors.Is(err, failure)) || (fail == "" && (err != nil || !result.Completed())) || (fail != "" && result.Completed()) {
				t.Fatalf("calls=%v result=%+v err=%v", calls, result, err)
			}
		})
	}
}
func TestTopicsOnlyPlanAndApply(t *testing.T) {
	for _, names := range []string{"[]", "[b, a]"} {
		t.Run(names, func(t *testing.T) {
			appCheckout(t, "version: daiksud/gh-settings@v0\nrepository: {about: {topics: "+names+"}}", true)
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
			applied, err := Apply(checkoutDeclaration, checkoutRepository, Writer{Topics: func(repository.Repository, settings.TopicsDeclaration) error { calls++; return nil }})
			if err != nil || !applied.Completed() || calls != 1 {
				t.Fatalf("result=%+v calls=%d error=%v", applied, calls, err)
			}
		})
	}
}
func TestEmptyTopicsIsLocalNoop(t *testing.T) {
	appCheckout(t, "version: daiksud/gh-settings@v0\nrepository: {about: {}}", false)
	if _, err := Plan(checkoutDeclaration, checkoutRepository, Reader{}); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(checkoutDeclaration, checkoutRepository, Writer{}); err != nil {
		t.Fatal(err)
	}
}
func TestInvalidTopicsBeforeRemote(t *testing.T) {
	appCheckout(t, "version: daiksud/gh-settings@v0\nrepository: {about: {topics: [null]}}", false)
	if _, err := Plan(checkoutDeclaration, checkoutRepository, Reader{}); err == nil || !strings.Contains(err.Error(), "repository.about.topics must contain strings") {
		t.Fatalf("error=%v", err)
	}
	if _, err := Apply(checkoutDeclaration, checkoutRepository, Writer{}); err == nil || !strings.Contains(err.Error(), "repository.about.topics must contain strings") {
		t.Fatalf("error=%v", err)
	}
}
