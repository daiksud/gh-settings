package app

import (
	"errors"
	"reflect"
	"strings"
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
		{"web sign-off required", settings.RepositoryState{WebCommitSignoffRequired: true}},
		{"template repository", settings.RepositoryState{IsTemplate: true}},
		{"selected existing branch", settings.RepositoryState{DefaultBranch: "release/v2"}},
		{"projects enabled", settings.RepositoryState{Description: "Projects", HasProjects: true, HasWiki: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.state.DefaultBranch == "" {
				test.state.DefaultBranch = "main"
			}
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
			err := Export(checkout, exportFixtureReader(t, test.state, func(name string, repo repository.Repository) error {
				calls = append(calls, name)
				if repo != target {
					t.Fatalf("read target=%+v, want %+v", repo, target)
				}
				return nil
			}))
			no, yes := false, true
			policy := "ALL"
			squashTitle, squashMessage, mergeTitle, mergeMessage := "COMMIT_OR_PR_TITLE", "COMMIT_MESSAGES", "MERGE_MESSAGE", "PR_TITLE"
			names := []string{"a", "b"}
			color, description := "abcdef", "existing label"
			labels := appLabelDeclarations(t, settings.LabelDeclaration{Name: "bug", Color: &color, Description: &description})
			want := settings.Declaration{
				Repository: settings.RepositoryDeclaration{
					HasPullRequests: &no, PullRequestCreationPolicy: &policy,
					Description: &test.state.Description, Homepage: &test.state.Homepage, DefaultBranch: &test.state.DefaultBranch,
					HasIssues: &test.state.HasIssues, HasProjects: &test.state.HasProjects, HasWiki: &test.state.HasWiki, HasDiscussions: &test.state.HasDiscussions, WebCommitSignoffRequired: &test.state.WebCommitSignoffRequired, IsTemplate: &test.state.IsTemplate,
					AllowMergeCommit: &yes, AllowSquashMerge: &no, AllowRebaseMerge: &yes, DeleteBranchOnMerge: &no, AllowAutoMerge: &no, AllowUpdateBranch: &yes,
					SquashMergeCommitTitle: &squashTitle, SquashMergeCommitMessage: &squashMessage, MergeCommitTitle: &mergeTitle, MergeCommitMessage: &mergeMessage,
				}, Topics: settings.TopicsDeclaration{Names: &names}, Labels: &labels,
			}
			wantCalls := []string{"target", "root", "repository", "topics", "labels", "save"}
			if err != nil || !reflect.DeepEqual(saved, want) || !reflect.DeepEqual(calls, wantCalls) {
				t.Fatalf("saved=%+v error=%v calls=%v, want %+v after %v", saved, err, calls, want, wantCalls)
			}
		})
	}
}

func TestExportReadFailure(t *testing.T) {
	for _, stage := range []string{"repository", "topics", "labels"} {
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
			err := Export(checkout, exportFixtureReader(t, settings.RepositoryState{}, func(name string, _ repository.Repository) error {
				calls = append(calls, name)
				if name == stage {
					return cause
				}
				return nil
			}))
			want := []string{"target", "root", "repository", "topics", "labels"}
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
	}, exportFixtureReader(t, settings.RepositoryState{}, func(name string, _ repository.Repository) error { calls = append(calls, name); return nil }))
	want := []string{"target", "root", "repository", "topics", "labels", "save"}
	if err != cause || !reflect.DeepEqual(calls, want) {
		t.Fatalf("error=%v calls=%v, want original save error after %v", err, calls, want)
	}
}

func exportFixtureReader(t *testing.T, state settings.RepositoryState, step func(string, repository.Repository) error) Reader {
	t.Helper()
	return Reader{
		ObserveRepository: func(repo repository.Repository, fields []settings.FieldID) (settings.RepositoryObservation, error) {
			var paths []string
			for _, field := range fields {
				paths = append(paths, field.Path())
			}
			want := []string{"repository.description", "repository.homepage", "repository.default_branch", "repository.has_issues", "repository.has_projects", "repository.has_wiki", "repository.has_discussions", "repository.web_commit_signoff_required", "repository.is_template", "repository.allow_merge_commit", "repository.allow_squash_merge", "repository.allow_rebase_merge", "repository.delete_branch_on_merge", "repository.allow_auto_merge", "repository.allow_update_branch", "repository.squash_merge_commit_title", "repository.squash_merge_commit_message", "repository.merge_commit_title", "repository.merge_commit_message", "repository.has_pull_requests", "repository.pull_request_creation_policy"}
			if !reflect.DeepEqual(paths, want) {
				t.Fatalf("export fields=%v, want %v", paths, want)
			}
			methods := settings.MergeMethodsState{AllowMergeCommit: true, AllowRebaseMerge: true, AllowUpdateBranch: true, SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES", MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE"}
			return settings.NewRepositoryObservation(&state, &methods, &settings.PRAccessState{PullRequestCreationPolicy: "ALL"}), step("repository", repo)
		},
		Topics: func(repo repository.Repository) ([]string, error) { return []string{"b", "a"}, step("topics", repo) },
		Labels: func(repo repository.Repository) (settings.LabelObservation, error) {
			if err := step("labels", repo); err != nil {
				return settings.LabelObservation{}, err
			}
			return settings.NewLabelObservation([]settings.LabelState{{Name: "bug", Color: "abcdef", Description: "existing label"}})
		},
	}
}

func TestExportMissingRepositoryObservationPreventsSave(t *testing.T) {
	for _, test := range []struct {
		name     string
		observed settings.RepositoryObservation
		missing  string
	}{
		{"missing base", settings.NewRepositoryObservation(nil, &settings.MergeMethodsState{}, nil), "repository.description"},
		{"missing merge", settings.NewRepositoryObservation(&settings.RepositoryState{}, nil, nil), "repository.allow_merge_commit"},
		{"missing PR access", settings.NewRepositoryObservation(&settings.RepositoryState{}, &settings.MergeMethodsState{}, nil), "repository.has_pull_requests"},
	} {
		t.Run(test.name, func(t *testing.T) {
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
			err := Export(checkout, Reader{
				ObserveRepository: func(repository.Repository, []settings.FieldID) (settings.RepositoryObservation, error) {
					calls = append(calls, "repository")
					return test.observed, nil
				},
				Topics: func(repository.Repository) ([]string, error) { calls = append(calls, "topics"); return nil, nil },
				Labels: func(repository.Repository) (settings.LabelObservation, error) {
					calls = append(calls, "labels")
					return settings.LabelObservation{}, nil
				},
			})
			if err == nil || !strings.Contains(err.Error(), test.missing) || !reflect.DeepEqual(calls, []string{"target", "root", "repository"}) {
				t.Fatalf("error=%v calls=%v", err, calls)
			}
		})
	}
}
