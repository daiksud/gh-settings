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
			issuePolicy := "ALL"
			mergeTitle, mergeMessage := "MERGE_MESSAGE", "PR_TITLE"
			names := []string{"a", "b"}
			id, color, description := int64(1), "abcdef", "existing label"
			labels := appLabelDeclarations(t, settings.LabelDeclaration{ID: &id, Name: "bug", Color: &color, Description: &description})
			want := settings.Declaration{
				Repository: settings.RepositoryDeclaration{
					HasPullRequests: &no, IssueCreationPolicy: &issuePolicy, HasSponsorships: &no,
					Description: &test.state.Description, Homepage: &test.state.Homepage, DefaultBranch: &test.state.DefaultBranch,
					HasIssues: &test.state.HasIssues, HasProjects: &test.state.HasProjects, HasWiki: &test.state.HasWiki, HasDiscussions: &test.state.HasDiscussions, WebCommitSignoffRequired: &test.state.WebCommitSignoffRequired, IsTemplate: &test.state.IsTemplate,
					AllowMergeCommit: &yes, AllowSquashMerge: &no, AllowRebaseMerge: &yes, DeleteBranchOnMerge: &no, AllowAutoMerge: &no, AllowUpdateBranch: &yes,
					MergeCommitTitle: &mergeTitle, MergeCommitMessage: &mergeMessage,
				}, Topics: settings.TopicsDeclaration{Names: &names}, Labels: &labels,
			}
			if !test.state.HasIssues {
				want.Repository.IssueCreationPolicy = nil
			}
			wantCalls := []string{"target", "root", "repository", "merge", "pr", "issue", "sponsorships", "topics", "labels", "save"}
			if err != nil || !reflect.DeepEqual(saved, want) || !reflect.DeepEqual(calls, wantCalls) {
				t.Fatalf("saved=%+v error=%v calls=%v, want %+v after %v", saved, err, calls, want, wantCalls)
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
	want := []string{"target", "root", "repository", "merge", "pr", "issue", "sponsorships", "topics", "labels", "save"}
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
			groups := []struct {
				name  string
				paths []string
			}{
				{"repository", []string{"repository.description", "repository.homepage", "repository.default_branch", "repository.has_issues", "repository.has_projects", "repository.has_wiki", "repository.has_discussions", "repository.web_commit_signoff_required", "repository.is_template"}},
				{"merge", []string{"repository.allow_merge_commit", "repository.allow_squash_merge", "repository.allow_rebase_merge", "repository.delete_branch_on_merge", "repository.allow_auto_merge", "repository.allow_update_branch", "repository.squash_merge_commit_title", "repository.squash_merge_commit_message", "repository.merge_commit_title", "repository.merge_commit_message"}},
				{"pr", []string{"repository.has_pull_requests", "repository.pull_request_creation_policy"}},
				{"issue", []string{"repository.issue_creation_policy"}},
				{"sponsorships", []string{"repository.has_sponsorships"}},
			}
			name := ""
			for _, group := range groups {
				if reflect.DeepEqual(paths, group.paths) {
					name = group.name
					break
				}
			}
			if name == "" {
				t.Fatalf("unexpected export group fields=%v", paths)
			}

			methods := settings.MergeMethodsState{AllowMergeCommit: true, AllowRebaseMerge: true, AllowUpdateBranch: true, SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES", MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE"}
			return settings.NewRepositoryObservation(&state, &methods, &settings.PRAccessState{PullRequestCreationPolicy: "ALL"}).WithIssueCreationPolicy(settings.IssueCreationPolicyState{Policy: "ALL"}).WithSponsorships(settings.SponsorshipsState{Enabled: false}), step(name, repo)
		},
		Topics: func(repo repository.Repository) ([]string, error) { return []string{"b", "a"}, step("topics", repo) },
		Labels: func(repo repository.Repository) (settings.LabelObservation, error) {
			if err := step("labels", repo); err != nil {
				return settings.LabelObservation{}, err
			}
			return settings.NewLabelObservation([]settings.LabelState{{ID: 1, Name: "bug", Color: "abcdef", Description: "existing label"}})
		},
	}
}
