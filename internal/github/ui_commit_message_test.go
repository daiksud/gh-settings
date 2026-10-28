package github

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/daiksud/gh-settings/internal/settings"
	"github.com/daiksud/gh-settings/internal/settings/uimodel"
)

func TestUICommitMessageMenus(t *testing.T) {
	for _, test := range []struct {
		method, choice, title, message string
	}{
		{"merge", "default_message", "MERGE_MESSAGE", "PR_TITLE"},
		{"merge", "pull_request_title", "PR_TITLE", "BLANK"},
		{"merge", "pull_request_title_and_description", "PR_TITLE", "PR_BODY"},
		{"squash", "default_message", "COMMIT_OR_PR_TITLE", "COMMIT_MESSAGES"},
		{"squash", "pull_request_title", "PR_TITLE", "BLANK"},
		{"squash", "pull_request_title_and_commit_details", "PR_TITLE", "COMMIT_MESSAGES"},
		{"squash", "pull_request_title_and_description", "PR_TITLE", "PR_BODY"},
	} {
		t.Run(test.method+"/"+test.choice, func(t *testing.T) {
			choice := uimodel.CommitMessageChoice(test.choice)
			ui := uimodel.General{}
			var observed settings.RepositoryDeclaration
			want := map[string]any{}
			if test.method == "merge" {
				ui.PullRequests.MergeCommit.DefaultCommitMessage = &choice
				observed.MergeCommitTitle, observed.MergeCommitMessage = &test.title, &test.message
				want = map[string]any{"merge_commit_title": test.title, "merge_commit_message": test.message}
			} else {
				ui.PullRequests.SquashMerge.DefaultCommitMessage = &choice
				observed.SquashMergeCommitTitle, observed.SquashMergeCommitMessage = &test.title, &test.message
				want = map[string]any{"squash_merge_commit_title": test.title, "squash_merge_commit_message": test.message}
			}
			got, err := GeneralObservation(observed)
			if err != nil || !reflect.DeepEqual(got.General, ui) {
				t.Fatalf("observation=%+v err=%v want %+v", got, err, ui)
			}
			desired, err := GeneralDeclaration(ui)
			if err != nil {
				t.Fatal(err)
			}
			client := topicsClient(t, func(req *http.Request) (*http.Response, error) {
				var got map[string]any
				if err := json.NewDecoder(req.Body).Decode(&got); err != nil {
					t.Fatal(err)
				}
				if req.Method != "PATCH" || req.URL.Path != "/repos/owner/current" || !reflect.DeepEqual(got, want) {
					t.Fatalf("payload=%v want %v", got, want)
				}
				return labelsResponse(req, 200, `{}`), nil
			})
			if err := client.ApplyRepository(repository.Repository{Owner: "owner", Name: "current"}, desired); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUICommitMessageFailures(t *testing.T) {
	for _, ui := range []uimodel.General{
		{PullRequests: uimodel.PullRequests{MergeCommit: uimodel.CommitMessage{DefaultCommitMessage: pointer(uimodel.CommitMessageChoice("PR_BODY"))}}},
		{PullRequests: uimodel.PullRequests{MergeCommit: uimodel.CommitMessage{DefaultCommitMessage: pointer(uimodel.CommitMessageChoice("pull_request_title_and_commit_details"))}}},
		{PullRequests: uimodel.PullRequests{AllowMergeCommits: pointer(false), MergeCommit: uimodel.CommitMessage{DefaultCommitMessage: pointer(uimodel.CommitMessageChoice("default_message"))}}},
		{PullRequests: uimodel.PullRequests{AllowSquashMerging: pointer(false), SquashMerge: uimodel.CommitMessage{DefaultCommitMessage: pointer(uimodel.CommitMessageChoice("default_message"))}}},
	} {
		if _, err := GeneralDeclaration(ui); err == nil {
			t.Fatal("invalid menu intent accepted")
		}
	}
	for _, state := range []settings.RepositoryDeclaration{
		{MergeCommitTitle: pointer("PR_TITLE")},
		{MergeCommitMessage: pointer("PR_BODY")},
		{MergeCommitTitle: pointer("MERGE_MESSAGE"), MergeCommitMessage: pointer("BLANK")},
		{SquashMergeCommitTitle: pointer("COMMIT_OR_PR_TITLE"), SquashMergeCommitMessage: pointer("BLANK")},
		{SquashMergeCommitTitle: pointer("UNKNOWN"), SquashMergeCommitMessage: pointer("PR_BODY")},
	} {
		if _, err := GeneralObservation(state); err == nil {
			t.Fatal("incomplete or out-of-UI pair accepted")
		}
	}
}
