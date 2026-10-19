package github

import (
	"errors"
	"github.com/cli/go-gh/v2/pkg/api"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestObserveRepositorySelectsReads(t *testing.T) {
	for _, test := range []struct {
		name        string
		paths, want []string
		acquired    int
	}{
		{"REST only", []string{"repository.has_issues"}, []string{"rest"}, 6},
		{"GraphQL only", []string{"repository.allow_merge_commit"}, []string{"graphql"}, 10},
		{"GraphQL string only", []string{"repository.merge_commit_title"}, []string{"graphql"}, 10},
		{"mixed preserves REST first", []string{"repository.allow_merge_commit", "repository.description"}, []string{"rest", "graphql"}, 16},
		{"no fields", nil, nil, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			var calls []string
			check := func(name string, repo repository.Repository) {
				calls = append(calls, name)
				if repo != target {
					t.Fatalf("target=%+v", repo)
				}
			}
			got, err := observeRepository(target, observationFields(t, test.paths...),
				func(repo repository.Repository) (settings.RepositoryState, error) {
					check("rest", repo)
					return settings.RepositoryState{}, nil
				},
				func(repo repository.Repository) (settings.MergeMethodsState, error) {
					check("graphql", repo)
					return settings.MergeMethodsState{}, nil
				},
			)
			if err != nil || !reflect.DeepEqual(calls, test.want) || len(got.Fields()) != test.acquired {
				t.Fatalf("fields=%v calls=%v error=%v, want %d fields and %v", got.Fields(), calls, err, test.acquired, test.want)
			}
		})
	}
}

func TestObserveRepositoryDiscardsPartialRead(t *testing.T) {
	for _, fail := range []string{"rest", "graphql"} {
		t.Run(fail, func(t *testing.T) {
			cause := errors.New("complete read denied")
			var calls []string
			step := func(name string) error {
				calls = append(calls, name)
				if name == fail {
					return cause
				}
				return nil
			}
			got, err := observeRepository(repository.Repository{}, observationFields(t, "repository.has_issues", "repository.allow_merge_commit"),
				func(repository.Repository) (settings.RepositoryState, error) {
					return settings.RepositoryState{HasIssues: true}, step("rest")
				},
				func(repository.Repository) (settings.MergeMethodsState, error) {
					return settings.MergeMethodsState{AllowMergeCommit: true}, step("graphql")
				},
			)
			want := []string{"rest"}
			if fail == "graphql" {
				want = append(want, "graphql")
			}
			if err != cause || len(got.Fields()) != 0 || !reflect.DeepEqual(calls, want) {
				t.Fatalf("fields=%v calls=%v error=%v", got.Fields(), calls, err)
			}
		})
	}
}

func TestObserveRepositoryRejectsInvalidRequestBeforeReads(t *testing.T) {
	for _, fields := range [][]settings.FieldID{{{}}, {settings.TopicsField()}} {
		got, err := observeRepository(repository.Repository{}, fields, nil, nil)
		if err == nil || len(got.Fields()) != 0 {
			t.Fatalf("observation=%+v error=%v", got, err)
		}
	}
}

func observationFields(t *testing.T, paths ...string) []settings.FieldID {
	t.Helper()
	var fields []settings.FieldID
	for _, path := range paths {
		field, err := settings.ParseFieldID(path)
		if err != nil {
			t.Fatal(err)
		}
		fields = append(fields, field)
	}
	return fields
}

func TestObserveRepositoryRetainsCompleteResponseValidation(t *testing.T) {
	for _, test := range []struct{ name, field, rest, graphql, want string }{
		{name: "undeclared REST field missing", field: "repository.has_issues", rest: `{"homepage":"","has_issues":false,"has_projects":false,"has_wiki":false,"has_discussions":false}`, want: "missing description"},
		{name: "undeclared GraphQL field missing", field: "repository.allow_merge_commit", graphql: `{"data":{"repository":{"mergeCommitAllowed":false,"squashMergeAllowed":false,"rebaseMergeAllowed":false,"deleteBranchOnMerge":false,"autoMergeAllowed":false,"allowUpdateBranch":false,"squashMergeCommitTitle":"PR_TITLE","squashMergeCommitMessage":"BLANK","mergeCommitTitle":"PR_TITLE"}}}`, want: "missing mergeCommitMessage"},
		{name: "GraphQL partial error", field: "repository.allow_merge_commit", graphql: `{"data":{"repository":{"mergeCommitAllowed":false}},"errors":[{"message":"partial response denied"}]}`, want: "partial response denied"},
	} {
		t.Run(test.name, func(t *testing.T) {
			rest := readTestClient(t, func(req *http.Request) (*http.Response, error) { return labelsResponse(req, 200, test.rest), nil })
			gql, err := api.NewGraphQLClient(api.ClientOptions{Host: "github.com", AuthToken: "fixture-token", LogIgnoreEnv: true, Transport: transportFunc(func(req *http.Request) (*http.Response, error) { return labelsResponse(req, 200, test.graphql), nil })})
			if err != nil {
				t.Fatal(err)
			}
			merge := &MergeMethodsClient{graphql: gql}
			got, err := observeRepository(repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}, observationFields(t, test.field), rest.ReadRepository, merge.ReadMergeMethods)
			if err == nil || !strings.Contains(err.Error(), test.want) || len(got.Fields()) != 0 {
				t.Fatalf("fields=%v error=%v, want %q", got.Fields(), err, test.want)
			}
		})
	}
}
