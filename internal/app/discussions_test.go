package app

import (
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"reflect"
	"testing"
)

func TestPlanRequestsDiscussionsField(t *testing.T) {
	disabled := false
	calls := 0
	got, err := Plan(func() (settings.Declaration, error) {
		return settings.Declaration{Repository: settings.RepositoryDeclaration{HasDiscussions: &disabled}}, nil
	},
		func() (repository.Repository, error) {
			return repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}, nil
		},
		Reader{ObserveRepository: func(repo repository.Repository, fields []settings.FieldID) (settings.RepositoryObservation, error) {
			calls++
			var paths []string
			for _, field := range fields {
				paths = append(paths, field.Path())
			}
			if repo.Owner != "owner" || repo.Name != "current" || !reflect.DeepEqual(paths, []string{"repository.has_discussions"}) {
				t.Fatalf("target=%+v fields=%v", repo, paths)
			}
			return settings.NewRepositoryObservation(&settings.RepositoryState{HasDiscussions: true}, nil), nil
		}})
	if err != nil || calls != 1 || got.Comparison.Managed != 1 || len(got.Comparison.Changes) != 1 || got.Comparison.Changes[0].Field().Path() != "repository.has_discussions" || comparisonAfterBool(t, got.Comparison.Changes[0]) != false {
		t.Fatalf("plan=%+v calls=%d error=%v", got, calls, err)
	}
}
