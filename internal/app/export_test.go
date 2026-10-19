package app

import (
	"errors"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestExportStoredSettings(t *testing.T) {
	for _, test := range []struct {
		name  string
		state settings.RepositoryState
	}{
		{"five values", settings.RepositoryState{Description: "Example repository", Homepage: "https://example.com", HasIssues: true, HasWiki: true}},
		{"zero values", settings.RepositoryState{}},
		{"discussions enabled", settings.RepositoryState{HasDiscussions: true}},
		{"projects enabled", settings.RepositoryState{Description: "Projects", HasProjects: true, HasWiki: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			var calls []string
			var saved settings.Declaration
			checkout := ExportCheckout{
				Repository: func() (repository.Repository, error) { calls = append(calls, "target"); return target, nil },
				PrepareSave: func() (func(settings.Declaration) error, error) {
					calls = append(calls, "root")
					return func(d settings.Declaration) error { calls = append(calls, "save"); saved = d; return nil }, nil
				},
			}
			err := Export(checkout, exportFixtureReader(test.state, func(name string, repo repository.Repository) error {
				calls = append(calls, name)
				if repo != target {
					t.Fatalf("read target=%+v, want %+v", repo, target)
				}
				return nil
			}))
			no, yes := false, true
			squashTitle, squashMessage, mergeTitle, mergeMessage := "COMMIT_OR_PR_TITLE", "COMMIT_MESSAGES", "MERGE_MESSAGE", "PR_TITLE"
			names := []string{"a", "b"}
			color, description := "abcdef", "existing label"
			labels := []settings.LabelDeclaration{{Name: "bug", Color: &color, Description: &description}}
			want := settings.Declaration{
				Repository: settings.RepositoryDeclaration{
					Description: &test.state.Description, Homepage: &test.state.Homepage,
					HasIssues: &test.state.HasIssues, HasProjects: &test.state.HasProjects, HasWiki: &test.state.HasWiki, HasDiscussions: &test.state.HasDiscussions,
					AllowMergeCommit: &yes, AllowSquashMerge: &no, AllowRebaseMerge: &yes, DeleteBranchOnMerge: &no, AllowAutoMerge: &no, AllowUpdateBranch: &yes,
					SquashMergeCommitTitle: &squashTitle, SquashMergeCommitMessage: &squashMessage, MergeCommitTitle: &mergeTitle, MergeCommitMessage: &mergeMessage,
				}, Topics: settings.TopicsDeclaration{Names: &names}, Labels: &labels,
			}
			wantCalls := []string{"target", "root", "repository", "merge-methods", "topics", "labels", "save"}
			if err != nil || !reflect.DeepEqual(saved, want) || !reflect.DeepEqual(calls, wantCalls) {
				t.Fatalf("saved=%+v error=%v calls=%v, want %+v after %v", saved, err, calls, want, wantCalls)
			}
		})
	}
}

func TestExportReadFailure(t *testing.T) {
	for _, stage := range []string{"repository", "merge-methods", "topics", "labels"} {
		t.Run(stage, func(t *testing.T) {
			cause := errors.New("complete read unavailable")
			var calls []string
			checkout := ExportCheckout{
				Repository: func() (repository.Repository, error) {
					calls = append(calls, "target")
					return repository.Repository{}, nil
				},
				PrepareSave: func() (func(settings.Declaration) error, error) {
					calls = append(calls, "root")
					return func(settings.Declaration) error { calls = append(calls, "save"); return nil }, nil
				},
			}
			err := Export(checkout, exportFixtureReader(settings.RepositoryState{}, func(name string, _ repository.Repository) error {
				calls = append(calls, name)
				if name == stage {
					return cause
				}
				return nil
			}))
			want := []string{"target", "root", "repository", "merge-methods", "topics", "labels"}
			for i, name := range want {
				if name == stage {
					want = want[:i+1]
					break
				}
			}
			if err != cause || !reflect.DeepEqual(calls, want) {
				t.Fatalf("error=%v calls=%v, want original read failure after %v", err, calls, want)
			}
		})
	}
}

func TestExportTargetFailureBeforeRead(t *testing.T) {
	cause := errors.New("resolve current repository: original origin failure")
	var calls []string
	err := Export(ExportCheckout{
		Repository: func() (repository.Repository, error) {
			calls = append(calls, "target")
			return repository.Repository{}, cause
		},
		PrepareSave: func() (func(settings.Declaration) error, error) { calls = append(calls, "root"); return nil, nil },
	}, Reader{})
	if err != cause || !reflect.DeepEqual(calls, []string{"target"}) {
		t.Fatalf("error=%v calls=%v", err, calls)
	}
}

func TestExportRootFailureBeforeRead(t *testing.T) {
	cause := errors.New("locate .github/settings.yml: original Git root failure")
	var calls []string
	err := Export(ExportCheckout{
		Repository: func() (repository.Repository, error) {
			calls = append(calls, "target")
			return repository.Repository{}, nil
		},
		PrepareSave: func() (func(settings.Declaration) error, error) { calls = append(calls, "root"); return nil, cause },
	}, Reader{})
	if err != cause || !reflect.DeepEqual(calls, []string{"target", "root"}) {
		t.Fatalf("error=%v calls=%v", err, calls)
	}
}

func TestExportStorageFailure(t *testing.T) {
	cause := errors.New(".github/settings.yml must be a regular file, not a symlink or other file")
	var calls []string
	err := Export(ExportCheckout{
		Repository: func() (repository.Repository, error) {
			calls = append(calls, "target")
			return repository.Repository{}, nil
		},
		PrepareSave: func() (func(settings.Declaration) error, error) {
			calls = append(calls, "root")
			return func(settings.Declaration) error { calls = append(calls, "save"); return cause }, nil
		},
	}, exportFixtureReader(settings.RepositoryState{}, func(name string, _ repository.Repository) error { calls = append(calls, name); return nil }))
	want := []string{"target", "root", "repository", "merge-methods", "topics", "labels", "save"}
	if err != cause || !reflect.DeepEqual(calls, want) {
		t.Fatalf("error=%v calls=%v, want original save error after %v", err, calls, want)
	}
}

func exportFixtureReader(state settings.RepositoryState, step func(string, repository.Repository) error) Reader {
	return Reader{
		Repository: func(repo repository.Repository) (settings.RepositoryState, error) {
			return state, step("repository", repo)
		},
		MergeMethods: func(repo repository.Repository) (settings.MergeMethodsState, error) {
			return settings.MergeMethodsState{AllowMergeCommit: true, AllowRebaseMerge: true, AllowUpdateBranch: true, SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES", MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE"}, step("merge-methods", repo)
		},
		Topics: func(repo repository.Repository) ([]string, error) { return []string{"b", "a"}, step("topics", repo) },
		Labels: func(repo repository.Repository) ([]settings.LabelState, error) {
			return []settings.LabelState{{Name: "bug", Color: "abcdef", Description: "existing label"}}, step("labels", repo)
		},
	}
}
