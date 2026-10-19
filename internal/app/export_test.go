package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
)

func TestExportStoredSettings(t *testing.T) {
	for _, test := range []struct {
		name     string
		state    settings.RepositoryState
		want     string
		existing bool
	}{
		{"five values", settings.RepositoryState{Description: "Example repository", Homepage: "https://example.com", HasIssues: true, HasWiki: true}, "schemaVersion: 1\nsettings:\n    repository:\n        description: Example repository\n        homepage: https://example.com\n        has_issues: true\n        has_projects: false\n        has_wiki: true\n        has_discussions: false\n", false},
		{"zero values", settings.RepositoryState{}, "schemaVersion: 1\nsettings:\n    repository:\n        description: \"\"\n        homepage: \"\"\n        has_issues: false\n        has_projects: false\n        has_wiki: false\n        has_discussions: false\n", true},
		{"discussions enabled", settings.RepositoryState{HasDiscussions: true}, "schemaVersion: 1\nsettings:\n    repository:\n        description: \"\"\n        homepage: \"\"\n        has_issues: false\n        has_projects: false\n        has_wiki: false\n        has_discussions: true\n", true},
		{"projects enabled", settings.RepositoryState{Description: "Projects", HasProjects: true, HasWiki: true}, "schemaVersion: 1\nsettings:\n    repository:\n        description: Projects\n        homepage: \"\"\n        has_issues: false\n        has_projects: true\n        has_wiki: true\n        has_discussions: false\n", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := []byte("previous configuration\n") // Export does not validate an old document.
			path := appCheckout(t, string(before), true)
			if !test.existing {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(filepath.Dir(path)); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("GH_REPO", "other/repository")
			target := repository.Repository{Host: "github.com", Owner: "owner", Name: "current"}
			calls := 0
			err := Export(Reader{MergeMethods: func(repository.Repository) (settings.MergeMethodsState, error) {
				return settings.MergeMethodsState{SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES", MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE"}, nil
			}, Repository: func(repo repository.Repository) (settings.RepositoryState, error) {
				calls++
				if repo != target {
					t.Fatalf("read target=%+v, want %+v", repo, target)
				}
				if test.existing {
					got, err := os.ReadFile(path)
					if err != nil || !bytes.Equal(got, before) {
						t.Fatalf("destination changed before read: %q error=%v", got, err)
					}
				} else if _, err := os.Lstat(filepath.Dir(path)); !os.IsNotExist(err) {
					t.Fatalf("destination directory created before read: %v", err)
				}
				return test.state, nil
			}, Topics: func(repository.Repository) ([]string, error) { return []string{}, nil }, Labels: func(repository.Repository) ([]settings.LabelState, error) { return []settings.LabelState{}, nil }})
			if err != nil || calls != 1 {
				t.Fatalf("error=%v reads=%d, want one read", err, calls)
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != test.want+"        allow_merge_commit: false\n        allow_squash_merge: false\n        allow_rebase_merge: false\n        delete_branch_on_merge: false\n        allow_auto_merge: false\n        allow_update_branch: false\n        squash_merge_commit_title: COMMIT_OR_PR_TITLE\n        squash_merge_commit_message: COMMIT_MESSAGES\n        merge_commit_title: MERGE_MESSAGE\n        merge_commit_message: PR_TITLE\n    topics:\n        names: []\n    labels: []\n" {
				t.Fatalf("stored=%q error=%v, want %q", got, err, test.want)
			}
		})
	}
}

func TestExportReadFailure(t *testing.T) {
	for _, existing := range []bool{false, true} {
		name := "absent destination"
		if existing {
			name = "existing destination"
		}
		t.Run(name, func(t *testing.T) {
			before := []byte("previous configuration\n")
			path := appCheckout(t, string(before), true)
			if !existing {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(filepath.Dir(path)); err != nil {
					t.Fatal(err)
				}
			}
			checkUnchanged := func() {
				t.Helper()
				if existing {
					got, err := os.ReadFile(path)
					if err != nil || !bytes.Equal(got, before) {
						t.Fatalf("destination changed: %q error=%v", got, err)
					}
				} else if _, err := os.Lstat(filepath.Dir(path)); !os.IsNotExist(err) {
					t.Fatalf("destination directory created: %v", err)
				}
			}
			readError := errors.New("read unavailable")
			calls := 0
			err := Export(Reader{MergeMethods: func(repository.Repository) (settings.MergeMethodsState, error) {
				return settings.MergeMethodsState{SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES", MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE"}, nil
			}, Repository: func(repository.Repository) (settings.RepositoryState, error) {
				calls++
				checkUnchanged()
				return settings.RepositoryState{}, readError
			}, Topics: func(repository.Repository) ([]string, error) { return []string{}, nil }, Labels: func(repository.Repository) ([]settings.LabelState, error) { return []settings.LabelState{}, nil }})
			if !errors.Is(err, readError) || calls != 1 {
				t.Fatalf("error=%v reads=%d, want read error", err, calls)
			}
			checkUnchanged()
		})
	}
}

func TestExportTargetFailureBeforeRead(t *testing.T) {
	for _, checkout := range []bool{false, true} {
		name := "outside Git"
		if checkout {
			name = "missing origin"
		}
		t.Run(name, func(t *testing.T) {
			if checkout {
				appCheckout(t, "schemaVersion: 1\n", false)
			} else {
				t.Chdir(t.TempDir())
			}
			calls := 0
			err := Export(Reader{MergeMethods: func(repository.Repository) (settings.MergeMethodsState, error) {
				return settings.MergeMethodsState{SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES", MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE"}, nil
			}, Repository: func(repository.Repository) (settings.RepositoryState, error) {
				calls++
				return settings.RepositoryState{}, nil
			}, Topics: func(repository.Repository) ([]string, error) { return []string{}, nil }, Labels: func(repository.Repository) ([]settings.LabelState, error) { return []settings.LabelState{}, nil }})
			if err == nil || !strings.Contains(err.Error(), "resolve current repository") || calls != 0 {
				t.Fatalf("error=%v reads=%d, want target failure before read", err, calls)
			}
		})
	}
}

func TestExportStorageFailure(t *testing.T) {
	path := appCheckout(t, "old configuration\n", true)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	calls := 0
	err := Export(Reader{MergeMethods: func(repository.Repository) (settings.MergeMethodsState, error) {
		return settings.MergeMethodsState{SquashMergeCommitTitle: "COMMIT_OR_PR_TITLE", SquashMergeCommitMessage: "COMMIT_MESSAGES", MergeCommitTitle: "MERGE_MESSAGE", MergeCommitMessage: "PR_TITLE"}, nil
	}, Repository: func(repository.Repository) (settings.RepositoryState, error) {
		calls++
		return settings.RepositoryState{}, nil
	}, Topics: func(repository.Repository) ([]string, error) { return []string{}, nil }, Labels: func(repository.Repository) ([]settings.LabelState, error) { return []settings.LabelState{}, nil }})
	if err == nil || !strings.Contains(err.Error(), "must be a regular file") || calls != 1 {
		t.Fatalf("error=%v reads=%d, want storage error after read", err, calls)
	}
}
